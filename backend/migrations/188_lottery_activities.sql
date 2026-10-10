CREATE TABLE IF NOT EXISTS lottery_campaigns (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(120) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    win_rate DECIMAL(7,4) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lottery_campaigns_win_rate_check CHECK (win_rate >= 0 AND win_rate <= 100),
    CONSTRAINT lottery_campaigns_status_check CHECK (status IN ('active', 'stopped', 'exhausted'))
);

CREATE TABLE IF NOT EXISTS lottery_prizes (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES lottery_campaigns(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    min_amount DECIMAL(20,8) NOT NULL,
    max_amount DECIMAL(20,8) NOT NULL,
    total_quantity INT NOT NULL,
    remaining_quantity INT NOT NULL,
    weight INT NOT NULL DEFAULT 1,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lottery_prizes_amount_check CHECK (min_amount >= 0 AND max_amount >= min_amount),
    CONSTRAINT lottery_prizes_quantity_check CHECK (
        total_quantity > 0 AND remaining_quantity >= 0 AND remaining_quantity <= total_quantity
    ),
    CONSTRAINT lottery_prizes_weight_check CHECK (weight > 0)
);

CREATE TABLE IF NOT EXISTS lottery_draws (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES lottery_campaigns(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    prize_id BIGINT REFERENCES lottery_prizes(id) ON DELETE SET NULL,
    amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lottery_draws_status_check CHECK (status IN ('won', 'lost')),
    CONSTRAINT lottery_draws_user_once UNIQUE (campaign_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_lottery_campaigns_status_expires
    ON lottery_campaigns(status, expires_at);
CREATE INDEX IF NOT EXISTS idx_lottery_prizes_campaign_remaining
    ON lottery_prizes(campaign_id, remaining_quantity);
CREATE INDEX IF NOT EXISTS idx_lottery_draws_campaign_created
    ON lottery_draws(campaign_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lottery_draws_campaign_winners
    ON lottery_draws(campaign_id, status, created_at DESC);
