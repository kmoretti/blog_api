CREATE TABLE IF NOT EXISTS friend_rss (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    friend_link_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    rss_url TEXT NOT NULL,
    times INTEGER NOT NULL DEFAULT 0,
    is_died BOOLEAN NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'survival' CHECK ( status IN (
    'survival',
    'timeout',
    'error',
    'pause'  
  )),
  updated_at INTEGER NOT NULL DEFAULT 0
);

-- 创建索引优化
CREATE INDEX IF NOT EXISTS idx_friend_rss_friend_link_id ON friend_rss(friend_link_id);
CREATE INDEX IF NOT EXISTS idx_friend_rss_status ON friend_rss(status);
CREATE INDEX IF NOT EXISTS idx_friend_rss_times ON friend_rss(times);
CREATE INDEX IF NOT EXISTS idx_friend_rss_is_died ON friend_rss(is_died);

-- 为 friend_rss 表创建触发器, 用于自动更新 updated_at
CREATE TRIGGER IF NOT EXISTS trg_friend_rss_updated_at
AFTER UPDATE ON friend_rss
FOR EACH ROW
BEGIN
  UPDATE friend_rss SET updated_at = strftime('%s','now') WHERE id = OLD.id;
END;
