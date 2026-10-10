-- Speed up the user-facing channel request timeline.
-- This is intentionally a concurrent, additive index: it does not modify or
-- remove usage data and does not block normal reads/writes for the full build.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_channel_created_id
  ON usage_logs (channel_id, created_at DESC, id DESC)
  WHERE channel_id IS NOT NULL;
