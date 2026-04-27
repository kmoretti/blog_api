package authHandler

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"blog_api/src/service"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateFingerprintPreservesTokenAndUpdatesIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Setenv("FINGERPRINT_SECRET", "test-secret")
	if _, err := config.Load(); err != nil {
		t.Fatalf("load config: %v", err)
	}

	db := openFingerprintTestDB(t)

	record := &model.Fingerprint{
		Fingerprint:      hashFingerprint("198.51.100.10", "old-agent", "test-secret"),
		UserAgent:        "old-agent",
		IP:               "198.51.100.10",
		PermissionsLevel: "normal",
		CreatedAt:        time.Now().Unix(),
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("create fingerprint: %v", err)
	}

	tokenService := service.NewFingerprintTokenService("test-secret")
	token := tokenService.Sign(record.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/verify/fingerprint", nil)
	req.RemoteAddr = "203.0.113.8:1234"
	req.Header.Set("User-Agent", "new-agent")
	req.Header.Set("X-Fingerprint-Token", token)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	handler := NewFingerprintHandler(db)
	handler.CreateFingerprint(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var resp model.ApiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected response data map, got %T", resp.Data)
	}

	returnedToken, ok := data["fingerprint_token"].(string)
	if !ok {
		t.Fatalf("expected fingerprint_token string, got %T", data["fingerprint_token"])
	}
	if returnedToken != token {
		t.Fatalf("expected token %q, got %q", token, returnedToken)
	}

	var updated model.Fingerprint
	if err := db.First(&updated, record.ID).Error; err != nil {
		t.Fatalf("reload fingerprint: %v", err)
	}

	expectedFingerprint := hashFingerprint("203.0.113.8", "new-agent", "test-secret")
	if updated.Fingerprint != expectedFingerprint {
		t.Fatalf("expected fingerprint %q, got %q", expectedFingerprint, updated.Fingerprint)
	}
	if updated.UserAgent != "new-agent" {
		t.Fatalf("expected user agent %q, got %q", "new-agent", updated.UserAgent)
	}
	if updated.IP != "203.0.113.8" {
		t.Fatalf("expected ip %q, got %q", "203.0.113.8", updated.IP)
	}

	var count int64
	if err := db.Model(&model.Fingerprint{}).Count(&count).Error; err != nil {
		t.Fatalf("count fingerprints: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 fingerprint record, got %d", count)
	}
}

func openFingerprintTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Fingerprint{}); err != nil {
		t.Fatalf("migrate fingerprint table: %v", err)
	}

	return db
}
