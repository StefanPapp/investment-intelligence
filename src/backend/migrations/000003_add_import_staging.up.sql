CREATE TABLE imports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    filename TEXT NOT NULL,
    file_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE import_staging_rows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    import_id UUID NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    trade_date TEXT,
    symbol TEXT,
    side TEXT,
    quantity NUMERIC(18, 8),
    price_per_share NUMERIC(18, 8),
    currency TEXT DEFAULT 'USD',
    fees NUMERIC(18, 8) DEFAULT 0,
    account TEXT,
    source_row TEXT,
    warnings TEXT[] DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'ready',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_staging_rows_import_id ON import_staging_rows(import_id);
