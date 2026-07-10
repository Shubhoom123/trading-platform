import { useEffect, useRef, useState } from "react";
import { tokens, WS_BASE } from "../api/client";
import type { Fill } from "../api/types";

const MAX_FILLS = 50;

type Status = "connecting" | "open" | "closed";

// Subscribes to the gateway's live fill stream for a symbol over a WebSocket,
// keeping the most recent fills. Reconnects with a short backoff if the socket
// drops. The token goes in the query string because browsers can't set headers
// on a WebSocket upgrade.
export function useFills(symbol: string) {
  const [fills, setFills] = useState<Fill[]>([]);
  const [status, setStatus] = useState<Status>("connecting");
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    setFills([]);
    let closedByUs = false;
    let reconnect: number;

    const connect = () => {
      const token = tokens.access();
      if (!token) return;
      setStatus("connecting");
      const ws = new WebSocket(
        `${WS_BASE}/ws?symbol=${encodeURIComponent(symbol)}&token=${encodeURIComponent(token)}`,
      );
      wsRef.current = ws;

      ws.onopen = () => setStatus("open");
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data) as Fill;
          if (msg.type === "fill") {
            setFills((prev) => [msg, ...prev].slice(0, MAX_FILLS));
          }
        } catch {
          /* ignore malformed frames */
        }
      };
      ws.onclose = () => {
        setStatus("closed");
        if (!closedByUs) reconnect = window.setTimeout(connect, 1500);
      };
      ws.onerror = () => ws.close();
    };

    connect();

    return () => {
      closedByUs = true;
      window.clearTimeout(reconnect);
      wsRef.current?.close();
    };
  }, [symbol]);

  return { fills, status };
}
