#include "engine_service.hpp"

#include "proto_convert.hpp"

#include <chrono>

namespace engine {

grpc::Status MatchingEngineServiceImpl::SubmitOrder(
    grpc::ServerContext* /*context*/,
    const trading::v1::SubmitOrderRequest* request,
    trading::v1::SubmitOrderResponse* response) {

    const auto& po = request->order();
    if (po.symbol().empty()) {
        return {grpc::StatusCode::INVALID_ARGUMENT, "order.symbol is required"};
    }

    Order order;
    if (!to_engine_order(po, order)) {
        return {grpc::StatusCode::INVALID_ARGUMENT,
                "order.side and order.type must be specified"};
    }

    BookRegistry::SubmitOutcome outcome = registry_.submit(po.symbol(), order);

    response->set_status(to_proto_status(outcome.status));
    response->set_remaining_quantity(outcome.remaining);
    response->set_resting(outcome.resting);
    for (const auto& f : outcome.fills) {
        *response->add_fills() = f;
    }
    return grpc::Status::OK;
}

grpc::Status MatchingEngineServiceImpl::CancelOrder(
    grpc::ServerContext* /*context*/,
    const trading::v1::CancelOrderRequest* request,
    trading::v1::CancelOrderResponse* response) {

    if (request->symbol().empty()) {
        return {grpc::StatusCode::INVALID_ARGUMENT, "symbol is required"};
    }
    response->set_cancelled(registry_.cancel(request->symbol(), request->order_id()));
    return grpc::Status::OK;
}

grpc::Status MatchingEngineServiceImpl::GetBookSnapshot(
    grpc::ServerContext* /*context*/,
    const trading::v1::GetBookSnapshotRequest* request,
    trading::v1::BookSnapshot* response) {

    if (request->symbol().empty()) {
        return {grpc::StatusCode::INVALID_ARGUMENT, "symbol is required"};
    }

    OrderBook::Snapshot snap = registry_.snapshot(request->symbol(), request->depth());

    response->set_symbol(request->symbol());
    for (const auto& lvl : snap.bids) {
        auto* out = response->add_bids();
        out->set_price_ticks(lvl.price);
        out->set_quantity(lvl.quantity);
    }
    for (const auto& lvl : snap.asks) {
        auto* out = response->add_asks();
        out->set_price_ticks(lvl.price);
        out->set_quantity(lvl.quantity);
    }
    return grpc::Status::OK;
}

grpc::Status MatchingEngineServiceImpl::StreamFills(
    grpc::ServerContext* context,
    const trading::v1::StreamFillsRequest* request,
    grpc::ServerWriter<trading::v1::Fill>* writer) {

    if (request->symbol().empty()) {
        return {grpc::StatusCode::INVALID_ARGUMENT, "symbol is required"};
    }

    auto sub = registry_.broadcaster().subscribe(request->symbol());
    using PopResult = FillBroadcaster::PopResult;

    trading::v1::Fill fill;
    while (!context->IsCancelled()) {
        switch (sub->wait_and_pop(fill, std::chrono::milliseconds(200))) {
            case PopResult::Item:
                if (!writer->Write(fill)) {
                    registry_.broadcaster().unsubscribe(request->symbol(), sub);
                    return grpc::Status::OK;
                }
                break;
            case PopResult::Closed:
                registry_.broadcaster().unsubscribe(request->symbol(), sub);
                return grpc::Status::OK;
            case PopResult::Timeout:
                break;
        }
    }

    registry_.broadcaster().unsubscribe(request->symbol(), sub);
    return grpc::Status::OK;
}

} // namespace engine
