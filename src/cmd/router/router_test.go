package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blog_api/src/config"
	"blog_api/src/model"
)

func TestSetupRouterRedirectsRootToPanel(t *testing.T) {
	if _, err := config.Load(); err != nil {
		t.Fatalf("load config: %v", err)
	}
	router := SetupRouter(nil, &model.Config{Safe: model.SafeConfig{CorsAllowHostlist: []string{"http://localhost"}}}, time.Now())
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/panel/" {
		t.Fatalf("expected redirect location /panel/, got %q", location)
	}
}
