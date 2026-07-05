-- +goose Up
-- Per-user database-count quota bonus (summed with the global default).
ALTER TABLE user_quotas ADD COLUMN bonus_databases INTEGER NOT NULL DEFAULT 0;

-- MariaDB databases provisioned for users on their servers' nodes. Credentials
-- are stored so the owner can view them; the actual database lives on the node.
-- Rows are removed with their server (CASCADE); the panel drops the underlying
-- database on the node first.
CREATE TABLE databases (
    id         TEXT PRIMARY KEY,
    owner_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id  TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    node_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    username   TEXT NOT NULL,
    password   TEXT NOT NULL,
    host       TEXT NOT NULL,
    port       INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    UNIQUE(node_id, name)
);

CREATE INDEX idx_databases_owner ON databases(owner_id);
CREATE INDEX idx_databases_server ON databases(server_id);

-- +goose Down
DROP TABLE databases;
ALTER TABLE user_quotas DROP COLUMN bonus_databases;
