ALTER TABLE lottery_campaigns
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_lottery_campaigns_not_deleted_created
    ON lottery_campaigns(created_at DESC)
    WHERE deleted_at IS NULL;
