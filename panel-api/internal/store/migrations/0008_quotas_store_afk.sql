-- +goose Up
-- Declared disk allocation per server, counted against the owner's disk quota.
-- Not hard-enforced on the node yet (Docker disk quotas are storage-driver
-- dependent); this is the reservation the panel accounts for.
ALTER TABLE servers ADD COLUMN disk_bytes INTEGER NOT NULL DEFAULT 0;

-- Per-user bonus quota accumulated from store purchases and admin grants.
-- Effective quota = global default (settings) + these bonuses. A user with no
-- row simply has zero bonus on top of the defaults.
CREATE TABLE user_quotas (
    user_id            TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bonus_memory_bytes INTEGER NOT NULL DEFAULT 0,
    bonus_cpu_percent  INTEGER NOT NULL DEFAULT 0,
    bonus_disk_bytes   INTEGER NOT NULL DEFAULT 0
);

-- Single active AFK session per user: a second browser tab is rejected until
-- the current session goes stale. session_id is the tab's random token, and
-- session_started_at lets the client show an accurate session timer that
-- survives a page refresh.
ALTER TABLE afk_state ADD COLUMN session_id TEXT NOT NULL DEFAULT '';
ALTER TABLE afk_state ADD COLUMN session_started_at TIMESTAMP;

-- +goose Down
ALTER TABLE afk_state DROP COLUMN session_started_at;
ALTER TABLE afk_state DROP COLUMN session_id;
DROP TABLE user_quotas;
ALTER TABLE servers DROP COLUMN disk_bytes;
