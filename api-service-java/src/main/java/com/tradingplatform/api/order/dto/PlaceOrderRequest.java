package com.tradingplatform.api.order.dto;

import com.tradingplatform.api.domain.OrderType;
import com.tradingplatform.api.domain.Side;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;

import java.math.BigDecimal;

// Callers speak in decimal prices; the API owns the tick conversion, so the
// wire format for the engine (integer ticks) never leaks to clients.
public record PlaceOrderRequest(
        @NotBlank String symbol,
        @NotNull Side side,
        @NotNull OrderType type,
        // Required for LIMIT, ignored for MARKET (validated in the service).
        BigDecimal price,
        @NotNull @Positive Long quantity) {
}
