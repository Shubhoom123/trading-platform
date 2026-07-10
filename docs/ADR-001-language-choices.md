# ADR-001: Language choice per component

Status: accepted · Date: 2026-07-09

## Context

Each service should use a language with a one-sentence defensible reason, or
the component gets cut. This records those reasons and what I'd reconsider.

## Decisions

**C++ for the matching engine.** Matching is the latency-critical hot path:
it needs allocation-light, cache-friendly data structures and deterministic
execution, which is why real exchanges (NASDAQ, LMAX, Cboe) build their
matchers this way. The engine is a pure library with no I/O in the hot path;
transport wraps it — a gRPC server for reads/tests and, as of Phase 4, a
Kafka bridge (librdkafka) that consumes the `orders` topic and produces to
`fills`, both sharing one book registry.

**Java + Spring Boot for the business API.** Auth, validation, persistence,
and orchestration are classic enterprise-backend concerns, and Spring
Security / Data JPA / Flyway is the stack most teams actually run — the goal
here is demonstrating fluency in the dominant ecosystem, not novelty.

**Go for the gateway.** Fanning out live fills to thousands of WebSocket
clients is an I/O-concurrency problem, and goroutines + channels are the
standard, cheap answer. Doing this in Java or C++ would be more code for a
worse fit.

**Kafka between services.** Turns three services calling each other into an
event-driven system: replayable order flow, natural sharding by symbol via
partitions, and no synchronous coupling between the API and the engine.

## What I'd reconsider with more time

- Rust instead of C++ for the engine — comparable performance with memory
  safety, at the cost of a less battle-tested exchange lineage to cite.
- Redpanda over Kafka locally for a lighter footprint (wire-compatible).
- gRPC end-to-end instead of Kafka if strict request/response semantics
  mattered more than replayability.
