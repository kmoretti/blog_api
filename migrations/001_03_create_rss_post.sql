CREATE TABLE IF NOT EXISTS friend_rss_post (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rss_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    author TEXT,
    link TEXT NOT NULL,
    description TEXT NOT NULL,
    time INTEGER NOT NULL,

    -- 级联
    FOREIGN KEY (rss_id) REFERENCES friend_rss(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_friend_rss_post_link ON friend_rss_post(link);
CREATE INDEX IF NOT EXISTS idx_friend_rss_post_rss_id_time ON friend_rss_post(rss_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_friend_rss_post_time ON friend_rss_post(time DESC);
