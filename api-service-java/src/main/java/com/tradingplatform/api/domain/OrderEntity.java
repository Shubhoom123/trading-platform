package com.tradingplatform.api.domain;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;

// Named OrderEntity, not Order, to avoid clashing with the generated proto
// Order message and with SQL's reserved word in queries.
@Entity
@Table(name = "orders")
public class OrderEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "account_id", nullable = false)
    private Long accountId;

    @Column(nullable = false)
    private String symbol;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private Side side;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private OrderType type;

    @Column(name = "price_ticks", nullable = false)
    private long priceTicks;

    @Column(nullable = false)
    private long quantity;

    @Column(name = "filled_quantity", nullable = false)
    private long filledQuantity;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private OrderStatus status = OrderStatus.NEW;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt = Instant.now();

    protected OrderEntity() {
        // JPA
    }

    public OrderEntity(Long accountId, String symbol, Side side, OrderType type,
                       long priceTicks, long quantity) {
        this.accountId = accountId;
        this.symbol = symbol;
        this.side = side;
        this.type = type;
        this.priceTicks = priceTicks;
        this.quantity = quantity;
    }

    public void applyFill(long filledDelta) {
        this.filledQuantity += filledDelta;
        if (filledQuantity >= quantity) {
            this.status = OrderStatus.FILLED;
        } else if (filledQuantity > 0) {
            this.status = OrderStatus.PARTIALLY_FILLED;
        }
    }

    public void markRejected() {
        this.status = OrderStatus.REJECTED;
    }

    public Long getId() {
        return id;
    }

    public Long getAccountId() {
        return accountId;
    }

    public String getSymbol() {
        return symbol;
    }

    public Side getSide() {
        return side;
    }

    public OrderType getType() {
        return type;
    }

    public long getPriceTicks() {
        return priceTicks;
    }

    public long getQuantity() {
        return quantity;
    }

    public long getFilledQuantity() {
        return filledQuantity;
    }

    public OrderStatus getStatus() {
        return status;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }
}
