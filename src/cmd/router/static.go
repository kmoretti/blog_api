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

func staticFileHandler(cfg *model.Config) gin.HandlerFunc {
	baseDir := resolveStaticBaseDir(cfg)
	absBaseDir, _ := filepath.Abs(baseDir)

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

		if cfg.Data.Database.Path != "" {
			dbFileName := filepath.Base(cfg.Data.Database.Path)
			if reqPath == "/"+dbFileName || reqPath == "/"+dbFileName+"/" {
				c.String(http.StatusForbidden, "Forbidden")
				return
			}
		}

		fsPath := filepath.Join(baseDir, strings.TrimPrefix(reqPath, "/"))
		if !isWithinBaseDir(fsPath, absBaseDir) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}

		info, err := os.Stat(fsPath)

		if os.IsNotExist(err) {
			if strings.HasPrefix(reqPath, "/panel/") {
				spaIndex := filepath.Join(baseDir, "panel", "index.html")
				if _, err := os.Stat(spaIndex); err == nil {
					c.File(spaIndex)
					return
				}
			}
			c.String(http.StatusNotFound, "Not Found")
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

func normalizeRequestPath(raw string) (string, bool) {
	if strings.Contains(raw, "\x00") {
		return "", false
	}
	cleaned := path.Clean("/" + strings.ReplaceAll(raw, "\\", "/"))
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

func isWithinBaseDir(targetPath, absBaseDir string) bool {
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	baseWithSep := absBaseDir + string(filepath.Separator)
	if absTargetPath != absBaseDir && !strings.HasPrefix(absTargetPath, baseWithSep) {
		return false
	}

	resolvedPath, err := filepath.EvalSymlinks(absTargetPath)
	if err != nil {
		return true
	}

	return resolvedPath == absBaseDir || strings.HasPrefix(resolvedPath, baseWithSep)
}
