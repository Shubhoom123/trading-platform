package com.tradingplatform.api.domain;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;

@Entity
@Table(name = "accounts")
public class Account {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "user_id", nullable = false)
    private Long userId;

    // Available cash in integer ticks. Buy orders are checked against this.
    @Column(name = "balance_ticks", nullable = false)
    private long balanceTicks;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt = Instant.now();

    protected Account() {
        // JPA
    }

    public Account(Long userId, long balanceTicks) {
        this.userId = userId;
        this.balanceTicks = balanceTicks;
    }

    public Long getId() {
        return id;
    }

    public Long getUserId() {
        return userId;
    }

    public long getBalanceTicks() {
        return balanceTicks;
    }

    public void debit(long ticks) {
        if (ticks < 0 || ticks > balanceTicks) {
            throw new IllegalArgumentException("insufficient balance");
        }
        this.balanceTicks -= ticks;
    }
}
