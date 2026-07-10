#pragma once

#include "book_registry.hpp"
#include "trading.grpc.pb.h"

#include <grpcpp/grpcpp.h>

namespace engine {

// gRPC front door for the matching engine. It does not own the books — it
// borrows a shared BookRegistry so that reads served here (GetBookSnapshot)
// reflect orders matched by the Kafka bridge, and vice versa.
//
// SubmitOrder/StreamFills remain functional (useful for tests and direct pokes),
// but the production Phase 4 order flow runs through Kafka, not these RPCs.
class MatchingEngineServiceImpl final
    : public trading::v1::MatchingEngine::Service {
public:
    explicit MatchingEngineServiceImpl(BookRegistry& registry)
        : registry_(registry) {}

    grpc::Status SubmitOrder(grpc::ServerContext* context,
                             const trading::v1::SubmitOrderRequest* request,
                             trading::v1::SubmitOrderResponse* response) override;

    grpc::Status CancelOrder(grpc::ServerContext* context,
                             const trading::v1::CancelOrderRequest* request,
                             trading::v1::CancelOrderResponse* response) override;

    grpc::Status GetBookSnapshot(grpc::ServerContext* context,
                                 const trading::v1::GetBookSnapshotRequest* request,
                                 trading::v1::BookSnapshot* response) override;

    grpc::Status StreamFills(grpc::ServerContext* context,
                             const trading::v1::StreamFillsRequest* request,
                             grpc::ServerWriter<trading::v1::Fill>* writer) override;

private:
    BookRegistry& registry_;
};

} // namespace engine
