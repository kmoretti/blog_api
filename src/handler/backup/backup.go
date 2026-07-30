package backup

import (
	"blog_api/src/model"
	"blog_api/src/service/backup"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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
