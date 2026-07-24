package repositories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// oldFriendLinkSchema mirrors migrations/001_01 before skip_health_check was added.
const oldFriendLinkSchema = `
CREATE TABLE friend_link (
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
  updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_friend_link_status ON friend_link (status);
CREATE INDEX idx_friend_link_website_url ON friend_link (website_url);
CREATE INDEX idx_friend_link_email ON friend_link (email);
CREATE TRIGGER trg_friend_link_updated_at
AFTER UPDATE ON friend_link
FOR EACH ROW
BEGIN
  UPDATE friend_link SET updated_at = strftime('%s','now') WHERE id = OLD.id;
END;
INSERT INTO friend_link (website_url, website_name, description)
VALUES ('https://example.com', 'Example', 'Seed row');
`

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:upgrade_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

// chdir changes the working directory for the duration of the test and
// restores it on cleanup.
func chdir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
}

// TestUpgradesPreSchemaToSkipHealthCheck reproduces the production bug:
// a database created with the pre-skip_health_check schema must be
// upgraded by migrations/manual/add_skip_health_check.sql without
// "no such column: skip_health_check" errors.
func TestUpgradesPreSchemaToSkipHealthCheck(t *testing.T) {
	dir := t.TempDir()
	_ = dir
	t.Setenv("BLOG_API_MIGRATIONS_ROOT", filepath.Join("..", ".."))
	// Run migration collection relative to repo root.
	migrationsRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve migrations root: %v", err)
	}
	// collectMigrationFiles uses filepath.Glob with relative paths rooted at
	// the working directory. We chdir to the repo root for the test so the
	// glob patterns resolve to the real migrations directory.
	chdir(t, migrationsRoot)

	files, err := collectMigrationFiles()
	if err != nil {
		t.Fatalf("collect migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no migration files collected from %s", migrationsRoot)
	}

	db := openSQLite(t)
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	// Seed a database using the old schema that predates skip_health_check.
	if err := db.Exec(oldFriendLinkSchema).Error; err != nil {
		t.Fatalf("seed old schema: %v", err)
	}

	// Mark 001_01, 001_02, 001_03 as already applied (they created the
	// legacy schema). The migrator must not re-run them; the manual
	// migration handles the column add.
	preapplied := []string{
		"001_01_create_frined_link.sql",
		"001_02_create_friend_rss.sql",
		"001_03_create_rss_post.sql",
		"002_01_create_image_repo.sql",
		"003_01_create_moments_table.sql",
		"003_02_create_moments_media.sql",
		"003_03_create_moments_reaction.sql",
		"004_create_fingerprints.sql",
		"005_alter_friend_link_add_fields.sql",
		"005_optimize_query_indexes.sql",
		"006_alter_moments_add_tags_pinned_ad.sql",
		"007_alter_moments_add_extension.sql",
	}

	if err := db.Exec(`
		CREATE TABLE schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)`).Error; err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, name := range preapplied {
		var exists int64
		if err := db.Raw("SELECT count(*) FROM schema_migrations WHERE name = ?", name).Scan(&exists).Error; err != nil {
			t.Fatalf("check preapplied %s: %v", name, err)
		}
		if exists > 0 {
			continue
		}
		if err := db.Exec("INSERT INTO schema_migrations (name, applied_at) VALUES (?, 0)", name).Error; err != nil {
			t.Fatalf("mark %s applied: %v", name, err)
		}
	}

	// Run the migrations. The manual migration add_skip_health_check.sql
	// should add the column; the CREATE INDEX IF NOT EXISTS in 001_01's
	// new shape should not reach into a non-existent column.
	for _, file := range files {
		name := filepath.Base(file)
		var applied int64
		if err := db.Raw("SELECT count(*) FROM schema_migrations WHERE name = ?", name).Scan(&applied).Error; err != nil {
			t.Fatalf("check migration %s: %v", file, err)
		}
		if applied > 0 {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		statements := splitSQLStatements(string(content))
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, stmt := range statements {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return tx.Exec("INSERT INTO schema_migrations (name, applied_at) VALUES (?, 0)", name).Error
		}); err != nil {
			t.Fatalf("apply %s: %v", file, err)
		}
	}

	// Verify the column now exists and the seeded row survived.
	var count int64
	if err := db.Raw("SELECT count(*) FROM friend_link").Scan(&count).Error; err != nil {
		t.Fatalf("count friend_link after upgrade: %v", err)
	}
	if count != 1 {
		t.Fatalf("seeded row lost during upgrade: have %d rows, want 1", count)
	}

	// Verify querying the new column works.
	var skip bool
	if err := db.Raw("SELECT skip_health_check FROM friend_link LIMIT 1").Scan(&skip).Error; err != nil {
		t.Fatalf("select skip_health_check after upgrade: %v", err)
	}
	if skip {
		t.Fatalf("skip_health_check default is %v, want false", skip)
	}
}
