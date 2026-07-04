-- +goose Up
-- redeem_code_redemptions' PK is (code_id, user_id); a lookup by user_id alone
-- (HasRedeemed, used to derive the "code redeemer" achievement) can't use that
-- composite index and would full-scan. Match the convention used by every other
-- child table queried by user_id (idx_ledger_entries_user_id, etc.).
CREATE INDEX idx_redeem_code_redemptions_user_id ON redeem_code_redemptions(user_id);

-- +goose Down
DROP INDEX idx_redeem_code_redemptions_user_id;
