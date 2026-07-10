#pragma once

#include <memory>
#include <string>

namespace engine {

// Minimal metrics sink the engine reports to. The base class is a no-op so the
// engine has zero instrumentation overhead and zero extra dependencies when
// built without prometheus-cpp; make_metrics() returns a Prometheus-backed
// implementation (serving /metrics on bind_addr) only when HAVE_PROMETHEUS is
// defined. This keeps observability optional at the edges, like gRPC and Kafka.
class Metrics {
public:
    virtual ~Metrics() = default;
    virtual void order_consumed() {}
    virtual void fill_produced() {}
    virtual void observe_match_latency_seconds(double /*seconds*/) {}
};

// Returns a Prometheus exposer on bind_addr (e.g. "0.0.0.0:9101") when built
// with prometheus-cpp, otherwise a no-op sink. Never returns null.
std::unique_ptr<Metrics> make_metrics(const std::string& bind_addr);

} // namespace engine
