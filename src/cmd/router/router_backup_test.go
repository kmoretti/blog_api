package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"blog_api/src/config"
	"blog_api/src/model"
	"blog_api/src/service"
	"blog_api/src/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func backupTestToken(t *testing.T) string {
	t.Helper()
	authSvc := service.NewAuthService()
	token, _, err := authSvc.GenerateJWT("admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func setupBackupTestRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	if _, err := config.Load(); err != nil {
		t.Fatalf("load config: %v", err)
	}

	dataDir := t.TempDir()
	cfg := &model.Config{
		Data: model.DataConfig{
			Resource: model.ResourceConfig{Path: dataDir},
			Database: model.DatabaseConfig{Path: filepath.Join(dataDir, "database.db")},
		},
		Safe: model.SafeConfig{CorsAllowHostlist: []string{"http://localhost"}},
	}
	restore := config.ReplaceConfig(cfg)
	t.Cleanup(restore)

	return SetupRouter(db, cfg, time.Now())
}

func TestBackupExportRouteUnauthorized(t *testing.T) {
	router := setupBackupTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/action/backup/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestBackupExportRouteSuccess(t *testing.T) {
	router := setupBackupTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/action/backup/export", nil)
	req.Header.Set("Authorization", "Bearer "+backupTestToken(t))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("expected content type application/zip, got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected non-empty zip body")
	}
}

func TestBackupImportRouteSuccess(t *testing.T) {
	router := setupBackupTestRouter(t, nil)
	dataDir := config.GetConfig().Data.Resource.Path
	if err := os.WriteFile(filepath.Join(dataDir, "database.db"), []byte("dbdata"), 0o644); err != nil {
		t.Fatal(err)
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	fw, err := zw.Create("database.db")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("restored")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(zipBuf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/action/backup/import", &body)
	req.Header.Set("Authorization", "Bearer "+backupTestToken(t))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp model.ApiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("expected response code %d, got %d", http.StatusOK, resp.Code)
	}

	restored, err := os.ReadFile(filepath.Join(dataDir, "database.db"))
	if err != nil {
		t.Fatalf("read restored database: %v", err)
	}
	if string(restored) != "restored" {
		t.Fatalf("expected restored content 'restored', got %q", string(restored))
	}
}

func TestExportModuleRouteUnauthorized(t *testing.T) {
	router := setupBackupTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/action/export/friend_links", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestExportModuleRouteSuccess(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := db.Create(&model.FriendWebsite{Name: "Test", Link: "https://example.com"}).Error; err != nil {
		t.Fatal(err)
	}

	router := setupBackupTestRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/action/export/friend_links", nil)
	req.Header.Set("Authorization", "Bearer "+backupTestToken(t))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var env model.ExportEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Module != "friend_links" {
		t.Fatalf("expected module friend_links, got %q", env.Module)
	}
	if env.Count != 1 {
		t.Fatalf("expected count 1, got %d", env.Count)
	}
}

func TestImportModuleRouteSuccess(t *testing.T) {
	db := testutil.NewTestDB(t)
	router := setupBackupTestRouter(t, db)

	env := model.ExportEnvelope{
		Version:    "1.0",
		Module:     "friend_links",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      1,
		Items: []model.FriendWebsite{
			{Name: "Imported", Link: "https://imported.test"},
		},
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "friend_links.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/action/import/friend_links?strategy=replace", &body)
	req.Header.Set("Authorization", "Bearer "+backupTestToken(t))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp model.ApiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("expected response code %d, got %d", http.StatusOK, resp.Code)
	}
}
