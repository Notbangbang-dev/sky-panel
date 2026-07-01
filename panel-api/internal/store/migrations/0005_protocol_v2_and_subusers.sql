-- +goose Up

-- The daemon rewrite (sky-daemon, Rust) moves to a signed wire protocol:
-- every message after hello carries an HMAC-SHA256 signature keyed by the
-- node's raw token, so the panel needs the raw secret on hand (not just its
-- hash) to verify. token_hash stays as the fast lookup index for hello.
ALTER TABLE nodes ADD COLUMN token TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN expires_at TIMESTAMP NOT NULL DEFAULT (datetime('now', '+90 days'));

CREATE TABLE server_subusers (
    id         TEXT PRIMARY KEY,
    server_id  TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- comma-separated subset of: console,files,power,settings
    permissions TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, user_id)
);

CREATE INDEX idx_server_subusers_server_id ON server_subusers(server_id);
CREATE INDEX idx_server_subusers_user_id ON server_subusers(user_id);

-- +goose Down
DROP TABLE server_subusers;
ALTER TABLE nodes DROP COLUMN expires_at;
ALTER TABLE nodes DROP COLUMN token;
