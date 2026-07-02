-- +goose Up
-- Admin suspension: a suspended server is stopped and its owner can't start
-- or control it until an admin unsuspends it.
ALTER TABLE servers ADD COLUMN suspended INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE servers DROP COLUMN suspended;
