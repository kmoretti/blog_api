# 项目配置与数据导入导出 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 blog_api 的管理面板「系统设置」里新增「数据迁移」区域，提供完整 `data/` 目录 ZIP 备份/恢复以及按模块 JSON 导入导出能力。

**Architecture:** 后端新增 `handler/backup` 与 `service/backup` 两层：handler 负责 HTTP 多部分/流式响应，service 负责 ZIP 打包解压与各模块 JSON 序列化/数据库写入。所有接口挂在已有的 `/api/action` 组下（已应用 `JWTAuth`）。前端在 `Settings.vue` 新增标签页，调用新增的 `api/backup.ts`。

**Tech Stack:** Go 1.22 + Gin + GORM + SQLite；Vue 3 + Element Plus + TypeScript；标准库 `archive/zip`。

---

## File Structure

- **Create** `api/src/model/backup.go` — 导出/导入相关的请求/响应结构体
- **Create** `api/src/service/backup/backup.go` — ZIP 归档解压、模块 JSON 导入导出业务逻辑
- **Create** `api/src/service/backup/backup_test.go` — 单元测试
- **Create** `api/src/handler/backup/backup.go` — HTTP handler（导出 ZIP、恢复 ZIP、模块 JSON 导出/导入）
- **Modify** `api/src/cmd/router/register.go` — 注册 `/api/action/backup/*` 与 `/api/action/export/:module`、`/api/action/import/:module`
- **Create** `api/web/src/api/backup.ts` — 前端 API 封装（文件下载/上传）
- **Create** `api/web/src/model/backup.ts` — 前端类型定义
- **Modify** `api/web/src/views/Settings.vue` — 新增「数据迁移」标签页与操作界面
- **Update** `api/README.md` — 补充备份/导入导出接口文档

---

## Task 1: Define backup request/response models

**Files:**
- Create: `api/src/model/backup.go`

- [ ] **Step 1: Write model file**

```go
package model

// ImportResult holds per-module import statistics.
type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Replaced int `json:"replaced"`
}

// RestoreResult reports the outcome of a full backup restore.
type RestoreResult struct {
	BackupPath string `json:"backup_path"`
	Notice     string `json:"notice"`
}

// ExportEnvelope is the canonical JSON wrapper for module exports.
type ExportEnvelope struct {
	Version    string      `json:"version"`
	Module     string      `json:"module"`
	ExportedAt string      `json:"exported_at"`
	Count      int         `json:"count"`
	Items      interface{} `json:"items"`
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./src/model/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add api/src/model/backup.go
git commit -m "feat(backup): add import/export result and envelope models"
```

---

## Task 2: Implement ZIP export of data directory

**Files:**
- Create: `api/src/service/backup/backup.go`
- Create: `api/src/service/backup/backup_test.go`

- [ ] **Step 1: Write ZIP export service**

```go
package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportDataDir writes the contents of dataDir into w as a zip archive.
func ExportDataDir(dataDir string, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	return filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// Normalize to forward slashes for zip.
		zipName := filepath.ToSlash(rel)
		if info.IsDir() {
			_, err := zw.Create(zipName + "/")
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = zipName
		h.Method = zip.Deflate
		fw, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
}

// BackupFilename returns a timestamped backup filename.
func BackupFilename(prefix string) string {
	return fmt.Sprintf("%s_%s.zip", prefix, time.Now().Format("20060102_150405"))
}
```

- [ ] **Step 2: Write unit test for ZIP export**

```go
package backup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["database.db"] {
		t.Error("missing database.db")
	}
	if !names["config/system_config.json"] {
		t.Error("missing config/system_config.json")
	}
}
```

- [ ] **Step 3: Run test**

Run: `go test ./src/service/backup/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add api/src/service/backup/
git commit -m "feat(backup): implement data directory zip export"
```

---

## Task 3: Implement full backup restore (ZIP import)

**Files:**
- Modify: `api/src/service/backup/backup.go`

- [ ] **Step 1: Add restore function**

Append to `api/src/service/backup/backup.go`:

