-- +goose Up
-- Free-text note/description an owner can attach to a server.
ALTER TABLE servers ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- Redeemable coin codes: admins mint them, users redeem for coins. max_uses = 0
-- means unlimited; a per-(code,user) row enforces one redemption per user.
CREATE TABLE redeem_codes (
    id         TEXT PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    coins      INTEGER NOT NULL,
    max_uses   INTEGER NOT NULL DEFAULT 0,
    uses       INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE redeem_code_redemptions (
    code_id    TEXT NOT NULL REFERENCES redeem_codes(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (code_id, user_id)
);

-- +goose Down
DROP TABLE redeem_code_redemptions;
DROP TABLE redeem_codes;
ALTER TABLE servers DROP COLUMN description;
