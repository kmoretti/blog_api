BEGIN IMMEDIATE;

CREATE TABLE friend_link_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  website_url TEXT NOT NULL,
  website_name TEXT NOT NULL,
  website_icon_url TEXT,
  description TEXT NOT NULL,
  email TEXT,
  times INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'survival' CHECK (status IN (
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

INSERT INTO friend_link_new (
  id,
  website_url,
  website_name,
  website_icon_url,
  description,
  email,
  times,
  status,
  is_died,
  enable_rss,
  skip_health_check,
  updated_at
)
SELECT
  id,
  website_url,
  website_name,
  website_icon_url,
  description,
  email,
  times,
  CASE status WHEN 'ignored' THEN 'pending' ELSE status END,
  is_died,
  enable_rss,
  0,
  updated_at
FROM friend_link;

DROP TABLE friend_link;
ALTER TABLE friend_link_new RENAME TO friend_link;

CREATE INDEX idx_friend_link_status ON friend_link (status);
CREATE INDEX idx_friend_link_website_url ON friend_link (website_url);
CREATE INDEX idx_friend_link_email ON friend_link (email);
CREATE INDEX idx_friend_link_skip_health_check ON friend_link (skip_health_check);

CREATE TRIGGER trg_friend_link_updated_at
AFTER UPDATE ON friend_link
FOR EACH ROW
BEGIN
  UPDATE friend_link SET updated_at = strftime('%s','now') WHERE id = OLD.id;
END;

COMMIT;