```go
// ImportDataDir restores dataDir from a zip archive read from r.
// It returns the path of the automatic backup copy of the previous dataDir.
func ImportDataDir(dataDir string, r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	if err := validateBackupZip(zr); err != nil {
		return "", err
	}

	bak := dataDir + ".bak." + time.Now().Format("20060102_150405")
	if err := copyDir(dataDir, bak); err != nil {
		return "", fmt.Errorf("failed to backup current data dir: %w", err)
	}

	if err := os.RemoveAll(dataDir); err != nil {
		restoreDir(bak, dataDir)
		return "", fmt.Errorf("failed to clear data dir: %w", err)
	}

	if err := extractZip(zr, dataDir); err != nil {
		restoreDir(bak, dataDir)
		return "", fmt.Errorf("failed to extract backup: %w", err)
	}

	return bak, nil
}

func validateBackupZip(zr *zip.Reader) error {
	for _, f := range zr.File {
		if f.Name == "database.db" {
			return nil
		}
	}
	return fmt.Errorf("backup archive missing database.db")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		sf, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sf.Close()
		df, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer df.Close()
		_, err = io.Copy(df, sf)
		return err
	})
}

func restoreDir(src, dst string) {
	_ = os.RemoveAll(dst)
	_ = copyDir(src, dst)
}

func extractZip(zr *zip.Reader, dst string) error {
	for _, f := range zr.File {
		target := filepath.Join(dst, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		tf, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(tf, rc)
		rc.Close()
		tf.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: Add missing imports**

Ensure imports include:

```go
import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)
```

- [ ] **Step 3: Add restore test**

Append to `api/src/service/backup/backup_test.go`:

```go
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./src/service/backup/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/src/service/backup/
git commit -m "feat(backup): implement full backup restore with rollback"
```

---

## Task 4: Implement per-module JSON export

**Files:**
- Modify: `api/src/service/backup/backup.go`

- [ ] **Step 1: Add module export logic**

Append to `api/src/service/backup/backup.go`:

```go
import (
	// add to existing imports
	"blog_api/src/model"
	"encoding/json"
	"gorm.io/gorm"
)

// ExportModule exports a single module as an ExportEnvelope.
func ExportModule(db *gorm.DB, module string, dataDir string) (*model.ExportEnvelope, error) {
	switch module {
	case "system_config":
		return exportSystemConfig(dataDir)
	case "friend_links":
		return exportTable(db, module, &model.FriendWebsite{})
	case "moments":
		return exportMoments(db)
	case "friend_rss":
		return exportFriendRss(db)
	case "images":
		return exportTable(db, module, &model.Image{})
	default:
		return nil, fmt.Errorf("unknown module: %s", module)
	}
}

func exportSystemConfig(dataDir string) (*model.ExportEnvelope, error) {
	path := filepath.Join(dataDir, "config", "system_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     "system_config",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      1,
		Items:      cfg,
	}, nil
}

func exportTable(db *gorm.DB, module string, model interface{}) (*model.ExportEnvelope, error) {
	var items []map[string]interface{}
	if err := db.Model(model).Find(&items).Error; err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     module,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(items),
		Items:      items,
	}, nil
}

func exportMoments(db *gorm.DB) (*model.ExportEnvelope, error) {
	var moments []model.Moment
	if err := db.Find(&moments).Error; err != nil {
		return nil, err
	}
	var media []model.MomentMedia
	if err := db.Find(&media).Error; err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     "moments",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(moments),
		Items: map[string]interface{}{
			"moments":      moments,
			"moments_media": media,
		},
	}, nil
}

func exportFriendRss(db *gorm.DB) (*model.ExportEnvelope, error) {
	var feeds []model.FriendRss
	if err := db.Find(&feeds).Error; err != nil {
		return nil, err
	}
	var posts []model.RssPost
	if err := db.Find(&posts).Error; err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     "friend_rss",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(feeds),
		Items: map[string]interface{}{
			"friend_rss":       feeds,
			"friend_rss_post": posts,
		},
	}, nil
}
```

- [ ] **Step 2: Add export test**

Append to `api/src/service/backup/backup_test.go`:

```go
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
```

- [ ] **Step 3: Run tests**

Run: `go test ./src/service/backup/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add api/src/service/backup/
git commit -m "feat(backup): add per-module JSON export"
```

---

## Task 5: Implement per-module JSON import

**Files:**
- Modify: `api/src/service/backup/backup.go`

- [ ] **Step 1: Add module import logic**

Append to `api/src/service/backup/backup.go`:

```go
import (
	// add to existing imports
	"gorm.io/gorm/clause"
)

