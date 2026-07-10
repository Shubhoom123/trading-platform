-- Phase 2 schema. Owned by Flyway; Hibernate is validate-only.
-- Money and prices are stored as integer ticks (BIGINT), never floating point,
-- matching the C++ engine's Price type. The API owns tick<->decimal conversion.

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,       -- BCrypt; never plaintext
    role          VARCHAR(32)  NOT NULL DEFAULT 'USER',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE accounts (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Available cash in ticks. Buy orders are checked against this at intake.
    balance_ticks BIGINT NOT NULL DEFAULT 0 CHECK (balance_ticks >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_accounts_user_id ON accounts(user_id);

CREATE TABLE orders (
    id             BIGSERIAL PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    symbol         VARCHAR(16) NOT NULL,
    side           VARCHAR(4)  NOT NULL CHECK (side IN ('BUY', 'SELL')),
    type           VARCHAR(8)  NOT NULL CHECK (type IN ('LIMIT', 'MARKET')),
    price_ticks    BIGINT      NOT NULL,          -- 0 for market orders
    quantity       BIGINT      NOT NULL CHECK (quantity > 0),
    filled_quantity BIGINT     NOT NULL DEFAULT 0 CHECK (filled_quantity >= 0),
    status         VARCHAR(16) NOT NULL DEFAULT 'NEW'
                   CHECK (status IN ('NEW', 'PARTIALLY_FILLED', 'FILLED', 'REJECTED')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_account_id ON orders(account_id);
CREATE INDEX idx_orders_symbol ON orders(symbol);
