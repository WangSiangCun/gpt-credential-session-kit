-- Backfill the cumulative recharge cache introduced by migration 102.
-- Historical balance/admin-balance redemptions predate that cache and must
-- remain eligible for recharge-gated activities.
WITH historical_recharges AS (
  SELECT
    used_by AS user_id,
    SUM(value) AS total_recharged
  FROM redeem_codes
  WHERE used_by IS NOT NULL
    AND value > 0
    AND type IN ('balance', 'admin_balance')
  GROUP BY used_by
)
UPDATE users u
SET total_recharged = GREATEST(u.total_recharged, h.total_recharged),
    updated_at = NOW()
FROM historical_recharges h
WHERE u.id = h.user_id
  AND u.total_recharged < h.total_recharged;
