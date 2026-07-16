package authHandler

import (
	"net/http"

	"blog_api/src/model"
	"blog_api/src/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VerifyHandler handles verification related requests.
type VerifyHandler struct {
	DB *gorm.DB
}

// IssueVerifyToken handles POST /api/verify/turnstile request.
func (h *VerifyHandler) IssueVerifyToken(c *gin.Context) {
	token, expiresAt, err := service.IssueAntiBotToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to issue verification token"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
		"antibot_token": token,
		"expires_at":    expiresAt,
		"expires_in":    service.AntiBotTTLSeconds(),
	}))
}
