# Polyglot Trading & Risk Platform

An event-driven trading platform where each service uses the language the
problem actually calls for:

- **C++20 matching engine** — latency-critical order matching with strict
  price-time priority, the way real venues (NASDAQ, LMAX) are built.
- **Java 21 / Spring Boot API** — accounts, auth (JWT), order validation,
  persistence; publishes orders to Kafka and consumes fills. *(Phase 2–4)*
- **Go gateway** — WebSocket fan-out of live fills, REST reads, rate limiting,
  the single client front door; consumes fills from Kafka. *(Phase 3–4)*
- **React + TypeScript frontend** — a trading terminal (order book, order entry,
  live fills) talking to the gateway. See [`frontend-web/`](frontend-web/README.md).
- **Kafka** (Redpanda locally) between services, **Postgres** as the system of
  record, **Redis** for hot book reads, **Prometheus + Grafana** for
  observability — all wired, with Dockerfiles, a full compose stack, and k8s
  manifests. *(Phases 4–6)*

Data flow (event-driven as of Phase 4):
`Client -> Java API -> Kafka(orders) -> C++ engine -> Kafka(fills) -> {Java API updates DB, Go gateway -> Client}`

## Status

| Phase | Component | State |
| --- | --- | --- |
| 1 | C++ matching engine (core + tests + bench + gRPC server) | ✅ done |
| 2 | Java Spring Boot API (auth, JPA, order intake) | 🟡 scaffolded — needs Postgres to run |
| 3 | Go gateway (WebSocket fan-out, REST reads, JWT, rate limiting) | 🟡 scaffolded |
| 4 | Kafka wiring (engine consumes orders/produces fills; API publishes + consumes; gateway consumes) | 🟡 wired — needs the compose stack to run end-to-end |
| 5 | Postgres (users/accounts/orders/fills) + Redis book cache | 🟡 wired — fills persisted idempotently; gateway reads cached via Redis |
| 6 | Observability (Prometheus/Grafana) + Docker + K8s | 🟡 wired — /metrics on all three, dashboards, per-service Dockerfiles, compose + k8s manifests |

## Quick start (Phase 1)

```sh
cd matching-engine-cpp
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j
ctest --test-dir build --output-on-failure   # 14 tests
./build/engine_bench                          # throughput + latency percentiles

# gRPC server (requires gRPC + protobuf installed; core lib/tests build without them)
ENGINE_GRPC_ADDR=0.0.0.0:50051 ./build/matching_engine_server
```

## Quick start (Phase 2 — Java API)

Needs JDK 21, a Postgres, and the engine server above running.

```sh
cd api-service-java
export DB_URL=jdbc:postgresql://localhost:5432/trading DB_USER=trading DB_PASSWORD=... \
       JWT_SECRET=$(openssl rand -hex 32) ENGINE_GRPC_TARGET=localhost:50051
mvn spring-boot:run             # generates gRPC stubs from ../proto, runs Flyway, serves :8080
mvn verify                      # unit + Testcontainers integration tests
```

Endpoints: `POST /api/auth/{register,login,refresh}`, `POST /api/orders`,
`GET /api/orders`, `GET /actuator/prometheus`.

## Quick start (Phase 3 — Go gateway)

Needs Go 1.22, protoc, the engine server, and the Java API running.

```sh
cd gateway-go
make tools proto                # generate gRPC stubs from ../proto
export JWT_SECRET=<same secret as the Java API>   # gateway verifies, never issues
make run                        # serves :8090
make test                       # go test -race ./...
```

Endpoints: `GET /ws?symbol=X&token=JWT` (live fills), `GET /api/book/:symbol`,
and `GET /api/{orders,account}` (proxied to the Java API). See
[gateway-go/README.md](gateway-go/README.md).

## Quick start (Phase 4 — event-driven, full stack)

Bring up Kafka (Redpanda) + Postgres, then run the three services against them.
The engine consumes `orders` and produces `fills`; the API publishes `orders`
and consumes `fills`; the gateway consumes `fills` for its WebSocket stream.

```sh
docker compose -f infra/docker-compose.yml up -d      # redpanda + postgres + topics
export KAFKA_BOOTSTRAP_SERVERS=localhost:19092 JWT_SECRET=$(openssl rand -hex 32)

# engine: gRPC reads + Kafka bridge (needs librdkafka at build time)
ENGINE_GRPC_ADDR=0.0.0.0:50051 ./matching-engine-cpp/build/matching_engine_server &

# api: publishes orders, consumes fills, serves REST/auth
(cd api-service-java && DB_URL=jdbc:postgresql://localhost:5432/trading \
   DB_USER=trading DB_PASSWORD=trading mvn spring-boot:run) &

# gateway: consumes fills, serves WebSocket + REST
(cd gateway-go && make run) &
```

Place an order via the API and watch the fill arrive on the gateway WebSocket —
that round trip is the event-driven loop, no service calling another directly.

