-- +goose Up
CREATE TABLE nodes (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    token_hash    TEXT NOT NULL UNIQUE,
    address       TEXT NOT NULL,
    docker_socket TEXT NOT NULL DEFAULT '/var/run/docker.sock',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE eggs (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    category       TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    docker_image   TEXT NOT NULL,
    startup        TEXT NOT NULL,
    stop_command   TEXT NOT NULL DEFAULT '',
    variables_json TEXT NOT NULL DEFAULT '[]',
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE servers (
    id             TEXT PRIMARY KEY,
    owner_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    node_id        TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    egg_id         TEXT NOT NULL REFERENCES eggs(id) ON DELETE RESTRICT,
    name           TEXT NOT NULL,
    container_id   TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'installing',
    memory_bytes   INTEGER NOT NULL DEFAULT 0,
    variables_json TEXT NOT NULL DEFAULT '{}',
    primary_port   INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_servers_owner_id ON servers(owner_id);
CREATE INDEX idx_servers_node_id ON servers(node_id);

CREATE TABLE allocations (
    id        TEXT PRIMARY KEY,
    node_id   TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    port      INTEGER NOT NULL,
    server_id TEXT REFERENCES servers(id) ON DELETE SET NULL,
    UNIQUE(node_id, port)
);

CREATE INDEX idx_allocations_node_id ON allocations(node_id);

-- +goose Down
DROP TABLE allocations;
DROP TABLE servers;
DROP TABLE eggs;
DROP TABLE nodes;
