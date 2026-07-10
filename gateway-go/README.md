# gateway-go (Phase 3)

The Go edge service. Its job is the one thing goroutines are the obvious tool
for: fanning out live fills to many concurrent WebSocket clients cheaply, plus
lightweight REST reads, JWT validation, and rate limiting in front of the other
services.

## Layout

```
cmd/gateway/         main: config, wiring, graceful shutdown
internal/config/     env-driven config (JWT secret required, no defaults)
internal/auth/       JWT access-token verifier (shares the Java HS256 secret)
internal/middleware/ request logging, token-bucket rate limiter, JWT auth
internal/hub/        goroutine/channel fan-out — slow clients dropped, not blocking
internal/engine/     gRPC client to the C++ matching engine (book snapshots)
internal/kafka/      fills-topic consumer -> hub (Phase 4; replaced the gRPC pump)
internal/cache/      Redis read-through cache for book/quote reads (Phase 5)
internal/server/     gin router, WebSocket handler, REST handlers
internal/pb/         generated protobuf/gRPC stubs (make proto)
loadtest/            k6 WebSocket load test
```

## Endpoints

| Method | Path | Auth | Notes |
| --- | --- | --- | --- |
| GET | `/healthz` | none | liveness |
| GET | `/metrics` | none | Prometheus metrics (client_golang) |
| GET | `/ws?symbol=X&token=JWT` | access token | live fill stream |
| GET | `/api/book/:symbol?depth=N` | Bearer | order book snapshot (Redis cache → engine on miss) |
| GET | `/api/quote/:symbol` | Bearer | best bid/ask, served from the same cache |
| GET | `/api/orders` | Bearer | proxied to the Java API |
| GET | `/api/account` | Bearer | proxied to the Java API |

## Design notes

- **Slow consumers never block the stream.** Each WebSocket client has its own
  buffered channel; the hub uses non-blocking sends and drops (and counts)
  messages for a client that falls behind, so one slow browser can't back up
  matching-data delivery for everyone.
- **Fills come from Kafka (Phase 4).** A single consumer-group reader drains the
  `fills` topic and broadcasts to the hub by symbol. This replaced the Phase 3
  per-symbol `StreamFills` gRPC pump — and because the hub and WebSocket layer
  only ever saw a stream of per-symbol messages, nothing above the consumer
  changed. gRPC is still used for on-demand book snapshots.
- **Read-through book cache (Phase 5).** Book/quote reads try Redis first and
  fall back to a gRPC snapshot on a miss, caching the result under a short TTL
  (default 750ms). The cache is best-effort: a Redis outage degrades reads to
  the engine path, never fails them. Set `REDIS_ADDR` to enable; leave it empty
  and the gateway uses an always-miss no-op store.
- **Verifier, not issuer.** The gateway validates the same HS256 JWTs the Java
  API mints but never issues them. It enforces `alg=HS256` (no `none` downgrade)
  and rejects refresh tokens presented as access tokens.

## Build & run

```sh
make tools   # once: installs protoc-gen-go{,-grpc} (needs protoc on PATH)
make proto   # generate internal/pb from ../proto/trading.proto
export JWT_SECRET=<same 32+ byte secret as the Java API>
export REDIS_ADDR=localhost:6379   # optional; omit to disable the book cache
make run     # serves :8090, dials the engine at localhost:50051
```

## Load test

```sh
k6 run -e TOKEN=<access-jwt> -e GATEWAY=ws://localhost:8090 loadtest/ws_load.js
```

Ramps to 500 concurrent WebSocket clients and counts frames received.
For REST throughput, `vegeta` works too:

```sh
echo "GET http://localhost:8090/api/book/AAPL" | \
  vegeta attack -header "Authorization: Bearer <jwt>" -rate=500 -duration=30s | vegeta report
```
