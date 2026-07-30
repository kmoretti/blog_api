package testutil

import (
	"blog_api/src/model"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewTestDB returns a temporary-file sqlite db migrated with the core tables:
//   friend_link, friend_rss, friend_rss_post, moments, moments_media, images.
func NewTestDB(t *testing.T) *gorm.DB {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.FriendWebsite{},
		&model.FriendRss{},
		&model.RssPost{},
		&model.Moment{},
		&model.MomentMedia{},
		&model.Image{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return db
}
