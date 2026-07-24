-- Cover the filters and ordering used by paginated API queries.
CREATE INDEX IF NOT EXISTS idx_friend_link_status_updated_id
ON friend_link(status, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_friend_link_health_updated_id
ON friend_link(is_died, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_friend_rss_link_status_died_updated_id
ON friend_rss(friend_link_id, status, is_died, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_friend_rss_status_died_updated_id
ON friend_rss(status, is_died, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_images_status_id
ON images(status, id DESC);

CREATE INDEX IF NOT EXISTS idx_moments_status_created_id
ON moments(status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_moments_created_id
ON moments(created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_moments_media_moment_deleted_id
ON moments_media(moment_id, is_deleted, id);

CREATE INDEX IF NOT EXISTS idx_moments_media_deleted_type_id
ON moments_media(is_deleted, media_type, id DESC);
