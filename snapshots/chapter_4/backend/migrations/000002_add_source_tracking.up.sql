ALTER TABLE transactions ADD COLUMN source TEXT;
ALTER TABLE transactions ADD COLUMN source_id TEXT;

CREATE UNIQUE INDEX idx_transactions_source_unique
  ON transactions (source, source_id)
  WHERE source IS NOT NULL AND source_id IS NOT NULL;

ALTER TABLE stocks ADD COLUMN source TEXT;
