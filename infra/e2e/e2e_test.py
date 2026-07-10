#!/usr/bin/env python3
"""End-to-end test of the whole event-driven loop.

Standard-library only (no pip install): HTTP via urllib, a minimal hand-rolled
WebSocket client. Assumes the compose stack is already up (see run_e2e.sh). It:
  1. registers a user and logs in (JWT from the Java API),
  2. funds the account directly in Postgres (there's no funding endpoint),
  3. opens a gateway WebSocket subscribed to a symbol,
  4. places a SELL then a crossing BUY via the API,
  5. asserts a fill arrives on the WebSocket (Kafka -> gateway path), and
  6. asserts the order reaches FILLED in the API (Kafka -> API -> Postgres path).

No service is called directly by another anywhere in this flow — it all travels
over Kafka, which is the point of the architecture.
"""

import base64
import json
import os
import socket
import struct
import sys
import time
import urllib.request
import uuid

API = os.environ.get("API_BASE_URL", "http://localhost:8080")
GW_HOST = os.environ.get("GATEWAY_HOST", "localhost")
GW_PORT = int(os.environ.get("GATEWAY_PORT", "8090"))
COMPOSE = os.environ.get("COMPOSE", "docker compose -f infra/docker-compose.yml")
SYMBOL = "E2E"
DEADLINE_S = 60


def log(msg):
    print(f"[e2e] {msg}", flush=True)


# --- HTTP (stdlib) ----------------------------------------------------------

def http_post(path, body, token=None):
    req = urllib.request.Request(API + path, data=json.dumps(body).encode(),
                                 method="POST")
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


def http_get(path, token):
    req = urllib.request.Request(API + path, method="GET")
    req.add_header("Authorization", f"Bearer {token}")
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


# --- Minimal WebSocket client (stdlib) --------------------------------------

class WS:
    def __init__(self, host, port, path):
        self.sock = socket.create_connection((host, port), timeout=10)
        key = base64.b64encode(os.urandom(16)).decode()
        handshake = (
            f"GET {path} HTTP/1.1\r\n"
            f"Host: {host}:{port}\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            f"Sec-WebSocket-Key: {key}\r\n"
            "Sec-WebSocket-Version: 13\r\n\r\n"
        )
        self.sock.sendall(handshake.encode())
        self.buf = b""
        while b"\r\n\r\n" not in self.buf:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise ConnectionError("websocket handshake failed")
            self.buf += chunk
        status_line = self.buf.split(b"\r\n", 1)[0]
        if b" 101 " not in status_line:
            raise ConnectionError(f"websocket handshake: {status_line!r}")
        self.buf = self.buf.split(b"\r\n\r\n", 1)[1]  # keep any buffered frame bytes

    def _read(self, n):
        while len(self.buf) < n:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise ConnectionError("websocket closed")
            self.buf += chunk
        data, self.buf = self.buf[:n], self.buf[n:]
        return data

    def recv(self):
        """Returns (opcode, payload_bytes). Server frames are never masked."""
        b0, b1 = self._read(2)
        opcode = b0 & 0x0F
        length = b1 & 0x7F
        if length == 126:
            length = struct.unpack(">H", self._read(2))[0]
        elif length == 127:
            length = struct.unpack(">Q", self._read(8))[0]
        mask = self._read(4) if (b1 & 0x80) else b""
        payload = self._read(length)
        if mask:
            payload = bytes(payload[i] ^ mask[i % 4] for i in range(length))
        return opcode, payload

    def pong(self, payload=b""):
        # Client -> server frames must be masked.
        mask = os.urandom(4)
        masked = bytes(payload[i] ^ mask[i % 4] for i in range(len(payload)))
        header = bytes([0x8A, 0x80 | len(payload)]) + mask  # 0x8A = FIN | pong
        self.sock.sendall(header + masked)

    def close(self):
        try:
            self.sock.close()
        except OSError:
            pass


# --- Test steps -------------------------------------------------------------

def fund_all_accounts():
    # os.system (not subprocess) so we don't depend on a possibly-broken
    # subprocess module in some Python builds.
    sql = "UPDATE accounts SET balance_ticks = 1000000000;"
    cmd = (f'{COMPOSE} exec -T postgres '
           f'psql -U trading -d trading -c "{sql}" >/dev/null 2>&1')
    if os.system(cmd) != 0:
        raise RuntimeError("failed to fund accounts via psql")


def register_and_login():
    email = f"e2e-{uuid.uuid4().hex[:8]}@example.com"
    resp = http_post("/api/auth/register",
                     {"email": email, "password": "correct horse battery staple"})
    return resp["accessToken"]


def place(token, side):
    return http_post("/api/orders",
                     {"symbol": SYMBOL, "side": side, "type": "LIMIT",
                      "price": 100.00, "quantity": 5},
                     token=token)


def main():
    token = register_and_login()
    log("registered + logged in")
    fund_all_accounts()
    log("funded account")

    ws = WS(GW_HOST, GW_PORT, f"/ws?symbol={SYMBOL}&token={token}")
    log("websocket connected")
    time.sleep(0.5)  # let the hub subscription register before we trade

    place(token, "SELL")          # rests on the book
    taker = place(token, "BUY")   # crosses -> produces a fill
    log(f"placed SELL + BUY (taker order id={taker['id']})")

    # 1) A fill must arrive on the WebSocket.
    got_fill = False
    end = time.time() + DEADLINE_S
    while time.time() < end:
        self_timeout = max(1.0, end - time.time())
        ws.sock.settimeout(self_timeout)
        try:
            opcode, payload = ws.recv()
        except (socket.timeout, ConnectionError):
            break
        if opcode == 0x9:            # ping -> pong, keep the stream alive
            ws.pong(payload)
            continue
        if opcode == 0x8:            # close
            break
        if opcode == 0x1:            # text
            msg = json.loads(payload.decode())
            if msg.get("type") == "fill" and msg.get("symbol") == SYMBOL:
                log(f"received fill on websocket: {msg}")
                got_fill = True
                break
    ws.close()
    if not got_fill:
        log("FAIL: no fill received on the websocket")
        return 1

    # 2) The order must reach FILLED in the API (Postgres system of record).
    filled = False
    while time.time() < end:
        statuses = [o["status"] for o in http_get("/api/orders", token)]
        if "FILLED" in statuses:
            log(f"order statuses in DB: {statuses}")
            filled = True
            break
        time.sleep(1)
    if not filled:
        log("FAIL: order never reached FILLED in the API")
        return 1

    log("PASS: fill delivered over websocket AND persisted as FILLED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
