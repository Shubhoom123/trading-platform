package com.tradingplatform.api.engine;

import com.tradingplatform.api.domain.OrderEntity;
import com.tradingplatform.proto.v1.Order;
import com.tradingplatform.proto.v1.OrderType;
import com.tradingplatform.proto.v1.Side;
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

// The API's link to the matching engine. As of Phase 4 this is event-driven:
// validated orders are published as Order protobufs to the Kafka "orders" topic
// (keyed by symbol so a symbol's flow stays ordered within its partition)
// instead of a synchronous gRPC call. Fills come back asynchronously on the
// "fills" topic — see OrderFillConsumer.
@Component
public class MatchingEngineClient {

    private final KafkaTemplate<String, byte[]> kafka;
    private final String ordersTopic;
    private final Counter ordersPublished;

    public MatchingEngineClient(KafkaTemplate<String, byte[]> kafka,
                                @Value("${kafka.topics.orders}") String ordersTopic,
                                MeterRegistry meters) {
        this.kafka = kafka;
        this.ordersTopic = ordersTopic;
        this.ordersPublished = Counter.builder("api.orders.published")
                .description("Orders published to the orders topic")
                .register(meters);
    }

    public void submit(OrderEntity order) {
        Order proto = Order.newBuilder()
                .setId(order.getId())
                .setSymbol(order.getSymbol())
                .setSide(order.getSide() == com.tradingplatform.api.domain.Side.BUY
                        ? Side.SIDE_BUY : Side.SIDE_SELL)
                .setType(order.getType() == com.tradingplatform.api.domain.OrderType.LIMIT
                        ? OrderType.ORDER_TYPE_LIMIT : OrderType.ORDER_TYPE_MARKET)
                .setPriceTicks(order.getPriceTicks())
                .setQuantity(order.getQuantity())
                .build();

        // Key by symbol for partition affinity. Fire-and-forget: the order is
        // already persisted as NEW, so delivery is retried by the producer.
        kafka.send(ordersTopic, order.getSymbol(), proto.toByteArray());
        ordersPublished.increment();
    }
}
