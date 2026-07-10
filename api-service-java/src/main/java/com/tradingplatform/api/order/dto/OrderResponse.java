package com.tradingplatform.api.order.dto;

import com.tradingplatform.api.domain.OrderEntity;

import java.time.Instant;

public record OrderResponse(
        Long id,
        String symbol,
        String side,
        String type,
        long priceTicks,
        long quantity,
        long filledQuantity,
        String status,
        Instant createdAt) {

    public static OrderResponse from(OrderEntity o) {
        return new OrderResponse(
                o.getId(),
                o.getSymbol(),
                o.getSide().name(),
                o.getType().name(),
                o.getPriceTicks(),
                o.getQuantity(),
                o.getFilledQuantity(),
                o.getStatus().name(),
                o.getCreatedAt());
    }
}
