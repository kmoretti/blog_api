-- 006 为 moments 表新增 tags、pinned_order、is_ad 字段
ALTER TABLE moments ADD COLUMN tags TEXT DEFAULT '';
ALTER TABLE moments ADD COLUMN pinned_order INTEGER DEFAULT 0;
ALTER TABLE moments ADD COLUMN is_ad INTEGER DEFAULT 0;
