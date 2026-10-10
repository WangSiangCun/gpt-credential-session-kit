-- Scope contributed-account identity to its provider and account type.
-- Keep the already published migration 239 unchanged.
ALTER TABLE account_market_accounts
    ADD COLUMN IF NOT EXISTS platform VARCHAR(32),
    ADD COLUMN IF NOT EXISTS account_type VARCHAR(32),
    ADD COLUMN IF NOT EXISTS operation_token TEXT,
    ADD COLUMN IF NOT EXISTS operation_expires_at TIMESTAMPTZ;

UPDATE account_market_accounts ama
SET platform = a.platform,
    account_type = a.type
FROM accounts a
WHERE a.id = ama.account_id
  AND (ama.platform IS NULL OR ama.account_type IS NULL);

UPDATE account_market_accounts
SET platform = COALESCE(platform, 'legacy'),
    account_type = COALESCE(account_type, 'legacy-' || account_id::TEXT)
WHERE platform IS NULL OR account_type IS NULL;

ALTER TABLE account_market_accounts
    ALTER COLUMN platform SET NOT NULL,
    ALTER COLUMN account_type SET NOT NULL;

DROP INDEX IF EXISTS account_market_accounts_email_uq;
CREATE UNIQUE INDEX account_market_accounts_email_uq
    ON account_market_accounts (platform, account_type, account_email_hash)
    WHERE account_email_hash IS NOT NULL;
