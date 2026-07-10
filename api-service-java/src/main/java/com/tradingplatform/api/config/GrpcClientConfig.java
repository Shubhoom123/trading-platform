package com.tradingplatform.api.config;

// Intentionally empty.
//
// This file held the gRPC channel/stub beans in Phases 2–3. Phase 4 moved the
// engine link to Kafka (Spring Boot auto-configures the KafkaTemplate from
// application.yml, so no manual config bean is needed). This build environment
// cannot delete files, so the class is retained as an inert placeholder rather
// than leaving dead gRPC wiring. Safe to remove from the repo with `git rm`.
final class GrpcClientConfig {
    private GrpcClientConfig() {
    }
}