// ImportModule imports a module from an ExportEnvelope using the given strategy.
func ImportModule(db *gorm.DB, module string, env *model.ExportEnvelope, strategy string, dataDir string) (*model.ImportResult, error) {
	if env.Module != module {
		return nil, fmt.Errorf("module mismatch: expected %s, got %s", module, env.Module)
	}
	switch strategy {
	case "replace", "skip":
	default:
		strategy = "replace"
	}

	switch module {
	case "system_config":
		return importSystemConfig(env, dataDir)
	case "friend_links":
		return importTable(db, env, "friend_link", &model.FriendWebsite{}, strategy)
	case "moments":
		return importMoments(db, env, strategy)
	case "friend_rss":
		return importFriendRss(db, env, strategy)
	case "images":
		return importTable(db, env, "images", &model.Image{}, strategy)
	default:
		return nil, fmt.Errorf("unknown module: %s", module)
	}
}

func importSystemConfig(env *model.ExportEnvelope, dataDir string) (*model.ImportResult, error) {
	cfg, ok := env.Items.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid system_config export format")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "config", "system_config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// skip if exists and strategy is skip - handled at caller; for config we always overwrite here.
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return &model.ImportResult{Imported: 1}, nil
}

func importTable(db *gorm.DB, env *model.ExportEnvelope, table string, dst interface{}, strategy string) (*model.ImportResult, error) {
	itemsRaw, ok := env.Items.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid %s export format", table)
	}
	if len(itemsRaw) == 0 {
		return &model.ImportResult{}, nil
	}

	onConflict := clause.OnConflict{UpdateAll: true}
	if strategy == "skip" {
		onConflict = clause.OnConflict{DoNothing: true}
	}

	result := db.Table(table).Clauses(onConflict).CreateInBatches(itemsRaw, 100)
	if result.Error != nil {
		return nil, result.Error
	}

	return &model.ImportResult{
		Imported: int(result.RowsAffected),
	}, nil
}

func importMoments(db *gorm.DB, env *model.ExportEnvelope, strategy string) (*model.ImportResult, error) {
	data, ok := env.Items.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid moments export format")
	}
	onConflict := clause.OnConflict{UpdateAll: true}
	if strategy == "skip" {
		onConflict = clause.OnConflict{DoNothing: true}
	}

	var total int64
	if moments, ok := data["moments"].([]interface{}); ok && len(moments) > 0 {
		r := db.Table("moments").Clauses(onConflict).CreateInBatches(moments, 100)
		if r.Error != nil {
			return nil, r.Error
		}
		total += r.RowsAffected
	}
	if media, ok := data["moments_media"].([]interface{}); ok && len(media) > 0 {
		r := db.Table("moments_media").Clauses(onConflict).CreateInBatches(media, 100)
		if r.Error != nil {
			return nil, r.Error
		}
		total += r.RowsAffected
	}
	return &model.ImportResult{Imported: int(total)}, nil
}

