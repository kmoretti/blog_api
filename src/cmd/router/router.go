package cmd

import (
	"blog_api/src/model"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotImplemented 是一个处理尚未实现的功能的处理程序。
func NotImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Not Implemented",
	})
}

// SetupRouter 初始化并配置 Gin 路由器
func SetupRouter(db *gorm.DB, cfg *model.Config, startTime time.Time) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Safe.CorsAllowHostlist,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Antibot-Token", "CF-Turnstile-Token", "X-Turnstile-Token", "X-fingerprint-token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerRoutes(router, db, cfg, startTime)
	if os.Getenv("PPROF_ENABLED") == "true" {
		pprof.Register(router)
	}
	router.NoRoute(staticFileHandler(cfg))
	return router
}
