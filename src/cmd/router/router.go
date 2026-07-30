package cmd

import (
	"blog_api/src/config"
	"blog_api/src/handler"
	"blog_api/src/model"
	"blog_api/src/service/fcircle"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxRequestBodyBytes = int64(65 << 20)

var corsMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
var corsHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Antibot-Token", "CF-Turnstile-Token", "X-Turnstile-Token", "X-fingerprint-token"}

// SetupRouter 初始化并配置 Gin 路由器
func SetupRouter(db *gorm.DB, cfg *model.Config, startTime time.Time) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20
	router.Use(func(c *gin.Context) {
		if c.Request.Body != nil {
			path := strings.TrimSuffix(c.Request.URL.Path, "/")
			isBackupImport := path == "/api/action/backup/import"
			isModuleImport := strings.HasPrefix(c.Request.URL.Path, "/api/action/import/")
			if !isBackupImport && !isModuleImport {
				c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
			}
		}
		c.Next()
	})
	router.Use(dynamicCORSMiddleware())

	registerRoutes(router, db, cfg, startTime)
	if os.Getenv("PPROF_ENABLED") == "true" {
		pprof.Register(router)
	}

	// 初始化并暴露 fcircle.json（友链圈子聚合格式）
	dataDir := resolveStaticBaseDir(config.GetConfig())
	if err := fcircle.Init(db, dataDir); err != nil {
		log.Printf("[router][fcircle] 初始化失败: %v", err)
	}
	friendLinkGroupHandler := &handler.FriendLinkGroupHandler{DB: db}
	router.GET("/fcircle.json", fcircle.Handler())
	router.GET("/friend.json", friendLinkGroupHandler.GetFriendLinkJSON)

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/panel/")
	})
	router.NoRoute(staticFileHandler())
	return router
}

func dynamicCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if allowedOrigin, ok := allowedCORSOrigin(origin, config.GetConfig().Safe.CorsAllowHostlist); ok {
			header := c.Writer.Header()
			header.Set("Access-Control-Allow-Origin", allowedOrigin)
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Set("Access-Control-Allow-Methods", strings.Join(corsMethods, ", "))
			header.Set("Access-Control-Allow-Headers", strings.Join(corsHeaders, ", "))
			header.Set("Access-Control-Expose-Headers", "Content-Length")
			header.Set("Access-Control-Max-Age", "43200")
			header.Add("Vary", "Origin")
		} else if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func allowedCORSOrigin(origin string, allowlist []string) (string, bool) {
	for _, allowed := range allowlist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" {
			return origin, true
		}
		if allowed == origin {
			return origin, true
		}
	}
	return "", false
}