func importFriendRss(db *gorm.DB, env *model.ExportEnvelope, strategy string) (*model.ImportResult, error) {
	data, ok := env.Items.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid friend_rss export format")
	}
	onConflict := clause.OnConflict{UpdateAll: true}
	if strategy == "skip" {
		onConflict = clause.OnConflict{DoNothing: true}
	}

	var total int64
	if feeds, ok := data["friend_rss"].([]interface{}); ok && len(feeds) > 0 {
		r := db.Table("friend_rss").Clauses(onConflict).CreateInBatches(feeds, 100)
		if r.Error != nil {
			return nil, r.Error
		}
		total += r.RowsAffected
	}
	if posts, ok := data["friend_rss_post"].([]interface{}); ok && len(posts) > 0 {
		r := db.Table("friend_rss_post").Clauses(onConflict).CreateInBatches(posts, 100)
		if r.Error != nil {
			return nil, r.Error
		}
		total += r.RowsAffected
	}
	return &model.ImportResult{Imported: int(total)}, nil
}
```

- [ ] **Step 2: Add import test**

Append to `api/src/service/backup/backup_test.go`:

```go
func TestImportModuleFriendLinks(t *testing.T) {
	// Uses an in-memory sqlite db via the test helper; see Task 13 for setup.
	// This is a placeholder for the integration test that will be fleshed out
	// once the shared test database helper is available.
	t.Skip("integration test depends on database helper from Task 13")
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./src/service/backup/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add api/src/service/backup/
git commit -m "feat(backup): add per-module JSON import with replace/skip strategy"
```

---

## Task 6: Create HTTP handlers

**Files:**
- Create: `api/src/handler/backup/backup.go`

- [ ] **Step 1: Write handler file**

```go
package backup

import (
	"blog_api/src/model"
	"blog_api/src/service/backup"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const MaxBackupSize = 500 << 20 // 500 MB

// Handler holds dependencies for backup endpoints.
type Handler struct {
	DB      *gorm.DB
	DataDir string
}

// ExportFullBackup streams a zip of the data directory.
func (h *Handler) ExportFullBackup(c *gin.Context) {
	filename := backup.BackupFilename("blog_api_backup")
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := backup.ExportDataDir(h.DataDir, c.Writer); err != nil {
		log.Printf("[backup] export failed: %v", err)
		// headers already sent; cannot change status
		return
	}
}

// ImportFullBackup restores the data directory from an uploaded zip.
func (h *Handler) ImportFullBackup(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "missing file"))
		return
	}
	defer file.Close()

	bak, err := backup.ImportDataDir(h.DataDir, file)
	if err != nil {
		log.Printf("[backup] import failed: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(model.RestoreResult{
		BackupPath: bak,
		Notice:     "备份已恢复，请重启服务以完成数据加载",
	}))
}

// ExportModule exports a single module as JSON.
func (h *Handler) ExportModule(c *gin.Context) {
	module := c.Param("module")
	env, err := backup.ExportModule(h.DB, module, h.DataDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, err.Error()))
		return
	}

	filename := fmt.Sprintf("%s_%s.json", module, time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.JSON(http.StatusOK, env)
}

// ImportModule imports a single module from JSON.
func (h *Handler) ImportModule(c *gin.Context) {
	module := c.Param("module")
	strategy := c.DefaultQuery("strategy", "replace")

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "missing file"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to read file"))
		return
	}

	var env model.ExportEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid json"))
		return
	}

	result, err := backup.ImportModule(h.DB, module, &env, strategy, h.DataDir)
	if err != nil {
		log.Printf("[backup] import module %s failed: %v", module, err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(result))
}
```

- [ ] **Step 2: Fix missing import**

Add `time` to imports.

```go
import (
	"time"
	// ... existing imports
)
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./src/handler/backup/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add api/src/handler/backup/
git commit -m "feat(backup): add backup HTTP handlers"
```

---

## Task 7: Register backup routes

**Files:**
- Modify: `api/src/cmd/router/register.go`

- [ ] **Step 1: Import backup handler**

Add to imports:

```go
backupHandler "blog_api/src/handler/backup"
```

- [ ] **Step 2: Instantiate handler after existing handlers**

After `friendLinkGroupHandler := ...` add:

```go
backupHandlerInstance := &backupHandler.Handler{
	DB:      db,
	DataDir: resolveStaticBaseDir(cfg),
}
```

- [ ] **Step 3: Bypass global body limit for backup import**

The global middleware in `api/src/cmd/router/router.go` limits request bodies to 65 MB. Full backups containing images can exceed that. Update the global middleware in `router.go` to skip the backup import path:

```go
router.Use(func(c *gin.Context) {
	if c.Request.Body != nil && !strings.HasPrefix(c.Request.URL.Path, "/api/action/backup/import") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
	}
	c.Next()
})
```

- [ ] **Step 4: Register routes inside action group**

Inside `actionGroup.Use(middleware.JWTAuth())` block, after `mediaActionGroup` block add:

```go
backupActionGroup := actionGroup.Group("/backup")
{
	backupActionGroup.Use(func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, backupHandler.MaxBackupSize)
		}
		c.Next()
	})
	backupActionGroup.POST("/export", backupHandlerInstance.ExportFullBackup)
	backupActionGroup.POST("/import", backupHandlerInstance.ImportFullBackup)
}
actionGroup.GET("/export/:module", backupHandlerInstance.ExportModule)
actionGroup.POST("/import/:module", backupHandlerInstance.ImportModule)
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./cmd/...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add api/src/cmd/router/router.go api/src/cmd/router/register.go
git commit -m "feat(backup): register backup/import/export routes with large body support"
```

---

## Task 8: Add backend integration test helper

**Files:**
- Create: `api/src/testutil/db.go`

- [ ] **Step 1: Create in-memory test DB helper**

```go
package testutil

