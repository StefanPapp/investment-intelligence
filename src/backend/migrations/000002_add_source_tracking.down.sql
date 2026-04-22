DROP INDEX IF EXISTS idx_transactions_source_unique;
ALTER TABLE transactions DROP COLUMN IF EXISTS source_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS source;
ALTER TABLE stocks DROP COLUMN IF EXISTS source;
