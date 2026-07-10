#include "kafka_bridge.hpp"

#include "proto_convert.hpp"

#include <librdkafka/rdkafkacpp.h>

#include <chrono>
#include <iostream>
#include <memory>
#include <string>

namespace engine {

KafkaBridge::KafkaBridge(BookRegistry& registry, KafkaConfig cfg, Metrics& metrics)
    : registry_(registry), cfg_(std::move(cfg)), metrics_(metrics) {}

KafkaBridge::~KafkaBridge() = default;

void KafkaBridge::stop() { running_.store(false); }

namespace {

std::unique_ptr<RdKafka::Conf> globalConf() {
    return std::unique_ptr<RdKafka::Conf>(
        RdKafka::Conf::create(RdKafka::Conf::CONF_GLOBAL));
}

} // namespace

void KafkaBridge::run() {
    std::string err;

    auto cconf = globalConf();
    cconf->set("bootstrap.servers", cfg_.brokers, err);
    cconf->set("group.id", cfg_.group_id, err);
    cconf->set("enable.auto.commit", "false", err); // we commit after producing
    cconf->set("auto.offset.reset", "earliest", err);

    std::unique_ptr<RdKafka::KafkaConsumer> consumer(
        RdKafka::KafkaConsumer::create(cconf.get(), err));
    if (!consumer) {
        std::cerr << "kafka: failed to create consumer: " << err << std::endl;
        return;
    }
    if (consumer->subscribe({cfg_.orders_topic}) != RdKafka::ERR_NO_ERROR) {
        std::cerr << "kafka: failed to subscribe to " << cfg_.orders_topic << std::endl;
        return;
    }

    auto pconf = globalConf();
    pconf->set("bootstrap.servers", cfg_.brokers, err);
    // Idempotent producer: no duplicate fills on internal retries.
    pconf->set("enable.idempotence", "true", err);
    std::unique_ptr<RdKafka::Producer> producer(
        RdKafka::Producer::create(pconf.get(), err));
    if (!producer) {
        std::cerr << "kafka: failed to create producer: " << err << std::endl;
        return;
    }

    running_.store(true);
    while (running_.load()) {
        std::unique_ptr<RdKafka::Message> msg(consumer->consume(500));
        switch (msg->err()) {
            case RdKafka::ERR__TIMED_OUT:
                continue;
            case RdKafka::ERR_NO_ERROR:
                break;
            default:
                std::cerr << "kafka: consume error: " << msg->errstr() << std::endl;
                continue;
        }

        trading::v1::Order po;
        Order order;
        const bool parsed = po.ParseFromArray(msg->payload(),
                                              static_cast<int>(msg->len()));
        if (!parsed || po.symbol().empty() || !to_engine_order(po, order)) {
            // Poison message: log, commit past it, move on. A real system would
            // route this to a dead-letter topic.
            std::cerr << "kafka: skipping invalid order message" << std::endl;
            consumer->commitSync(msg.get());
            continue;
        }

        metrics_.order_consumed();
        const auto match_start = std::chrono::steady_clock::now();
        BookRegistry::SubmitOutcome outcome = registry_.submit(po.symbol(), order);
        const std::chrono::duration<double> elapsed =
            std::chrono::steady_clock::now() - match_start;
        metrics_.observe_match_latency_seconds(elapsed.count());

        for (const auto& fill : outcome.fills) {
            metrics_.fill_produced();
            std::string bytes;
            fill.SerializeToString(&bytes);
            RdKafka::ErrorCode e = producer->produce(
                cfg_.fills_topic,
                RdKafka::Topic::PARTITION_UA,
                RdKafka::Producer::RK_MSG_COPY,
                const_cast<char*>(bytes.data()), bytes.size(),
                po.symbol().data(), po.symbol().size(), // key = symbol
                0, nullptr);
            if (e != RdKafka::ERR_NO_ERROR) {
                std::cerr << "kafka: produce failed: " << RdKafka::err2str(e)
                          << std::endl;
            }
        }
        producer->poll(0);

        // At-least-once: commit only after fills have been enqueued to produce.
        consumer->commitSync(msg.get());
    }

    producer->flush(5000);
    consumer->close();
}

} // namespace engine
