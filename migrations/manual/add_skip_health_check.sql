-- Upgrade existing friend_link tables that predate skip_health_check.
-- Adds the column if missing, then builds its index. Idempotent: re-runs are
-- safe because duplicate-column errors are swallowed by the migration runner
-- and CREATE INDEX IF NOT EXISTS is a no-op once the index exists.

ALTER TABLE friend_link ADD COLUMN skip_health_check BOOLEAN NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_friend_link_skip_health_check
ON friend_link (skip_health_check);