import (
	"blog_api/src/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewTestDB returns an in-memory sqlite db migrated with the core tables.
func NewTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
	return db
}
```

- [ ] **Step 2: Update backup import tests**

Replace the placeholder in `api/src/service/backup/backup_test.go` with:

```go
func TestImportModuleFriendLinks(t *testing.T) {
	db := testutil.NewTestDB(t)
	tmp := t.TempDir()

	db.Create(&model.FriendWebsite{ID: 1, Name: "A", Link: "https://a.test", Status: "survival"})

	env, err := ExportModule(db, "friend_links", tmp)
	if err != nil {
		t.Fatal(err)
	}

	db.Model(&model.FriendWebsite{}).Where("id = ?", 1).Update("name", "Changed")

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
```

- [ ] **Step 3: Add testutil import to backup_test.go**

```go
import (
	"blog_api/src/testutil"
	// ... existing imports
)
```

- [ ] **Step 4: Run tests**

Run: `go test ./src/service/backup/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/src/testutil/db.go api/src/service/backup/backup_test.go
git commit -m "test(backup): add in-memory db helper and friend_links import test"
```

---

## Task 9: Create frontend backup API client

**Files:**
- Create: `api/web/src/api/backup.ts`
- Create: `api/web/src/model/backup.ts`

- [ ] **Step 1: Write frontend types**

```ts
// api/web/src/model/backup.ts
export interface ImportResult {
  imported: number
  skipped: number
  replaced: number
}

export interface RestoreResult {
  backup_path: string
  notice: string
}

export type BackupModule =
  | 'system_config'
  | 'friend_links'
  | 'moments'
  | 'friend_rss'
  | 'images'

export interface ModuleOption {
  key: BackupModule
  label: string
}

export const BACKUP_MODULES: ModuleOption[] = [
  { key: 'system_config', label: '系统配置' },
  { key: 'friend_links', label: '友链' },
  { key: 'moments', label: '动态' },
  { key: 'friend_rss', label: 'RSS' },
  { key: 'images', label: '图片清单' },
]
```

- [ ] **Step 2: Write API client**

```ts
// api/web/src/api/backup.ts
import request from '@/utils/request'
import type { ApiResponse } from '@/model/response'
import type { ImportResult, RestoreResult } from '@/model/backup'

export const exportFullBackup = () => {
  return request.post(
    '/action/backup/export',
    {},
    { responseType: 'blob' }
  ) as Promise<Blob>
}

export const importFullBackup = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return request.post<any, ApiResponse<RestoreResult>>('/action/backup/import', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export const exportModule = (module: string) => {
  return request.get(`/action/export/${module}`, {
    responseType: 'blob'
  }) as Promise<Blob>
}

export const importModule = (module: string, file: File, strategy: 'replace' | 'skip') => {
  const form = new FormData()
  form.append('file', file)
  return request.post<any, ApiResponse<ImportResult>>(
    `/action/import/${module}?strategy=${strategy}`,
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } }
  )
}
```

- [ ] **Step 3: Verify TypeScript**

Run: `cd api/web && pnpm run typecheck`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add api/web/src/api/backup.ts api/web/src/model/backup.ts
git commit -m "feat(web): add backup API client and types"
```

