package com.tradingplatform.api.engine;

import com.google.protobuf.InvalidProtocolBufferException;
import com.tradingplatform.api.domain.FillRecord;
import com.tradingplatform.api.domain.OrderEntity;
import com.tradingplatform.api.repository.FillRepository;
import com.tradingplatform.api.repository.OrderRepository;
import com.tradingplatform.proto.v1.Fill;
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

// Closes the event loop: consumes Fill protobufs the engine produces and folds
// them into the system of record — a durable fills row plus updated order state.
//
// Idempotent by design: Kafka is at-least-once, so a redelivered fill would
// otherwise double-count. The engine's monotonic `sequence` is the dedup key —
// if we've already stored it, we skip the whole message. Both sides of a trade
// are orders we own, so each fill advances the taker and the maker order.
@Component
public class OrderFillConsumer {

    private static final Logger log = LoggerFactory.getLogger(OrderFillConsumer.class);

    private final OrderRepository orders;
    private final FillRepository fills;
    private final Counter fillsConsumed;

    public OrderFillConsumer(OrderRepository orders, FillRepository fills, MeterRegistry meters) {
        this.orders = orders;
        this.fills = fills;
        this.fillsConsumed = Counter.builder("api.fills.consumed")
                .description("Distinct fills consumed and persisted")
                .register(meters);
    }

    @KafkaListener(topics = "${kafka.topics.fills}", groupId = "api-fill-updater")
    @Transactional
    public void onMessage(byte[] payload) {
        final Fill fill;
        try {
            fill = Fill.parseFrom(payload);
        } catch (InvalidProtocolBufferException e) {
            log.warn("skipping malformed fill message");
            return;
        }

        // Dedup on the engine sequence: a replayed fill is a no-op.
        if (fills.existsBySequence(fill.getSequence())) {
            return;
        }

        fills.save(new FillRecord(
                fill.getSequence(),
                fill.getSymbol(),
                fill.getPriceTicks(),
                fill.getQuantity(),
                fill.getMakerOrderId(),
                fill.getTakerOrderId()));

        applyTo(fill.getTakerOrderId(), fill.getQuantity());
        applyTo(fill.getMakerOrderId(), fill.getQuantity());
        fillsConsumed.increment();
    }

    private void applyTo(long orderId, long quantity) {
        OrderEntity order = orders.findById(orderId).orElse(null);
        if (order == null) {
            // A fill for an order this instance doesn't know about (e.g. history
            // replayed before this DB existed). The fills row is still recorded.
            return;
        }
        order.applyFill(quantity);
        orders.save(order);
    }
}
