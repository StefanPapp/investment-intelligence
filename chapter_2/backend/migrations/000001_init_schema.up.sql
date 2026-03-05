CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE stocks (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticker      TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transactions (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stock_id          UUID NOT NULL REFERENCES stocks(id),
    transaction_type  TEXT NOT NULL CHECK (transaction_type IN ('buy', 'sell')),
    shares            NUMERIC(12,4) NOT NULL CHECK (shares > 0),
    price_per_share   NUMERIC(12,4) NOT NULL CHECK (price_per_share > 0),
    transaction_date  DATE NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE prices_cache (
    ticker      TEXT PRIMARY KEY,
    price       NUMERIC(12,4) NOT NULL,
    currency    TEXT NOT NULL DEFAULT 'USD',
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_stock_id ON transactions(stock_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
