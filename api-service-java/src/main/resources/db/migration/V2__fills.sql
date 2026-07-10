-- Phase 5: fills become part of the system of record.
--
-- The engine assigns a strictly increasing sequence to every execution. We make
-- it the natural idempotency key: Kafka is at-least-once, so a redelivered fill
-- must not be inserted or applied twice. A UNIQUE constraint turns a replay into
-- a no-op insert the consumer can detect and skip.

CREATE TABLE fills (
    id              BIGSERIAL PRIMARY KEY,
    sequence        BIGINT      NOT NULL UNIQUE,   -- engine execution sequence
    symbol          VARCHAR(16) NOT NULL,
    price_ticks     BIGINT      NOT NULL,
    quantity        BIGINT      NOT NULL CHECK (quantity > 0),
    maker_order_id  BIGINT      NOT NULL,
    taker_order_id  BIGINT      NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fills_symbol ON fills(symbol);
CREATE INDEX idx_fills_maker_order_id ON fills(maker_order_id);
CREATE INDEX idx_fills_taker_order_id ON fills(taker_order_id);