## Quick start (Phase 6 — whole stack in one command)

Everything containerized: the three services (built from their multi-stage
Dockerfiles) plus Redpanda, Postgres, Redis, Prometheus, and Grafana.

```sh
export JWT_SECRET=$(openssl rand -hex 32)
docker compose -f infra/docker-compose.yml up --build -d
```

- API: http://localhost:8080  ·  Gateway: http://localhost:8090
- Prometheus: http://localhost:9090  ·  Grafana: http://localhost:3000 (anon access on)

Grafana auto-loads the **Trading Platform** dashboard (orders/sec, match-latency
p50/p99, active WebSocket connections, cache hit ratio). Kubernetes manifests for
the same stack live in [`infra/k8s/`](infra/k8s/README.md) (k3d/minikube).

### Metrics endpoints

| Service | Endpoint | Source |
| --- | --- | --- |
| Matching engine | `:9101/metrics` | prometheus-cpp (orders/fills counters, match-latency histogram) |
| API | `:8080/actuator/prometheus` | Micrometer (orders published, fills consumed) |
| Gateway | `:8090/metrics` | client_golang (active WS, fills broadcast, cache hits/misses) |

## Web frontend

A React + TypeScript (Vite) trading terminal in [`frontend-web/`](frontend-web/README.md):
register/login, live order book, order placement, order history, and a live
fills feed — all talking to the Go gateway as the single backend (the gateway
proxies auth/orders to the Java API and serves book/quote/WebSocket itself).

```sh
# with the stack up (compose or host), run the dev server:
cd frontend-web && npm install && npm run dev      # http://localhost:5173

# or serve it from the compose stack (opt-in profile):
docker compose -f infra/docker-compose.yml --profile ui up --build -d   # http://localhost:3001
```

## Testing

Every service has a unit suite, plus a cross-service end-to-end test. CI runs
them as separate jobs so failures are attributable.

| Layer | What's covered | How to run |
| --- | --- | --- |
| C++ engine (no deps) | Matching, price-time priority, market sweeps, cancel, validation, snapshots — pure `g++`, no framework | `g++ -std=c++20 -O2 -Isrc tests/standalone_check.cpp src/order_book.cpp -o check && ./check` |
| C++ engine (GoogleTest) | The above + monotonic fill sequences + a fuzzed "book never crosses" property | `ctest --test-dir build` |
| Java API | JwtService (access/refresh/expiry/wrong-secret), OrderService (tick conversion, balance, market-buy reject) + a Testcontainers lifecycle IT | `mvn verify` |
| Go gateway | Hub fan-out (incl. slow-consumer drop), JWT verifier, Redis cache, rate limiter, config parsing | `make test` (`go test -race ./...`) |
| End-to-end | Place an order → assert the fill lands on the gateway WebSocket **and** persists as FILLED | `JWT_SECRET=$(openssl rand -hex 32) infra/e2e/run_e2e.sh` |

The C++ core is verified here for real: the standalone check passes all cases,
and a 20k-order fuzz run produced **0 crossed-book violations** with strictly
monotonic fill sequences.

## Benchmark (Release, single thread)

Measured over 1M randomized limit orders against a warmed book
(fixed RNG seed, so runs are comparable across commits):

| Metric | Value |
| --- | --- |
| Throughput | ~6.5M orders/sec |
| Latency p50 | ~84 ns |
| Latency p99 | ~375 ns |
| Latency p99.9 | ~1.0 µs |

*(Measured with `g++ -O3` over 1M orders on a shared cloud container — expect
different absolute values on your hardware; the point is having a measured
baseline, not the raw figure. Reproduce: `./build/engine_bench`.)*

## Repository layout

```
matching-engine-cpp/   C++ engine: src/, tests/, bench/, CMakeLists.txt
api-service-java/      Spring Boot service (Phase 2)
gateway-go/            Go gateway (Phase 3)
proto/                 shared .proto contracts — single source of truth
infra/                 docker-compose, k8s, prometheus/grafana (Phases 4-6)
docs/                  architecture notes and ADRs
.github/workflows/     CI, one job per service
```

## Design decisions

See [docs/ADR-001-language-choices.md](docs/ADR-001-language-choices.md).
Key ones baked into Phase 1:

- **Integer tick prices, never floats.** Floating-point money is a
  correctness bug; the API layer owns decimal conversion.
- **Trades execute at the maker's price.** The resting order set the terms;
  aggressive takers get price improvement.
- **Deterministic sequence numbers, not wall clocks,** drive time priority —
  replayable and testable.
- **One book per symbol, single-threaded by design.** Concurrency comes from
  sharding symbols across Kafka partitions (Phase 4), not from locks inside
  the hot path.
- **Validation at every boundary.** The engine rejects bad and duplicate
  orders itself instead of trusting the Java layer.
