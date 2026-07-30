-- 010 修复 friend_link 表中 tags 字段的非法 JSON
-- SQLite may leave existing rows with NULL after ALTER TABLE ADD COLUMN,
-- and early map-based Updates could store non-JSON text. Normalize these
-- values to a valid JSON empty array.

UPDATE friend_link
SET tags = '[]'
WHERE tags IS NULL
   OR tags = ''
   OR (tags NOT LIKE '[%' AND tags NOT LIKE '"%')
   OR tags = 'null';
