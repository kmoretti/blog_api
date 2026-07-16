package authHandler

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// VerifyPublicHandler exposes safe verification config for public clients.
type VerifyPublicHandler struct{}

// GetVerifyConfig handles GET /api/public/verify-conf request.
func (h *VerifyPublicHandler) GetVerifyConfig(c *gin.Context) {
	cfg := config.GetConfig()
	c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
		"turnstile": map[string]interface{}{
			"enable":   cfg.Verify.Turnstile.Enable,
			"site_key": cfg.Verify.Turnstile.SiteKey,
		},
	}))
}
