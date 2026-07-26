-- 为 friend_link 表添加拒绝理由字段，用于友链审核通知
ALTER TABLE friend_link ADD COLUMN rejection_reason TEXT DEFAULT '';
