package backup

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"blog_api/src/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExportDataDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("dbdata"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(tmp, "config")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "system_config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := ExportDataDir(tmp, &buf); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.File) != 2 {
		t.Fatalf("expected 2 files, got %d", len(r.File))
	}

	want := map[string]string{
		"database.db":               "dbdata",
		"config/system_config.json": "{}",
	}
	got := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		got[f.Name] = string(data)
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("%s: expected %q, got %q", name, content, got[name])
		}
	}
}

func TestImportDataDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Build a backup zip in memory.
	var buf bytes.Buffer
	if err := ExportDataDir(tmp, &buf); err != nil {
		t.Fatal(err)
	}

	// Mutate the original dir.
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	bak, err := ImportDataDir(tmp, &buf)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "database.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("expected restored content 'old', got %q", string(data))
	}
	if _, err := os.Stat(bak); err != nil {
		t.Fatalf("backup dir missing: %v", err)
	}
}

func TestValidateBackupZip_MissingDatabase(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create("other.txt"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	if err := validateBackupZip(zr); err == nil {
		t.Fatal("expected error for archive missing database.db")
	}
}

func TestExtractZip_ZipSlip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create("../evil.txt"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := extractZip(zr, dst); err == nil {
		t.Fatal("expected zip-slip entry to be rejected")
	}

	evil := filepath.Clean(filepath.Join(dst, "..", "evil.txt"))
	if _, err := os.Stat(evil); !os.IsNotExist(err) {
		t.Fatalf("zip-slip file was created outside destination: %v", err)
	}
}

func TestExportModuleSystemConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "system_config.json"), []byte(`{"site":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	env, err := ExportModule(nil, "system_config", tmp)
	if err != nil {
		t.Fatal(err)
	}
	if env.Module != "system_config" {
		t.Fatalf("expected module system_config, got %s", env.Module)
	}
}

func newTestDB(t *testing.T) *gorm.DB {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FriendWebsite{}, &model.Moment{}, &model.MomentMedia{}, &model.FriendRss{}, &model.RssPost{}, &model.Image{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestExportModuleFriendLinksJSONTags(t *testing.T) {
	db := newTestDB(t)
	link := model.FriendWebsite{
		Name:   "Test",
		Link:   "https://example.com",
		Avatar: "https://example.com/avatar.png",
		Tags:   []string{"blog", "test"},
	}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	env, err := ExportModule(db, "friend_links", "")
	if err != nil {
		t.Fatal(err)
	}
	if env.Module != "friend_links" {
		t.Fatalf("expected module friend_links, got %s", env.Module)
	}
	if env.Count != 1 {
		t.Fatalf("expected count 1, got %d", env.Count)
	}

	data, err := json.Marshal(env.Items)
	if err != nil {
		t.Fatal(err)
	}
	jsonStr := string(data)
	for _, field := range []string{"website_name", "website_url", "website_icon_url"} {
		if strings.Contains(jsonStr, field) {
			t.Errorf("exported JSON should not contain db column name %q", field)
		}
	}
	for _, field := range []string{`"name"`, `"link"`, `"avatar"`, `"tags"`} {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("exported JSON should contain JSON-tagged field %q", field)
		}
	}
}

func TestExportModuleMoments(t *testing.T) {
	db := newTestDB(t)
	moment := model.Moment{Content: "hello"}
	if err := db.Create(&moment).Error; err != nil {
		t.Fatal(err)
	}
	media := model.MomentMedia{MomentID: moment.ID, MediaURL: "https://example.com/img.png", MediaType: "image"}
	if err := db.Create(&media).Error; err != nil {
		t.Fatal(err)
	}

	env, err := ExportModule(db, "moments", "")
	if err != nil {
		t.Fatal(err)
	}
	if env.Module != "moments" {
		t.Fatalf("expected module moments, got %s", env.Module)
	}
	if env.Count != 1 {
		t.Fatalf("expected count 1, got %d", env.Count)
	}

	items, ok := env.Items.(map[string]interface{})
	if !ok {
		t.Fatalf("expected items to be map, got %T", env.Items)
	}
	if _, ok := items["moments"]; !ok {
		t.Error("expected items to contain moments")
	}
	if _, ok := items["moments_media"]; !ok {
		t.Error("expected items to contain moments_media")
	}
}

func TestExportModuleFriendRss(t *testing.T) {
	db := newTestDB(t)
	feed := model.FriendRss{Name: "Test Feed", RssURL: "https://example.com/feed.xml"}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatal(err)
	}
	post := model.RssPost{RssID: feed.ID, Title: "Post"}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}

	env, err := ExportModule(db, "friend_rss", "")
	if err != nil {
		t.Fatal(err)
	}
	if env.Module != "friend_rss" {
		t.Fatalf("expected module friend_rss, got %s", env.Module)
	}
	if env.Count != 1 {
		t.Fatalf("expected count 1, got %d", env.Count)
	}

	items, ok := env.Items.(map[string]interface{})
	if !ok {
		t.Fatalf("expected items to be map, got %T", env.Items)
	}
	if _, ok := items["friend_rss"]; !ok {
		t.Error("expected items to contain friend_rss")
	}
	if _, ok := items["friend_rss_post"]; !ok {
		t.Error("expected items to contain friend_rss_post")
	}
}

func TestImportModuleFriendLinks(t *testing.T) {
	db := newTestDB(t)
	tmp := t.TempDir()

	// Insert initial link
	db.Create(&model.FriendWebsite{Name: "A", Link: "https://a.test", Status: "survival"})

	// Export
	env, err := ExportModule(db, "friend_links", tmp)
	if err != nil {
		t.Fatal(err)
	}

	// Mutate
	db.Model(&model.FriendWebsite{}).Where("id = ?", 1).Update("website_name", "Changed")

	// Import with replace
	res, err := ImportModule(db, "friend_links", env, "replace", tmp)
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 1 {
		t.Fatalf("expected 1 imported, got %d", res.Imported)
	}

	var link model.FriendWebsite
	if err := db.First(&link, 1).Error; err != nil {
		t.Fatal(err)
	}
	if link.Name != "A" {
		t.Fatalf("expected name A after replace, got %s", link.Name)
	}
}
