package handler

import (
	"blog_api/src/model"
	"blog_api/src/repositories"
	"blog_api/src/service"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatusHandler handles system status requests.
type StatusHandler struct {
	DB           *gorm.DB
	StartTime    time.Time
	DatabasePath string
	DataPath     string
}

// GetSystemStatus handles the GET /api/status request.
func (h *StatusHandler) GetSystemStatus(c *gin.Context) {
	stats, err := repositories.GetSystemStats(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve system stats"))
		return
	}
	databaseSize, err := service.FileSize(h.DatabasePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve database size"))
		return
	}
	dataFolderSize, err := service.DirectorySize(h.DataPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve data folder size"))
		return
	}
	uptime := time.Since(h.StartTime)
	systemStatus := model.SystemStatus{
		Uptime:              fmt.Sprintf("%v", uptime.Round(time.Second)),
		StatusData:          stats,
		DatabaseSizeBytes:   databaseSize,
		DataFolderSizeBytes: dataFolderSize,
		Time:                time.Now().Unix(),
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(systemStatus))
}
