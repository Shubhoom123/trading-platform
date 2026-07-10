#include "order_book.hpp"

#include <gtest/gtest.h>

#include <random>

using namespace engine;

namespace {

Order limit(OrderId id, Side side, Price price, Quantity qty) {
    return Order{.id = id, .side = side, .type = OrderType::Limit,
                 .price = price, .quantity = qty};
}

Order market(OrderId id, Side side, Quantity qty) {
    return Order{.id = id, .side = side, .type = OrderType::Market,
                 .price = 0, .quantity = qty};
}

} // namespace

TEST(OrderBook, NonCrossingLimitRestsOnBook) {
    OrderBook book;
    auto res = book.submit(limit(1, Side::Buy, 100, 10));

    EXPECT_EQ(res.status, SubmitStatus::Accepted);
    EXPECT_TRUE(res.fills.empty());
    EXPECT_TRUE(res.resting);
    EXPECT_EQ(res.remaining, 10u);
    EXPECT_EQ(book.best_bid(), Price{100});
    EXPECT_EQ(book.open_order_count(), 1u);
}

TEST(OrderBook, ExactFullFill) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 105, 10));
    auto res = book.submit(limit(2, Side::Buy, 105, 10));

    ASSERT_EQ(res.fills.size(), 1u);
    EXPECT_EQ(res.fills[0].maker_order_id, 1u);
    EXPECT_EQ(res.fills[0].taker_order_id, 2u);
    EXPECT_EQ(res.fills[0].price, 105);
    EXPECT_EQ(res.fills[0].quantity, 10u);
    EXPECT_EQ(res.remaining, 0u);
    EXPECT_FALSE(res.resting);
    EXPECT_FALSE(book.best_ask().has_value());
    EXPECT_EQ(book.open_order_count(), 0u);
}

TEST(OrderBook, PartialFillRestsRemainder) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 105, 4));
    auto res = book.submit(limit(2, Side::Buy, 105, 10));

    ASSERT_EQ(res.fills.size(), 1u);
    EXPECT_EQ(res.fills[0].quantity, 4u);
    EXPECT_EQ(res.remaining, 6u);
    EXPECT_TRUE(res.resting);
    EXPECT_EQ(book.best_bid(), Price{105}); // remainder now rests as a bid
    EXPECT_FALSE(book.best_ask().has_value());
}

TEST(OrderBook, PartiallyFilledMakerKeepsPriority) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 105, 10));
    book.submit(limit(2, Side::Sell, 105, 10));
    book.submit(market(3, Side::Buy, 4)); // bites 4 off order 1

    auto res = book.submit(market(4, Side::Buy, 10));
    ASSERT_EQ(res.fills.size(), 2u);
    EXPECT_EQ(res.fills[0].maker_order_id, 1u); // order 1 still first with its 6
    EXPECT_EQ(res.fills[0].quantity, 6u);
    EXPECT_EQ(res.fills[1].maker_order_id, 2u);
    EXPECT_EQ(res.fills[1].quantity, 4u);
}

TEST(OrderBook, TimePriorityFifoWithinLevel) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 105, 5));
    book.submit(limit(2, Side::Sell, 105, 5)); // same price, arrived later
    auto res = book.submit(limit(3, Side::Buy, 105, 5));

    ASSERT_EQ(res.fills.size(), 1u);
    EXPECT_EQ(res.fills[0].maker_order_id, 1u); // earlier order fills first
}

TEST(OrderBook, PricePriorityAcrossLevels) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 106, 5));
    book.submit(limit(2, Side::Sell, 104, 5)); // better ask, arrived later
    auto res = book.submit(limit(3, Side::Buy, 106, 5));

    ASSERT_EQ(res.fills.size(), 1u);
    EXPECT_EQ(res.fills[0].maker_order_id, 2u);
    EXPECT_EQ(res.fills[0].price, 104); // taker gets price improvement
}

TEST(OrderBook, SweepsMultipleLevels) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 104, 3));
    book.submit(limit(2, Side::Sell, 105, 3));
    book.submit(limit(3, Side::Sell, 106, 3));
    auto res = book.submit(limit(4, Side::Buy, 105, 10));

    ASSERT_EQ(res.fills.size(), 2u); // 104 and 105 cross; 106 does not
    EXPECT_EQ(res.fills[0].price, 104);
    EXPECT_EQ(res.fills[1].price, 105);
    EXPECT_EQ(res.remaining, 4u);
    EXPECT_TRUE(res.resting);
    EXPECT_EQ(book.best_bid(), Price{105});
    EXPECT_EQ(book.best_ask(), Price{106});
}

TEST(OrderBook, TradesExecuteAtMakerPrice) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 100, 5));
    auto res = book.submit(limit(2, Side::Buy, 110, 5)); // aggressive taker

    ASSERT_EQ(res.fills.size(), 1u);
    EXPECT_EQ(res.fills[0].price, 100); // not 110
}

