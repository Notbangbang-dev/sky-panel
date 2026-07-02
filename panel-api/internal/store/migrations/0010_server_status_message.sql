-- +goose Up
-- Human-readable detail for the current status — notably why provisioning
-- failed, so an "errored" server can explain itself in the UI instead of
-- leaving the operator guessing.
ALTER TABLE servers ADD COLUMN status_message TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE servers DROP COLUMN status_message;
