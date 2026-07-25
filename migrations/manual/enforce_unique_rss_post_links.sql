-- Ensure the target table exists before running the deduplication.
-- This makes the migration safe to run on databases that have not yet
-- created friend_rss_post (e.g. during incremental upgrades).
CREATE TABLE IF NOT EXISTS friend_rss_post (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rss_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    author TEXT,
    link TEXT NOT NULL,
    description TEXT NOT NULL,
    time INTEGER NOT NULL
);

-- Keep the oldest row for links duplicated by a legacy database.
DELETE FROM friend_rss_post
WHERE id NOT IN (
  SELECT MIN(id)
  FROM friend_rss_post
  GROUP BY link
);

-- Legacy databases created this as a non-unique index. Rebuild it so
-- INSERT ... ON CONFLICT(link) matches a unique constraint.
DROP INDEX IF EXISTS idx_friend_rss_post_link;
CREATE UNIQUE INDEX idx_friend_rss_post_link ON friend_rss_post(link);
