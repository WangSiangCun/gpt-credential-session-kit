ALTER TABLE lottery_campaigns
    ADD COLUMN IF NOT EXISTS activity_type VARCHAR(32) NOT NULL DEFAULT 'instant_lottery',
    ADD COLUMN IF NOT EXISTS min_total_recharged DECIMAL(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS registration_deadline TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS draw_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS winner_limit INT NOT NULL DEFAULT 0;

ALTER TABLE lottery_campaigns
    DROP CONSTRAINT IF EXISTS lottery_campaigns_activity_type_check;

ALTER TABLE lottery_campaigns
    ADD CONSTRAINT lottery_campaigns_activity_type_check
        CHECK (activity_type IN ('instant_lottery', 'registration_lottery'));

ALTER TABLE lottery_campaigns
    DROP CONSTRAINT IF EXISTS lottery_campaigns_recharge_threshold_check;

ALTER TABLE lottery_campaigns
    ADD CONSTRAINT lottery_campaigns_recharge_threshold_check
        CHECK (min_total_recharged >= 0);

ALTER TABLE lottery_draws
    DROP CONSTRAINT IF EXISTS lottery_draws_status_check;

ALTER TABLE lottery_draws
    ADD CONSTRAINT lottery_draws_status_check
        CHECK (status IN ('won', 'lost', 'registered'));

ALTER TABLE lottery_draws
    DROP CONSTRAINT IF EXISTS lottery_draws_result_consistency_check;

ALTER TABLE lottery_draws
    ADD CONSTRAINT lottery_draws_result_consistency_check
        CHECK (
            (status IN ('lost', 'registered') AND prize_id IS NULL AND amount = 0)
            OR
            (status = 'won' AND amount >= 0 AND (prize_id IS NOT NULL OR amount = 0))
        );

CREATE INDEX IF NOT EXISTS idx_lottery_campaigns_registration_due
    ON lottery_campaigns(activity_type, draw_at)
    WHERE deleted_at IS NULL AND status = 'active';