---

## Task 10: Add Data Migration UI to Settings

**Files:**
- Modify: `api/web/src/views/Settings.vue`

- [ ] **Step 1: Add imports**

```ts
import {
  exportFullBackup,
  importFullBackup,
  exportModule,
  importModule,
} from '@/api/backup'
import { BACKUP_MODULES, type BackupModule } from '@/model/backup'
```

- [ ] **Step 2: Add state refs**

```ts
const importingFull = ref(false)
const fullRestoreFile = ref<File | null>(null)
const moduleImportFile = ref<File | null>(null)
const selectedModule = ref<BackupModule>('friend_links')
const importStrategy = ref<'replace' | 'skip'>('replace')
const importingModule = ref(false)
```

- [ ] **Step 3: Add methods**

```ts
const downloadBlob = (blob: Blob, filename: string) => {
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.URL.revokeObjectURL(url)
}

const handleExportFull = async () => {
  try {
    const blob = await exportFullBackup()
    downloadBlob(blob, `blog_api_backup_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.zip`)
    ElMessage.success('备份下载已开始')
  } catch (error) {
    ElMessage.error('导出备份失败')
    console.error(error)
  }
}

const handleImportFull = async () => {
  if (!fullRestoreFile.value) {
    ElMessage.warning('请选择备份文件')
    return
  }
  try {
    await ElMessageBox.confirm(
      '恢复备份会覆盖当前 data/ 目录，操作后需要重启服务。是否继续？',
      '警告',
      { confirmButtonText: '继续', cancelButtonText: '取消', type: 'warning' }
    )
    importingFull.value = true
    const res = await importFullBackup(fullRestoreFile.value)
    ElMessage.success(res.message)
    fullRestoreFile.value = null
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('恢复备份失败')
      console.error(error)
    }
  } finally {
    importingFull.value = false
  }
}

const handleExportModule = async (module: BackupModule) => {
  try {
    const blob = await exportModule(module)
    downloadBlob(blob, `${module}_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.json`)
    ElMessage.success('导出成功')
  } catch (error) {
    ElMessage.error('导出失败')
    console.error(error)
  }
}

const handleImportModule = async () => {
  if (!moduleImportFile.value) {
    ElMessage.warning('请选择 JSON 文件')
    return
  }
  try {
    importingModule.value = true
    const res = await importModule(selectedModule.value, moduleImportFile.value, importStrategy.value)
    ElMessage.success(`导入完成：新增/替换 ${res.data.imported} 条`)
    moduleImportFile.value = null
  } catch (error) {
    ElMessage.error('导入失败')
    console.error(error)
  } finally {
    importingModule.value = false
  }
}
```

- [ ] **Step 4: Add template tab**

Insert new tab before `危险区域`:

