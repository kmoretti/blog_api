-- 009 创建友链分组相关表并扩展 friend_link 字段

-- 友链分组表
CREATE TABLE IF NOT EXISTS friend_link_group (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_friend_link_group_sort_order ON friend_link_group (sort_order);

-- 友链与分组的关联表
CREATE TABLE IF NOT EXISTS friend_link_group_mapping (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  friend_link_id INTEGER NOT NULL,
  friend_link_group_id INTEGER NOT NULL,
  FOREIGN KEY (friend_link_id) REFERENCES friend_link(id) ON DELETE CASCADE,
  FOREIGN KEY (friend_link_group_id) REFERENCES friend_link_group(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_friend_link_group_mapping_link_id ON friend_link_group_mapping (friend_link_id);
CREATE INDEX IF NOT EXISTS idx_friend_link_group_mapping_group_id ON friend_link_group_mapping (friend_link_group_id);

-- 扩展 friend_link 字段
ALTER TABLE friend_link ADD COLUMN color TEXT DEFAULT '';
ALTER TABLE friend_link ADD COLUMN rss TEXT DEFAULT '';
ALTER TABLE friend_link ADD COLUMN tags TEXT DEFAULT '[]';
