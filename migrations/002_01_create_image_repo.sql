CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    url TEXT NOT NULL UNIQUE,
    local_path TEXT,
    is_local INTEGER NOT NULL DEFAULT 0,
    is_oss INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'normal' CHECK (status IN (
        'normal',
        'pause',
        'broken',
        'pending'
    )),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);

-- 为 images 表创建触发器, 用于自动更新 updated_at
CREATE TRIGGER IF NOT EXISTS trg_images_updated_at
AFTER UPDATE ON images
FOR EACH ROW
BEGIN
  UPDATE images SET updated_at = strftime('%s','now') WHERE id = OLD.id;
END;
