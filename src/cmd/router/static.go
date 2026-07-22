package cmd

import (
	"blog_api/src/model"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var statFile = os.Stat

func staticFileHandler(cfg *model.Config) gin.HandlerFunc {
	baseDir := resolveStaticBaseDir(cfg)
	panelDir := resolvePanelDir()
	absBaseDir, _ := filepath.Abs(baseDir)
	absPanelDir, _ := filepath.Abs(panelDir)

	return func(c *gin.Context) {
		reqPath, ok := normalizeRequestPath(c.Request.URL.Path)
		if !ok {
			c.String(http.StatusBadRequest, "Bad Request")
			return
		}

		if hasHiddenPathSegment(reqPath) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}

		for _, excludedPath := range cfg.Safe.ExcludePaths {
			normalizedExclude, valid := normalizeRequestPath(excludedPath)
			if !valid || normalizedExclude == "/" {
				continue
			}

			if reqPath == normalizedExclude || strings.HasPrefix(reqPath, normalizedExclude+"/") {
				c.String(http.StatusForbidden, "Forbidden")
				return
			}
		}

		if isProtectedDatabasePath(reqPath, cfg.Data.Database.Path, absBaseDir) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}

		requestBaseDir := baseDir
		requestAbsBaseDir := absBaseDir
		requestPath := strings.TrimPrefix(reqPath, "/")
		isPanelPath := reqPath == "/panel" || strings.HasPrefix(reqPath, "/panel/")
		if isPanelPath {
			requestBaseDir = panelDir
			requestAbsBaseDir = absPanelDir
			requestPath = strings.TrimPrefix(strings.TrimPrefix(reqPath, "/panel"), "/")
		}

		fsPath := filepath.Join(requestBaseDir, requestPath)
		if !isWithinBaseDir(fsPath, requestAbsBaseDir) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}

		info, err := statFile(fsPath)

		if os.IsNotExist(err) {
			if isPanelPath {
				spaIndex := filepath.Join(panelDir, "index.html")
				if _, err := os.Stat(spaIndex); err == nil {
					c.File(spaIndex)
					return
				}
			}
			c.String(http.StatusNotFound, "Not Found")
			return
		}

		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}

		if info.IsDir() {
			indexPath := filepath.Join(fsPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				c.File(indexPath)
				return
			}
			c.String(http.StatusForbidden, "Directory listing is not allowed")
			return
		}
		c.File(fsPath)
	}
}

func resolveStaticBaseDir(cfg *model.Config) string {
	if cfg == nil {
		return "data"
	}

	baseDir := strings.TrimSpace(cfg.Data.Resource.Path)
	if baseDir == "" {
		baseDir = "data"
	}

	return filepath.Clean(baseDir)
}

func resolvePanelDir() string {
	panelDir := strings.TrimSpace(os.Getenv("PANEL_ROOT"))
	if panelDir == "" {
		panelDir = filepath.Join("data", "panel")
	}

	return filepath.Clean(panelDir)
}

func normalizeRequestPath(raw string) (string, bool) {
	if strings.Contains(raw, "\x00") {
		return "", false
	}

	normalized := strings.ReplaceAll(raw, "\\", "/")
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "." || segment == ".." {
			return "", false
		}
	}

	cleaned := path.Clean("/" + normalized)
	if !strings.HasPrefix(cleaned, "/") {
		return "", false
	}
	return cleaned, true
}

func hasHiddenPathSegment(reqPath string) bool {
	parts := strings.Split(strings.Trim(reqPath, "/"), "/")
	for _, part := range parts {
		if part != "" && strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func isProtectedDatabasePath(reqPath, databasePath, absBaseDir string) bool {
	if databasePath == "" {
		return false
	}

	absDatabasePath, err := filepath.Abs(databasePath)
	if err != nil {
		return false
	}

	relativeDatabasePath, err := filepath.Rel(absBaseDir, absDatabasePath)
	if err != nil || relativeDatabasePath == "." || filepath.IsAbs(relativeDatabasePath) || relativeDatabasePath == ".." || strings.HasPrefix(relativeDatabasePath, ".."+string(filepath.Separator)) {
		return false
	}

	databaseRequestPath := "/" + filepath.ToSlash(relativeDatabasePath)
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		if reqPath == databaseRequestPath+suffix {
			return true
		}
	}
	return false
}

func isWithinBaseDir(targetPath, absBaseDir string) bool {
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	resolvedBaseDir, err := filepath.EvalSymlinks(absBaseDir)
	if err != nil {
		resolvedBaseDir = absBaseDir
	}

	baseWithSep := resolvedBaseDir + string(filepath.Separator)
	if absTargetPath != absBaseDir && !strings.HasPrefix(absTargetPath, absBaseDir+string(filepath.Separator)) {
		return false
	}

	resolvedPath, err := filepath.EvalSymlinks(absTargetPath)
	if err != nil {
		return true
	}

	return resolvedPath == resolvedBaseDir || strings.HasPrefix(resolvedPath, baseWithSep)
}
