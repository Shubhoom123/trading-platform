#include "metrics.hpp"

#ifdef HAVE_PROMETHEUS
#include <prometheus/counter.h>
#include <prometheus/exposer.h>
#include <prometheus/histogram.h>
#include <prometheus/registry.h>
#endif

namespace engine {

#ifdef HAVE_PROMETHEUS

namespace {

// Latency buckets tuned for the engine's hot path: sub-microsecond to a few
// milliseconds. Matching a single order is typically hundreds of nanoseconds.
prometheus::Histogram::BucketBoundaries latency_buckets() {
    return {1e-7, 3e-7, 1e-6, 3e-6, 1e-5, 3e-5, 1e-4, 1e-3, 1e-2};
}

class PrometheusMetrics : public Metrics {
public:
    explicit PrometheusMetrics(const std::string& bind_addr)
        : exposer_(bind_addr),
          registry_(std::make_shared<prometheus::Registry>()),
          orders_(prometheus::BuildCounter()
                      .Name("engine_orders_consumed_total")
                      .Help("Orders consumed from the orders topic")
                      .Register(*registry_)
                      .Add({})),
          fills_(prometheus::BuildCounter()
                     .Name("engine_fills_produced_total")
                     .Help("Fills produced to the fills topic")
                     .Register(*registry_)
                     .Add({})),
          latency_(prometheus::BuildHistogram()
                       .Name("engine_match_latency_seconds")
                       .Help("Time to match one order against the book")
                       .Register(*registry_)
                       .Add({}, latency_buckets())) {
        exposer_.RegisterCollectable(registry_);
    }

    void order_consumed() override { orders_.Increment(); }
    void fill_produced() override { fills_.Increment(); }
    void observe_match_latency_seconds(double s) override { latency_.Observe(s); }

private:
    prometheus::Exposer                      exposer_;
    std::shared_ptr<prometheus::Registry>    registry_;
    prometheus::Counter&                     orders_;
    prometheus::Counter&                     fills_;
    prometheus::Histogram&                   latency_;
};

} // namespace

std::unique_ptr<Metrics> make_metrics(const std::string& bind_addr) {
    if (!bind_addr.empty()) {
        return std::make_unique<PrometheusMetrics>(bind_addr);
    }
    return std::make_unique<Metrics>();
}

#else // no prometheus-cpp

std::unique_ptr<Metrics> make_metrics(const std::string& /*bind_addr*/) {
    return std::make_unique<Metrics>();
}

#endif

} // namespace engine
