#pragma once

#include "order_book.hpp"
#include "trading.grpc.pb.h"

#include <chrono>
#include <condition_variable>
#include <deque>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

namespace engine {

// Fan-out of fills to gRPC StreamFills subscribers, keyed by symbol.
// A slow subscriber never blocks a matching thread (see publish()).
class FillBroadcaster {
public:
    enum class PopResult { Item, Timeout, Closed };

    class Subscription {
    public:
        PopResult wait_and_pop(trading::v1::Fill& out,
                               std::chrono::milliseconds timeout);

    private:
        friend class FillBroadcaster;
        std::mutex                    mu_;
        std::condition_variable       cv_;
        std::deque<trading::v1::Fill> queue_;
        bool                          closed_{false};
    };

    std::shared_ptr<Subscription> subscribe(const std::string& symbol);
    void unsubscribe(const std::string& symbol,
                     const std::shared_ptr<Subscription>& sub);
    void publish(const std::string& symbol, const trading::v1::Fill& fill);

private:
    std::mutex mu_;
    std::unordered_map<std::string,
                       std::vector<std::shared_ptr<Subscription>>> subs_;
};

// Shared, thread-safe home for one order book per symbol.
//
// Both front doors — the gRPC service (reads + legacy direct submit) and the
// Kafka bridge (the Phase 4 order flow) — drive the SAME registry, so a book
// snapshot served over gRPC always reflects orders matched from Kafka. The
// engine itself is single-threaded per book, so a single mutex serializes all
// mutating access; Phase 4's partition-per-symbol model is the path to removing
// even that lock later.
class BookRegistry {
public:
    struct SubmitOutcome {
        SubmitStatus                   status{SubmitStatus::Accepted};
        std::vector<trading::v1::Fill> fills;          // proto, symbol populated
        Quantity                       remaining{0};
        bool                           resting{false};
    };

    // Validates, matches, rests any remainder, and fans fills out to any gRPC
    // stream subscribers. Returns the fills so the caller (e.g. the Kafka
    // bridge) can also forward them onward.
    SubmitOutcome submit(const std::string& symbol, Order order);

    bool cancel(const std::string& symbol, OrderId id);

    OrderBook::Snapshot snapshot(const std::string& symbol, std::size_t depth);

    FillBroadcaster& broadcaster() { return broadcaster_; }

private:
    OrderBook& book_for(const std::string& symbol); // caller holds mu_

    std::mutex                                 mu_;
    std::unordered_map<std::string, OrderBook> books_;
    FillBroadcaster                            broadcaster_;
};

} // namespace engine
