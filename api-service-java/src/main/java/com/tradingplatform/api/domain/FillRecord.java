package com.tradingplatform.api.domain;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;

// A persisted execution. Named FillRecord to avoid clashing with the generated
// proto Fill message. The engine `sequence` is unique and drives idempotency.
@Entity
@Table(name = "fills")
public class FillRecord {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false, unique = true)
    private long sequence;

    @Column(nullable = false)
    private String symbol;

    @Column(name = "price_ticks", nullable = false)
    private long priceTicks;

    @Column(nullable = false)
    private long quantity;

    @Column(name = "maker_order_id", nullable = false)
    private long makerOrderId;

    @Column(name = "taker_order_id", nullable = false)
    private long takerOrderId;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt = Instant.now();

    protected FillRecord() {
        // JPA
    }

    public FillRecord(long sequence, String symbol, long priceTicks, long quantity,
                      long makerOrderId, long takerOrderId) {
        this.sequence = sequence;
        this.symbol = symbol;
        this.priceTicks = priceTicks;
        this.quantity = quantity;
        this.makerOrderId = makerOrderId;
        this.takerOrderId = takerOrderId;
    }

    public Long getId() {
        return id;
    }

    public long getSequence() {
        return sequence;
    }

    public String getSymbol() {
        return symbol;
    }

    public long getPriceTicks() {
        return priceTicks;
    }

    public long getQuantity() {
        return quantity;
    }

    public long getMakerOrderId() {
        return makerOrderId;
    }

    public long getTakerOrderId() {
        return takerOrderId;
    }
}
