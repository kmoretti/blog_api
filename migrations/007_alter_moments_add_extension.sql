-- 007 为 moments 表新增 extension 字段（JSON 格式的扩展卡片数据）
ALTER TABLE moments ADD COLUMN extension TEXT DEFAULT NULL;
