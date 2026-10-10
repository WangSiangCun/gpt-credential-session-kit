ALTER TABLE redeem_codes
    ADD COLUMN IF NOT EXISTS global_user_once BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_redeem_codes_global_user_once
    ON redeem_codes (global_user_once);

CREATE TABLE IF NOT EXISTS redeem_global_user_claims (
    user_id BIGINT PRIMARY KEY,
    redeem_code_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
