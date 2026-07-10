#pragma once

#include "types.hpp"

#include <deque>
#include <functional>
#include <map>
#include <optional>
#include <unordered_map>
#include <vector>

namespace engine {

// A single-symbol limit order book with strict price-time priority.
//
// Structure (as in the build plan): each side is a std::map keyed by price
// level, holding a FIFO std::deque of resting orders. Bids sort descending
// (best bid = highest), asks ascending (best ask = lowest), so begin() is
// always the best level on either side.
//
// Not thread-safe by design: the intended deployment is one book per symbol,
// each owned by a single thread consuming from a Kafka partition (Phase 4).
// Serializing per symbol is how real venues get determinism without locks.
class OrderBook {
public:
    struct SubmitResult {
        SubmitStatus      status{SubmitStatus::Accepted};
        std::vector<Fill> fills;          // executions produced by this order
        Quantity          remaining{0};   // qty left after matching
        bool              resting{false}; // true if remainder was placed on the book
    };

    struct Level {
        Price    price{0};
        Quantity quantity{0}; // aggregate resting quantity at this price
    };

    struct Snapshot {
        std::vector<Level> bids; // best first
        std::vector<Level> asks; // best first
    };

    // Validates, matches against the opposite side, rests any limit remainder.
    SubmitResult submit(Order order);

    // Removes a live resting order. Returns false if the id is unknown
    // (already filled, already cancelled, or never accepted).
    bool cancel(OrderId id);

    std::optional<Price> best_bid() const;
    std::optional<Price> best_ask() const;

    // Top `depth` levels per side; depth == 0 means the full book.
    Snapshot snapshot(std::size_t depth = 0) const;

    std::size_t open_order_count() const { return index_.size(); }

private:
    using BidMap = std::map<Price, std::deque<Order>, std::greater<Price>>;
    using AskMap = std::map<Price, std::deque<Order>, std::less<Price>>;

    // Walks the opposite side while the incoming order still crosses,
    // appending fills and consuming resting orders FIFO within each level.
    template <typename OppositeMap>
    void match(Order& incoming, OppositeMap& opposite, std::vector<Fill>& fills);

    void rest(const Order& order);

    static bool crosses(const Order& incoming, Price resting_price);

    BidMap bids_;
    AskMap asks_;

    // order id -> (side, price) so cancel can find the right level in O(log n)
    // instead of scanning the whole book.
    struct Locator {
        Side  side;
        Price price;
    };
    std::unordered_map<OrderId, Locator> index_;

    Sequence next_sequence_{1};
};

} // namespace engine
