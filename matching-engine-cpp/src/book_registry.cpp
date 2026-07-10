#include "book_registry.hpp"

#include <algorithm>

namespace engine {

// --- FillBroadcaster --------------------------------------------------------

FillBroadcaster::PopResult
FillBroadcaster::Subscription::wait_and_pop(trading::v1::Fill& out,
                                            std::chrono::milliseconds timeout) {
    std::unique_lock<std::mutex> lock(mu_);
    cv_.wait_for(lock, timeout, [this] { return closed_ || !queue_.empty(); });
    if (!queue_.empty()) {
        out = std::move(queue_.front());
        queue_.pop_front();
        return PopResult::Item;
    }
    return closed_ ? PopResult::Closed : PopResult::Timeout;
}

std::shared_ptr<FillBroadcaster::Subscription>
FillBroadcaster::subscribe(const std::string& symbol) {
    auto sub = std::make_shared<Subscription>();
    std::lock_guard<std::mutex> lock(mu_);
    subs_[symbol].push_back(sub);
    return sub;
}

void FillBroadcaster::unsubscribe(const std::string& symbol,
                                  const std::shared_ptr<Subscription>& sub) {
    {
        std::lock_guard<std::mutex> sub_lock(sub->mu_);
        sub->closed_ = true;
    }
    sub->cv_.notify_all();

    std::lock_guard<std::mutex> lock(mu_);
    auto it = subs_.find(symbol);
    if (it == subs_.end()) return;
    auto& vec = it->second;
    vec.erase(std::remove(vec.begin(), vec.end(), sub), vec.end());
    if (vec.empty()) subs_.erase(it);
}

void FillBroadcaster::publish(const std::string& symbol,
                              const trading::v1::Fill& fill) {
    std::vector<std::shared_ptr<Subscription>> targets;
    {
        std::lock_guard<std::mutex> lock(mu_);
        auto it = subs_.find(symbol);
        if (it == subs_.end()) return;
        targets = it->second;
    }
    for (const auto& sub : targets) {
        {
            std::lock_guard<std::mutex> sub_lock(sub->mu_);
            if (sub->closed_) continue;
            sub->queue_.push_back(fill);
        }
        sub->cv_.notify_one();
    }
}

// --- BookRegistry -----------------------------------------------------------

OrderBook& BookRegistry::book_for(const std::string& symbol) {
    return books_[symbol];
}

BookRegistry::SubmitOutcome
BookRegistry::submit(const std::string& symbol, Order order) {
    OrderBook::SubmitResult result;
    {
        std::lock_guard<std::mutex> lock(mu_);
        result = book_for(symbol).submit(order);
    }

    SubmitOutcome outcome;
    outcome.status    = result.status;
    outcome.remaining = result.remaining;
    outcome.resting   = result.resting;
    outcome.fills.reserve(result.fills.size());

    for (const Fill& f : result.fills) {
        trading::v1::Fill pf;
        pf.set_maker_order_id(f.maker_order_id);
        pf.set_taker_order_id(f.taker_order_id);
        pf.set_symbol(symbol);
        pf.set_price_ticks(f.price);
        pf.set_quantity(f.quantity);
        pf.set_sequence(f.sequence);
        // Fan out to gRPC stream subscribers regardless of who submitted the
        // order (gRPC or Kafka), then hand the fill back to the caller.
        broadcaster_.publish(symbol, pf);
        outcome.fills.push_back(std::move(pf));
    }
    return outcome;
}

bool BookRegistry::cancel(const std::string& symbol, OrderId id) {
    std::lock_guard<std::mutex> lock(mu_);
    return book_for(symbol).cancel(id);
}

OrderBook::Snapshot BookRegistry::snapshot(const std::string& symbol,
                                           std::size_t depth) {
    std::lock_guard<std::mutex> lock(mu_);
    return book_for(symbol).snapshot(depth);
}

} // namespace engine
