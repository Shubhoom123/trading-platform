// Zero-dependency correctness harness for the matching engine.
//
// Unlike order_book_test.cpp (which uses GoogleTest, fetched at configure time),
// this file needs nothing but a C++20 compiler, so anyone can verify the core
// in one line without a network or a test framework:
//
//   g++ -std=c++20 -O2 -Isrc tests/standalone_check.cpp src/order_book.cpp -o check && ./check
//
// It is also built as the `engine_check` CMake target and run in CI. Assertions
// are always on (NDEBUG is intentionally not set here).

#include "order_book.hpp"

#include <cstdio>
#include <cstdlib>

using namespace engine;

namespace {

int g_checks = 0;
int g_failures = 0;

void check(bool cond, const char* what) {
    if (cond) {
        ++g_checks;
    } else {
        ++g_failures;
        std::printf("FAIL: %s\n", what);
    }
}

Order limit(OrderId id, Side s, Price p, Quantity q) {
    return Order{id, s, OrderType::Limit, p, q, 0};
}
Order market(OrderId id, Side s, Quantity q) {
    return Order{id, s, OrderType::Market, 0, q, 0};
}

} // namespace

int main() {
    { // A limit that doesn't cross rests on the book.
        OrderBook b;
        auto r = b.submit(limit(1, Side::Buy, 100, 10));
        check(r.status == SubmitStatus::Accepted && r.resting && r.fills.empty(),
              "non-crossing limit rests");
        check(b.best_bid().value() == 100, "best bid set");
    }
    { // A crossing order trades at the resting (maker) price.
        OrderBook b;
        b.submit(limit(1, Side::Sell, 100, 10));
        auto r = b.submit(limit(2, Side::Buy, 105, 10));
        check(r.fills.size() == 1 && r.fills[0].price == 100 && r.fills[0].quantity == 10,
              "fill at maker price");
        check(!r.resting && r.remaining == 0, "aggressive order fully filled");
        check(!b.best_ask().has_value(), "ask level emptied");
    }
    { // Partial fill leaves a resting remainder.
        OrderBook b;
        b.submit(limit(1, Side::Sell, 100, 4));
        auto r = b.submit(limit(2, Side::Buy, 100, 10));
        check(r.fills.size() == 1 && r.fills[0].quantity == 4, "partial fill quantity");
        check(r.remaining == 6 && r.resting, "remainder rests");
    }
    { // Price-time priority: oldest order at a level fills first.
        OrderBook b;
        b.submit(limit(1, Side::Sell, 100, 5));
        b.submit(limit(2, Side::Sell, 100, 5));
        auto r = b.submit(limit(3, Side::Buy, 100, 5));
        check(r.fills.size() == 1 && r.fills[0].maker_order_id == 1, "FIFO within a level");
    }
    { // Market order sweeps best price first; unfilled remainder is discarded.
        OrderBook b;
        b.submit(limit(1, Side::Sell, 101, 5));
        b.submit(limit(2, Side::Sell, 100, 5));
        auto r = b.submit(market(3, Side::Buy, 12));
        check(r.fills.size() == 2, "market sweeps two levels");
        check(r.fills[0].price == 100 && r.fills[1].price == 101, "best ask first");
        check(r.remaining == 2 && !r.resting, "market remainder discarded, never rests");
    }
    { // Cancel removes a resting order; cancelling an unknown id is a no-op.
        OrderBook b;
        b.submit(limit(1, Side::Buy, 100, 10));
        check(b.cancel(1) && !b.best_bid().has_value(), "cancel removes resting order");
        check(!b.cancel(999), "cancel of unknown id returns false");
    }
    { // The engine validates and rejects at its own boundary.
        OrderBook b;
        check(b.submit(limit(0, Side::Buy, 100, 10)).status == SubmitStatus::RejectedBadOrder,
              "reject zero id");
        check(b.submit(limit(1, Side::Buy, 100, 0)).status == SubmitStatus::RejectedBadOrder,
              "reject zero quantity");
        check(b.submit(limit(2, Side::Buy, 0, 10)).status == SubmitStatus::RejectedBadOrder,
              "reject non-positive limit price");
        b.submit(limit(3, Side::Buy, 100, 10));
        check(b.submit(limit(3, Side::Buy, 100, 5)).status == SubmitStatus::RejectedDuplicate,
              "reject duplicate live id");
    }
    { // Snapshot aggregates quantity per level and honours depth.
        OrderBook b;
        b.submit(limit(1, Side::Buy, 100, 5));
        b.submit(limit(2, Side::Buy, 100, 5));
        b.submit(limit(3, Side::Buy, 99, 7));
        auto top = b.snapshot(1);
        check(top.bids.size() == 1 && top.bids[0].price == 100 && top.bids[0].quantity == 10,
              "snapshot aggregates and respects depth");
        check(b.snapshot(0).bids.size() == 2, "depth 0 returns full book");
    }

    std::printf("%d checks passed, %d failed\n", g_checks, g_failures);
    return g_failures == 0 ? EXIT_SUCCESS : EXIT_FAILURE;
}
