-- +goose Up
-- Per-server automations: run a power action / backup / console command on a
-- fixed interval. A background scheduler fires those that are due.
CREATE TABLE server_schedules (
    id               TEXT PRIMARY KEY,
    server_id        TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name             TEXT NOT NULL DEFAULT '',
    action           TEXT NOT NULL,            -- start | stop | restart | kill | backup | command
    payload          TEXT NOT NULL DEFAULT '', -- console line, for action = command
    interval_minutes INTEGER NOT NULL,
    enabled          INTEGER NOT NULL DEFAULT 1,
    last_run_at      TIMESTAMP,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_server_schedules_server_id ON server_schedules(server_id);

-- +goose Down
DROP TABLE server_schedules;
