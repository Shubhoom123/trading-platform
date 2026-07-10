#include "book_registry.hpp"
#include "engine_service.hpp"
#include "metrics.hpp"

#ifdef HAVE_KAFKA
#include "kafka_bridge.hpp"
#include <thread>
#endif

#include <grpcpp/grpcpp.h>

#include <csignal>
#include <cstdlib>
#include <iostream>
#include <memory>
#include <string>

namespace {

std::unique_ptr<grpc::Server> g_server;
#ifdef HAVE_KAFKA
engine::KafkaBridge* g_bridge = nullptr;
#endif

void handle_signal(int /*sig*/) {
    if (g_server) g_server->Shutdown();
#ifdef HAVE_KAFKA
    if (g_bridge) g_bridge->stop();
#endif
}

std::string env_or(const char* key, const char* def) {
    const char* v = std::getenv(key);
    return (v && *v) ? v : def;
}

} // namespace

int main() {
    // One registry shared by the gRPC reads and (when enabled) the Kafka bridge.
    engine::BookRegistry registry;
    engine::MatchingEngineServiceImpl service(registry);

    // Prometheus /metrics exposer (no-op if built without prometheus-cpp).
    auto metrics = engine::make_metrics(env_or("ENGINE_METRICS_ADDR", "0.0.0.0:9101"));

    const std::string address = env_or("ENGINE_GRPC_ADDR", "0.0.0.0:50051");
    grpc::ServerBuilder builder;
    // Insecure locally; TLS between services is a tracked Phase 6 gap.
    builder.AddListeningPort(address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);

    g_server = builder.BuildAndStart();
    if (!g_server) {
        std::cerr << "failed to bind gRPC server on " << address << std::endl;
        return 1;
    }
    std::signal(SIGINT, handle_signal);
    std::signal(SIGTERM, handle_signal);
    std::cout << "matching-engine gRPC listening on " << address << std::endl;

#ifdef HAVE_KAFKA
    std::unique_ptr<engine::KafkaBridge> bridge;
    std::thread bridge_thread;
    if (const char* brokers = std::getenv("KAFKA_BOOTSTRAP_SERVERS");
        brokers && *brokers) {
        engine::KafkaConfig kc;
        kc.brokers      = brokers;
        kc.orders_topic = env_or("KAFKA_ORDERS_TOPIC", "orders");
        kc.fills_topic  = env_or("KAFKA_FILLS_TOPIC", "fills");
        kc.group_id     = env_or("KAFKA_GROUP_ID", "matching-engine");

        bridge = std::make_unique<engine::KafkaBridge>(registry, kc, *metrics);
        g_bridge = bridge.get();
        bridge_thread = std::thread([&bridge] { bridge->run(); });
        std::cout << "matching-engine kafka bridge: '" << kc.orders_topic
                  << "' -> '" << kc.fills_topic << "' on " << kc.brokers
                  << std::endl;
    } else {
        std::cout << "KAFKA_BOOTSTRAP_SERVERS not set; running gRPC-only"
                  << std::endl;
    }
#endif

    g_server->Wait();

#ifdef HAVE_KAFKA
    if (bridge_thread.joinable()) {
        bridge->stop();
        bridge_thread.join();
    }
#endif

    std::cout << "matching-engine stopped" << std::endl;
    return 0;
}
