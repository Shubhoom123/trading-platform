#pragma once

#include <cstdint>

namespace engine {

// Prices are integer ticks, never floating point. Using doubles for money is a
// classic correctness bug (0.1 + 0.2 != 0.3) and a red flag in review.
// The API layer owns the tick-size <-> decimal conversion.
using Price    = std::int64_t;
using Quantity = std::uint64_t;
using OrderId  = std::uint64_t;
using Sequence = std::uint64_t;

enum class Side : std::uint8_t { Buy, Sell };

enum class OrderType : std::uint8_t {
    Limit,  // rests on the book if not fully filled
    Market, // fills against the book; any remainder is discarded, never rests
};

struct Order {
    OrderId   id{0};
    Side      side{Side::Buy};
    OrderType type{OrderType::Limit};
    Price     price{0};      // ignored for Market orders
    Quantity  quantity{0};   // remaining (unfilled) quantity
    Sequence  sequence{0};   // engine-assigned arrival order; drives time priority
};

// One execution between an incoming (taker) order and a resting (maker) order.
// Trades always execute at the maker's price: the resting order set the terms.
struct Fill {
    OrderId  maker_order_id{0};
    OrderId  taker_order_id{0};
    Price    price{0};
    Quantity quantity{0};
    Sequence sequence{0};
};

// Outcome of submitting an order.
enum class SubmitStatus : std::uint8_t {
    Accepted,          // matched fully, or resting (limit), or exhausted book (market)
    RejectedBadOrder,  // non-positive qty, bad price, etc.
    RejectedDuplicate, // order id already live on the book
};

} // namespace engine
