#pragma once

#include "book_registry.hpp"
#include "metrics.hpp"

#include <atomic>
#include <string>

namespace engine {

struct KafkaConfig {
    std::string brokers;       // e.g. localhost:9092
    std::string orders_topic;  // consumed
    std::string fills_topic;   // produced
    std::string group_id;      // consumer group
};

// The Phase 4 order flow: consume Order protobufs from the orders topic, match
// them against the shared BookRegistry, and produce the resulting Fill
// protobufs to the fills topic. Keyed by symbol on both sides so a symbol's
// order flow stays ordered within its partition.
//
// At-least-once delivery: offsets are committed only after the fills for a
// message have been handed to the producer. The engine's duplicate-id guard
// makes a replayed order a no-op, so redelivery is safe.
class KafkaBridge {
public:
    KafkaBridge(BookRegistry& registry, KafkaConfig cfg, Metrics& metrics);
    ~KafkaBridge();

    // Blocks, consuming until stop() is called (typically from a signal handler).
    void run();
    void stop();

private:
    BookRegistry&     registry_;
    KafkaConfig       cfg_;
    Metrics&          metrics_;
    std::atomic<bool> running_{false};
};

} // namespace engine
