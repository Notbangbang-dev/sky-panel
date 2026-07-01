-- +goose Up
CREATE TABLE ledger_entries (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount     INTEGER NOT NULL,
    reason     TEXT NOT NULL,
    metadata   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ledger_entries_user_id ON ledger_entries(user_id);

CREATE TABLE afk_state (
    user_id            TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_heartbeat_at  TIMESTAMP NOT NULL
);

CREATE TABLE daily_reward_claims (
    user_id         TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_claimed_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE daily_reward_claims;
DROP TABLE afk_state;
DROP TABLE ledger_entries;
