-- +goose Up
-- The wallet reads a user's ledger ordered by time (ORDER BY created_at DESC,
-- rowid DESC LIMIT ?). The existing idx_ledger_entries_user_id filters by user
-- but leaves SQLite to sort the matched rows on the fastest-growing table in
-- the schema. A composite (user_id, created_at DESC) index serves both the
-- filter and the order, removing the sort.
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_created ON ledger_entries(user_id, created_at DESC);

-- Retention prunes refresh_tokens by expiry; index it so the sweep is a range
-- scan instead of a full-table scan as sessions accumulate.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_ledger_entries_user_created;
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
