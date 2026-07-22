package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGitHubRepositoryHandlerRejectsInvalidPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/public/github/repository/:owner/:repo", GitHubRepository)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/public/github/repository/owner!/repo", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestGitHubRepositoryHandlerNormalizesGitSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/public/github/repository/:owner/:repo", GitHubRepository)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/public/github/repository/owner/repo.git", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusBadRequest {
		t.Fatalf("expected normalized repository suffix to pass validation, got %d", recorder.Code)
	}
}
