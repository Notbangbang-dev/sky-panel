-- +goose Up
-- Per-user "starred" servers, surfaced first in the server list.
CREATE TABLE server_favorites (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id  TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, server_id)
);

-- +goose Down
DROP TABLE server_favorites;
