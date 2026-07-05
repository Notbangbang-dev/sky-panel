-- +goose Up
-- Opt-in flag exposing a read-only public status page for a server at /status/<id>.
ALTER TABLE servers ADD COLUMN public_status BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE servers DROP COLUMN public_status;
