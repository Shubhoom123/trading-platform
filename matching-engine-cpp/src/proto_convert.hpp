#pragma once

// Conversions between the shared proto contract and the engine's internal
// types. Shared by the gRPC service and the Kafka bridge so the mapping lives
// in exactly one place.

#include "trading.grpc.pb.h"
#include "types.hpp"

namespace engine {

// Fills `out` from a proto Order. Returns false if side/type are unspecified.
inline bool to_engine_order(const trading::v1::Order& po, Order& out) {
    switch (po.side()) {
        case trading::v1::SIDE_BUY:  out.side = Side::Buy;  break;
        case trading::v1::SIDE_SELL: out.side = Side::Sell; break;
        default: return false;
    }
    switch (po.type()) {
        case trading::v1::ORDER_TYPE_LIMIT:  out.type = OrderType::Limit;  break;
        case trading::v1::ORDER_TYPE_MARKET: out.type = OrderType::Market; break;
        default: return false;
    }
    out.id       = po.id();
    out.price    = po.price_ticks();
    out.quantity = po.quantity();
    return true;
}

inline trading::v1::SubmitOrderResponse::Status to_proto_status(SubmitStatus s) {
    switch (s) {
        case SubmitStatus::Accepted:
            return trading::v1::SubmitOrderResponse::STATUS_ACCEPTED;
        case SubmitStatus::RejectedBadOrder:
            return trading::v1::SubmitOrderResponse::STATUS_REJECTED_BAD_ORDER;
        case SubmitStatus::RejectedDuplicate:
            return trading::v1::SubmitOrderResponse::STATUS_REJECTED_DUPLICATE;
    }
    return trading::v1::SubmitOrderResponse::STATUS_UNSPECIFIED;
}

} // namespace engine
