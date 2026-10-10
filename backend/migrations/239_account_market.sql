-- User-contributed upstream accounts and withdrawal settlement.
-- Isolated from the core account, billing and affiliate schemas.
-- Earlier PoolAI experiments used the same earnings/withdrawals table names
-- with incompatible columns. Preserve those records under explicit legacy
-- names before installing the current schema.
DO $$
BEGIN
    IF to_regclass('public.account_market_earnings') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public'
             AND table_name = 'account_market_earnings'
             AND column_name = 'uploader_user_id'
       ) THEN
        ALTER TABLE account_market_earnings RENAME TO account_market_earnings_legacy_pre239;
        ALTER INDEX IF EXISTS account_market_earnings_pkey
            RENAME TO account_market_earnings_legacy_pre239_pkey;
        ALTER INDEX IF EXISTS account_market_earnings_upload_id_request_id_key
            RENAME TO account_market_earnings_legacy_pre239_upload_request_key;
        ALTER INDEX IF EXISTS idx_account_market_earnings_upload
            RENAME TO idx_account_market_earnings_legacy_pre239_upload;
    END IF;

    IF to_regclass('public.account_market_withdrawals') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'public'
             AND table_name = 'account_market_withdrawals'
             AND column_name = 'commission_percent'
       ) THEN
        ALTER TABLE account_market_withdrawals RENAME TO account_market_withdrawals_legacy_pre239;
        ALTER INDEX IF EXISTS account_market_withdrawals_pkey
            RENAME TO account_market_withdrawals_legacy_pre239_pkey;
        ALTER INDEX IF EXISTS idx_account_market_withdrawals_user
            RENAME TO idx_account_market_withdrawals_legacy_pre239_user;
        ALTER INDEX IF EXISTS idx_account_market_withdrawals_status
            RENAME TO idx_account_market_withdrawals_legacy_pre239_status;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS account_market_config (
    id BIGINT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    target_group_id BIGINT,
    commission_percent NUMERIC(7,4) NOT NULL DEFAULT 0,
    minimum_withdrawal NUMERIC(20,10) NOT NULL DEFAULT 0,
    withdrawal_review_required BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_market_config_commission_range CHECK (commission_percent >= 0 AND commission_percent <= 100),
    CONSTRAINT account_market_config_minimum_nonnegative CHECK (minimum_withdrawal >= 0)
);

INSERT INTO account_market_config (id) VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS account_market_accounts (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL UNIQUE,
    uploader_user_id BIGINT NOT NULL,
    target_group_id BIGINT NOT NULL,
    account_email TEXT,
    account_email_hash CHAR(64),
    credential_fingerprint CHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_market_accounts_status CHECK (status IN ('pending', 'active', 'paused', 'removed', 'rejected'))
);

CREATE UNIQUE INDEX IF NOT EXISTS account_market_accounts_email_uq
    ON account_market_accounts (account_email_hash)
    WHERE account_email_hash IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS account_market_accounts_fingerprint_uq
    ON account_market_accounts (credential_fingerprint);
CREATE INDEX IF NOT EXISTS account_market_accounts_uploader_idx
    ON account_market_accounts (uploader_user_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS account_market_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    gross_amount NUMERIC(20,10) NOT NULL,
    commission_percent NUMERIC(7,4) NOT NULL,
    commission_amount NUMERIC(20,10) NOT NULL,
    net_amount NUMERIC(20,10) NOT NULL,
    payout_method VARCHAR(32) NOT NULL,
    payout_account TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    review_note TEXT,
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_market_withdrawals_status CHECK (status IN ('pending', 'approved', 'paid', 'rejected', 'cancelled')),
    CONSTRAINT account_market_withdrawals_amounts_nonnegative CHECK (gross_amount >= 0 AND commission_amount >= 0 AND net_amount >= 0)
);
CREATE INDEX IF NOT EXISTS account_market_withdrawals_user_idx
    ON account_market_withdrawals (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS account_market_withdrawals_status_idx
    ON account_market_withdrawals (status, created_at DESC);

CREATE TABLE IF NOT EXISTS account_market_earnings (
    id BIGSERIAL PRIMARY KEY,
    uploader_user_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    usage_log_id BIGINT NOT NULL UNIQUE,
    gross_amount NUMERIC(20,10) NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'available',
    withdrawal_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_market_earnings_status CHECK (status IN ('available', 'locked', 'withdrawn', 'reversed')),
    CONSTRAINT account_market_earnings_amount_nonnegative CHECK (gross_amount >= 0)
);
CREATE INDEX IF NOT EXISTS account_market_earnings_user_status_idx
    ON account_market_earnings (uploader_user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS account_market_earnings_account_idx
    ON account_market_earnings (account_id, created_at DESC);