TEST(OrderBook, MarketOrderAgainstEmptyBookNeverRests) {
    OrderBook book;
    auto res = book.submit(market(1, Side::Buy, 10));

    EXPECT_EQ(res.status, SubmitStatus::Accepted);
    EXPECT_TRUE(res.fills.empty());
    EXPECT_EQ(res.remaining, 10u);
    EXPECT_FALSE(res.resting);
    EXPECT_EQ(book.open_order_count(), 0u);
}

TEST(OrderBook, CancelRemovesOrderAndEmptyLevel) {
    OrderBook book;
    book.submit(limit(1, Side::Buy, 100, 10));
    book.submit(limit(2, Side::Buy, 99, 10));

    EXPECT_TRUE(book.cancel(1));
    EXPECT_EQ(book.best_bid(), Price{99});
    EXPECT_FALSE(book.cancel(1)); // second cancel of same id fails
    EXPECT_FALSE(book.cancel(999)); // unknown id fails
    EXPECT_EQ(book.open_order_count(), 1u);
}

TEST(OrderBook, CancelReplaceLosesTimePriority) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 105, 5));
    book.submit(limit(2, Side::Sell, 105, 5));

    // Order 1 cancel-replaces at the same price: goes to the back of the queue.
    ASSERT_TRUE(book.cancel(1));
    book.submit(limit(3, Side::Sell, 105, 5));

    auto res = book.submit(limit(4, Side::Buy, 105, 5));
    ASSERT_EQ(res.fills.size(), 1u);
    EXPECT_EQ(res.fills[0].maker_order_id, 2u); // order 2 is now first
}

TEST(OrderBook, RejectsInvalidOrders) {
    OrderBook book;
    EXPECT_EQ(book.submit(limit(1, Side::Buy, 100, 0)).status,
              SubmitStatus::RejectedBadOrder); // zero quantity
    EXPECT_EQ(book.submit(limit(2, Side::Buy, 0, 10)).status,
              SubmitStatus::RejectedBadOrder); // non-positive limit price
    EXPECT_EQ(book.submit(limit(0, Side::Buy, 100, 10)).status,
              SubmitStatus::RejectedBadOrder); // reserved id 0
    EXPECT_EQ(book.open_order_count(), 0u);
}

TEST(OrderBook, RejectsDuplicateLiveOrderId) {
    OrderBook book;
    book.submit(limit(1, Side::Buy, 100, 10));
    auto res = book.submit(limit(1, Side::Buy, 101, 5));

    EXPECT_EQ(res.status, SubmitStatus::RejectedDuplicate);
    EXPECT_EQ(book.best_bid(), Price{100}); // book unchanged
    EXPECT_EQ(book.open_order_count(), 1u);
}

TEST(OrderBook, SnapshotAggregatesLevelsBestFirst) {
    OrderBook book;
    book.submit(limit(1, Side::Buy, 100, 10));
    book.submit(limit(2, Side::Buy, 100, 5));
    book.submit(limit(3, Side::Buy, 99, 7));
    book.submit(limit(4, Side::Sell, 101, 3));

    auto snap = book.snapshot();
    ASSERT_EQ(snap.bids.size(), 2u);
    EXPECT_EQ(snap.bids[0].price, 100);
    EXPECT_EQ(snap.bids[0].quantity, 15u); // aggregated across orders 1 and 2
    EXPECT_EQ(snap.bids[1].price, 99);
    ASSERT_EQ(snap.asks.size(), 1u);
    EXPECT_EQ(snap.asks[0].price, 101);

    auto top = book.snapshot(1);
    EXPECT_EQ(top.bids.size(), 1u);
    EXPECT_EQ(top.asks.size(), 1u);
}

TEST(OrderBook, FillSequenceNumbersStrictlyIncrease) {
    OrderBook book;
    book.submit(limit(1, Side::Sell, 100, 5));
    book.submit(limit(2, Side::Sell, 101, 5));
    auto res = book.submit(limit(3, Side::Buy, 101, 10)); // sweeps both levels

    ASSERT_EQ(res.fills.size(), 2u);
    EXPECT_LT(res.fills[0].sequence, res.fills[1].sequence);
}

// Property test: after every submit the book must never be crossed
// (best_bid < best_ask). A crossed book means a match was missed — the single
// most important invariant of a matching engine. Fuzzed with a fixed seed so
// failures are reproducible.
TEST(OrderBook, BookIsNeverCrossedUnderRandomFlow) {
    OrderBook book;
    std::mt19937_64 rng(1234);
    std::uniform_int_distribution<Price> price(95, 105);
    std::uniform_int_distribution<Quantity> qty(1, 20);
    std::bernoulli_distribution buy(0.5);

    for (OrderId id = 1; id <= 20000; ++id) {
        book.submit(limit(id, buy(rng) ? Side::Buy : Side::Sell, price(rng), qty(rng)));
        if (book.best_bid() && book.best_ask()) {
            EXPECT_LT(book.best_bid().value(), book.best_ask().value())
                << "crossed book after order " << id;
        }
    }
}
