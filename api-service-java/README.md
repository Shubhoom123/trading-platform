# api-service (Phase 2)

Spring Boot 3 / Java 21 business API: authentication, accounts, order intake,
and the event-driven link to the C++ matching engine over Kafka (Phase 4;
replaced the earlier synchronous gRPC call).

## Layout

```
config/      SecurityConfig (stateless JWT), GrpcClientConfig (engine channel)
domain/      JPA entities: User, Account, OrderEntity, FillRecord + enums
repository/  Spring Data repositories
security/    JwtService (access/refresh), JwtAuthenticationFilter, principal
auth/        register / login / refresh
order/       place order + history; OrderService owns tick<->decimal + balance checks
engine/      MatchingEngineClient (publishes Order protos to Kafka) +
             OrderFillConsumer (@KafkaListener updating order/fill state)
common/      ApiException + ProblemDetail error handling
```

Protobuf message classes are generated at build time from the repo-root
`proto/trading.proto` (`protobuf-maven-plugin`) and used as the Kafka payloads —
same contract as the engine, never duplicated. Order intake persists the order
as `NEW`, publishes it to the `orders` topic, and returns immediately; the
`fills` topic asynchronously advances it to `PARTIALLY_FILLED`/`FILLED`.

## Security posture

- Passwords hashed with BCrypt; plaintext never stored, logged, or held in a field.
- Stateless JWT: short-lived access tokens, longer refresh tokens, with a `typ`
  claim so a refresh token can't satisfy a protected endpoint.
- HS256 secret must be >= 32 bytes and comes from `JWT_SECRET` — the service
  fails fast on a weak key. No secrets in source (see repo-root `.env.example`).
- Bean Validation on every request body; input re-validated at the engine boundary.
- Login returns the same error for unknown-email and wrong-password (no user enumeration).
- Schema owned by Flyway migrations; Hibernate is `validate`-only.

## Run

See the repo-root README "Quick start (Phase 2)". Needs JDK 21, Postgres, and
the engine gRPC server running. `mvn verify` runs the Testcontainers
integration test against a real Postgres.

Fills are persisted (Phase 5): `OrderFillConsumer` writes a `fills` row and
advances order state, deduped on the engine's monotonic sequence so at-least-once
Kafka redelivery is a no-op.

## Known gaps (tracked)

- TLS/mTLS between services (Phase 6). The engine channel is plaintext locally.
- Market-buy balance reservation needs a price reference.
- Price-improvement refund: buys reserve at the limit price; the difference when
  filled below it isn't refunded yet (needs the fill price reconciliation loop).
- Sell-side position ledger (tracking held shares) is still out of scope.
