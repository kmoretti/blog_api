-- 005 为 friend_link 表新增字段
ALTER TABLE friend_link ADD COLUMN snapshot TEXT DEFAULT '';
ALTER TABLE friend_link ADD COLUMN friend_link_page TEXT DEFAULT '';
ALTER TABLE friend_link ADD COLUMN feed TEXT DEFAULT '';
