#include "order_book.hpp"

#include <algorithm>

namespace engine {

bool OrderBook::crosses(const Order& incoming, Price resting_price) {
    if (incoming.type == OrderType::Market) return true;
    return incoming.side == Side::Buy ? incoming.price >= resting_price
                                      : incoming.price <= resting_price;
}

template <typename OppositeMap>
void OrderBook::match(Order& incoming, OppositeMap& opposite, std::vector<Fill>& fills) {
    while (incoming.quantity > 0 && !opposite.empty()) {
        auto best_it = opposite.begin();
        if (!crosses(incoming, best_it->first)) break;

        auto& queue = best_it->second;
        while (incoming.quantity > 0 && !queue.empty()) {
            Order& resting = queue.front();
            const Quantity traded = std::min(incoming.quantity, resting.quantity);

            fills.push_back(Fill{
                .maker_order_id = resting.id,
                .taker_order_id = incoming.id,
                .price          = resting.price, // maker's price, always
                .quantity       = traded,
                .sequence       = next_sequence_++,
            });

            incoming.quantity -= traded;
            resting.quantity  -= traded;

            if (resting.quantity == 0) {
                index_.erase(resting.id);
                queue.pop_front();
            }
        }
        if (queue.empty()) opposite.erase(best_it);
    }
}

void OrderBook::rest(const Order& order) {
    if (order.side == Side::Buy) {
        bids_[order.price].push_back(order);
    } else {
        asks_[order.price].push_back(order);
    }
    index_.emplace(order.id, Locator{order.side, order.price});
}

OrderBook::SubmitResult OrderBook::submit(Order order) {
    SubmitResult result;

    // Validate at this boundary too — the engine must not trust upstream
    // callers blindly, even though the Java API validates first.
    const bool bad_qty   = order.quantity == 0;
    const bool bad_price = order.type == OrderType::Limit && order.price <= 0;
    const bool bad_id    = order.id == 0;
    if (bad_qty || bad_price || bad_id) {
        result.status = SubmitStatus::RejectedBadOrder;
        return result;
    }
    if (index_.count(order.id) != 0) {
        result.status = SubmitStatus::RejectedDuplicate;
        return result;
    }

    order.sequence = next_sequence_++;

    if (order.side == Side::Buy) {
        match(order, asks_, result.fills);
    } else {
        match(order, bids_, result.fills);
    }

    result.remaining = order.quantity;
    if (order.quantity > 0 && order.type == OrderType::Limit) {
        rest(order);
        result.resting = true;
    }
    return result;
}

bool OrderBook::cancel(OrderId id) {
    const auto it = index_.find(id);
    if (it == index_.end()) return false;

    const Locator loc = it->second;
    auto erase_from = [&](auto& side_map) {
        auto level_it = side_map.find(loc.price);
        auto& queue   = level_it->second;
        queue.erase(std::find_if(queue.begin(), queue.end(),
                                 [&](const Order& o) { return o.id == id; }));
        if (queue.empty()) side_map.erase(level_it);
    };

    if (loc.side == Side::Buy) erase_from(bids_); else erase_from(asks_);
    index_.erase(it);
    return true;
}

std::optional<Price> OrderBook::best_bid() const {
    if (bids_.empty()) return std::nullopt;
    return bids_.begin()->first;
}

std::optional<Price> OrderBook::best_ask() const {
    if (asks_.empty()) return std::nullopt;
    return asks_.begin()->first;
}

OrderBook::Snapshot OrderBook::snapshot(std::size_t depth) const {
    Snapshot snap;
    auto collect = [depth](const auto& side_map, std::vector<Level>& out) {
        for (const auto& [price, queue] : side_map) {
            if (depth != 0 && out.size() >= depth) break;
            Quantity total = 0;
            for (const Order& o : queue) total += o.quantity;
            out.push_back(Level{price, total});
        }
    };
    collect(bids_, snap.bids);
    collect(asks_, snap.asks);
    return snap;
}

} // namespace engine