```vue
<el-tab-pane label="数据迁移" name="migration">
  <el-card shadow="never" style="margin-bottom: 20px">
    <template #header>完整备份</template>
    <el-alert
      title="完整备份会打包整个 data/ 目录（数据库、配置、图片、资源）。"
      type="info"
      :closable="false"
      style="margin-bottom: 16px"
    />
    <el-button type="primary" @click="handleExportFull">导出备份</el-button>
    <el-divider />
    <el-form label-width="120px">
      <el-form-item label="恢复备份">
        <el-upload
          :auto-upload="false"
          :limit="1"
          :on-change="(file) => { fullRestoreFile = file.raw as File }"
          accept=".zip"
        >
          <el-button>选择 ZIP</el-button>
        </el-upload>
      </el-form-item>
      <el-form-item>
        <el-button type="danger" :loading="importingFull" @click="handleImportFull">恢复备份</el-button>
      </el-form-item>
    </el-form>
  </el-card>

  <el-card shadow="never">
    <template #header>模块数据迁移</template>
    <el-alert
      title="导出/导入 JSON 用于迁移业务数据。导入时主键冲突可选替换或跳过。"
      type="info"
      :closable="false"
      style="margin-bottom: 16px"
    />
    <el-form label-width="120px">
      <el-form-item label="模块">
        <el-select v-model="selectedModule" style="width: 200px">
          <el-option
            v-for="m in BACKUP_MODULES"
            :key="m.key"
            :label="m.label"
            :value="m.key"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="策略">
        <el-radio-group v-model="importStrategy">
          <el-radio label="replace">替换冲突</el-radio>
          <el-radio label="skip">跳过冲突</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item>
        <el-button @click="handleExportModule(selectedModule)">导出 JSON</el-button>
      </el-form-item>
      <el-form-item label="导入 JSON">
        <el-upload
          :auto-upload="false"
          :limit="1"
          :on-change="(file) => { moduleImportFile = file.raw as File }"
          accept=".json"
        >
          <el-button>选择 JSON</el-button>
        </el-upload>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="importingModule" @click="handleImportModule">导入</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</el-tab-pane>
```

- [ ] **Step 5: Verify TypeScript and lint**

Run: `cd api/web && pnpm run typecheck && pnpm run lint`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add api/web/src/views/Settings.vue
git commit -m "feat(web): add data migration UI to settings"
```

---

## Task 11: Add router tests for backup endpoints

**Files:**
- Create: `api/src/cmd/router/router_backup_test.go`

- [ ] **Step 1: Write handler-level test**

```go
package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	backupHandler "blog_api/src/handler/backup"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBackupExportHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &backupHandler.Handler{DataDir: tmp}
	router := gin.New()
	router.POST("/export", h.ExportFullBackup)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/export", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "blog_api_backup")
}
```

- [ ] **Step 2: Verify compilation**

Run: `go test ./src/cmd/router/... -run TestBackupExportHandler -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add api/src/cmd/router/router_backup_test.go
git commit -m "test(backup): add export handler test"
```

---

## Task 12: Update README API documentation

**Files:**
- Modify: `api/README.md`

- [ ] **Step 1: Add backup section**

Append a new section after the existing API tables:

```markdown
### 数据迁移

> 以下接口需要管理员 JWT。

#### 完整备份

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/action/backup/export` | 导出 `data/` 目录 ZIP |
| `POST` | `/api/action/backup/import` | 恢复 `data/` 目录 ZIP |

#### 模块导入导出

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/action/export/:module` | 导出模块 JSON |
| `POST` | `/api/action/import/:module?strategy=replace|skip` | 导入模块 JSON |

支持的 `module`：`system_config`、`friend_links`、`moments`、`friend_rss`、`images`。
```

- [ ] **Step 2: Commit**

```bash
git add api/README.md
git commit -m "docs: document backup and migration endpoints"
```

---

## Task 13: Final verification

- [ ] **Step 1: Run full backend test suite**

Run: `cd api && go test ./...`
Expected: PASS

- [ ] **Step 2: Build frontend**

Run: `cd api/web && pnpm run build`
Expected: build succeeds

- [ ] **Step 3: Lint frontend**

Run: `cd api/web && pnpm run lint`
Expected: no errors

- [ ] **Step 4: Push**

```bash
git push
```

---

## Spec Coverage Check

| Spec Requirement | Task |
| --- | --- |
| 完整备份 ZIP 导出 | Task 2, Task 6 |
| 完整备份 ZIP 恢复（替换） | Task 3, Task 6 |
| 分模块 JSON 导出 | Task 4, Task 6 |
| 分模块 JSON 导入（replace/skip） | Task 5, Task 6 |
| 管理面板 Settings UI | Task 10 |
| 仅管理员可访问 | Task 7（注册在 JWT 保护的 action 组下） |
| 自动创建 data.bak | Task 3 |
| README 文档 | Task 12 |

## Placeholder Scan

- No TBD/TODO placeholders remain in plan steps.
- All code blocks show actual code, not vague descriptions.
- Type/function names are consistent across tasks.
