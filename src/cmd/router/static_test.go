package cmd

import (
	"blog_api/src/model"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaticFileHandlerPanelDefaultRoot(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	panelDir := filepath.Join(root, "data", "panel")
	writeTestFile(t, filepath.Join(resourceDir, "asset.txt"), "resource")
	writeTestFile(t, filepath.Join(panelDir, "index.html"), "default panel")

	t.Setenv("PANEL_ROOT", "")
	chdirTestDir(t, root)

	recorder := performStaticRequest(t, &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}, "/panel/")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "default panel" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestStaticFileHandlerPanelCustomRoot(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	panelDir := filepath.Join(root, "custom-panel")
	writeTestFile(t, filepath.Join(resourceDir, "asset.txt"), "resource")
	writeTestFile(t, filepath.Join(panelDir, "index.html"), "custom panel")

	t.Setenv("PANEL_ROOT", panelDir)

	recorder := performStaticRequest(t, &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}, "/panel/")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "custom panel" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestStaticFileHandlerResourceRouting(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	panelDir := filepath.Join(root, "panel")
	writeTestFile(t, filepath.Join(resourceDir, "asset.txt"), "resource")
	writeTestFile(t, filepath.Join(panelDir, "index.html"), "panel")
	t.Setenv("PANEL_ROOT", panelDir)

	recorder := performStaticRequest(t, &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}, "/asset.txt")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "resource" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestStaticFileHandlerRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	writeTestFile(t, filepath.Join(root, "secret.txt"), "secret")
	writeTestFile(t, filepath.Join(resourceDir, "asset.txt"), "resource")
	config := &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}

	for _, requestPath := range []string{"/../secret.txt", "/./asset.txt", "/panel/../secret.txt"} {
		recorder := performStaticRequest(t, config, requestPath)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("path=%q status=%d body=%q", requestPath, recorder.Code, recorder.Body.String())
		}
	}
}

func TestStaticFileHandlerReturnsErrorForStatFailure(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	config := &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}
	originalStatFile := statFile
	statFile = func(string) (os.FileInfo, error) {
		return nil, errors.New("stat failed")
	}
	t.Cleanup(func() { statFile = originalStatFile })

	recorder := performStaticRequest(t, config, "/asset.txt")
	if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != "Internal Server Error" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestStaticFileHandlerRejectsHiddenPaths(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	writeTestFile(t, filepath.Join(resourceDir, ".env"), "secret")
	writeTestFile(t, filepath.Join(resourceDir, "visible.txt"), "visible")
	config := &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}

	for _, requestPath := range []string{"/.env", "/.hidden/visible.txt"} {
		recorder := performStaticRequest(t, config, requestPath)
		if recorder.Code != http.StatusForbidden {
			t.Errorf("path=%q status=%d body=%q", requestPath, recorder.Code, recorder.Body.String())
		}
	}
}

func TestStaticFileHandlerPanelSPAFallback(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	panelDir := filepath.Join(root, "panel")
	writeTestFile(t, filepath.Join(panelDir, "index.html"), "spa")
	t.Setenv("PANEL_ROOT", panelDir)

	recorder := performStaticRequest(t, &model.Config{Data: model.DataConfig{Resource: model.ResourceConfig{Path: resourceDir}}}, "/panel/dashboard/settings")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "spa" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestStaticFileHandlerProtectsNestedDatabaseAndSidecars(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources")
	databasePath := filepath.Join(resourceDir, "db", "app.sqlite")
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		writeTestFile(t, databasePath+suffix, "database")
	}
	config := &model.Config{Data: model.DataConfig{
		Database: model.DatabaseConfig{Path: databasePath},
		Resource: model.ResourceConfig{Path: resourceDir},
	}}

	for _, requestPath := range []string{"/db/app.sqlite", "/db/app.sqlite-wal", "/db/app.sqlite-shm", "/db/app.sqlite-journal"} {
		recorder := performStaticRequest(t, config, requestPath)
		if recorder.Code != http.StatusForbidden {
			t.Errorf("path=%q status=%d body=%q", requestPath, recorder.Code, recorder.Body.String())
		}
	}
}

func performStaticRequest(t *testing.T, config *model.Config, requestPath string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	engine.NoRoute(staticFileHandler(config))
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://example.com"+requestPath, nil))
	return recorder
}

func writeTestFile(t *testing.T, filePath, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func chdirTestDir(t *testing.T, dir string) {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(current) })
}
