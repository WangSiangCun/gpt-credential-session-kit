ALTER TABLE redeem_codes
    ADD COLUMN IF NOT EXISTS batch_id VARCHAR(32),
    ADD COLUMN IF NOT EXISTS batch_user_once BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_redeem_codes_batch_id
    ON redeem_codes (batch_id);

CREATE TABLE IF NOT EXISTS redeem_batch_user_claims (
    batch_id VARCHAR(32) NOT NULL,
    user_id BIGINT NOT NULL,
    redeem_code_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (batch_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_redeem_batch_user_claims_user_id
    ON redeem_batch_user_claims (user_id);
