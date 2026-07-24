-- 创建友链表
CREATE TABLE IF NOT EXISTS friend_link (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  website_url TEXT NOT NULL,
  website_name TEXT NOT NULL,
  website_icon_url TEXT ,
  description TEXT NOT NULL,
  email TEXT,
  times INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'survival' CHECK ( status IN (
    'survival',
    'timeout',
    'error',
    'died',
    'pending'
  )),
  is_died BOOLEAN NOT NULL DEFAULT 0,
  enable_rss BOOLEAN NOT NULL DEFAULT 1,
  skip_health_check BOOLEAN NOT NULL DEFAULT 0 CHECK (skip_health_check IN (0, 1)),
  updated_at INTEGER NOT NULL DEFAULT 0
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_friend_link_status ON friend_link (status);
CREATE INDEX IF NOT EXISTS idx_friend_link_website_url ON friend_link (website_url);
CREATE INDEX IF NOT EXISTS idx_friend_link_email ON friend_link (email);
CREATE INDEX IF NOT EXISTS idx_friend_link_skip_health_check ON friend_link (skip_health_check);

-- 为 friend_link 表创建触发器, 用于自动更新 updated_at
CREATE TRIGGER IF NOT EXISTS trg_friend_link_updated_at
AFTER UPDATE ON friend_link
FOR EACH ROW
BEGIN
  UPDATE friend_link SET updated_at = strftime('%s','now') WHERE id = OLD.id;
END;
