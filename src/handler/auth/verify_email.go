package authHandler

import (
	"fmt"
	"log"
	"net/http"

	"blog_api/src/config"
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"blog_api/src/service"

	"github.com/gin-gonic/gin"
)

// SendEmailCode handles POST /api/verify/email request.
// If code is provided, it confirms the code and returns a token; otherwise it sends a code.
func (h *VerifyHandler) SendEmailCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
		Code  string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if req.Code != "" {
		if !service.ValidateEmailVerifyCode(req.Email, req.Code) {
			c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "invalid email verification code"))
			return
		}

		token, expiresAt, err := service.IssueEmailToken(req.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to issue email token"))
			return
		}

		friendLinks := make([]model.FriendLinkDTO, 0)
		if h.DB != nil {
			result, err := friendsRepositories.QueryFriendLinks(h.DB, model.FriendLinkQueryOptions{
				Email: req.Email,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link"))
				return
			}
			for _, link := range result.Links {
				friendLinks = append(friendLinks, toPublicFriendLinkDTO(link))
			}
		}

		c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
			"token":        token,
			"expires_at":   expiresAt,
			"expires_in":   service.EmailTokenTTLSeconds(),
			"friend_links": friendLinks,
		}))
		return
	}

	cfg := config.GetConfig()
	code, expiresAt, err := service.IssueEmailVerifyCode(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to issue email code"))
		return
	}

	content := service.EmailContent{
		Subject: "blog api 登录验证",
		Body:    fmt.Sprintf("你的验证码是： %s 。 它将在 %d 后过期。", code, service.EmailCodeTTLSeconds()/60),
		IsHTML:  false,
	}
	if !cfg.Email.Enable {
		log.Printf("[email][disabled] To=%s Subject=%s Body=%s", req.Email, content.Subject, content.Body)
		if cfg.IsDev {
			c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
				"expires_at": expiresAt,
				"expires_in": service.EmailCodeTTLSeconds(),
			}))
		} else {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResponse(503, "email service is disabled"))
		}
		return
	}
	if err := service.SendEmail(cfg.Email, []string{req.Email}, content); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to send verification email"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
		"expires_at": expiresAt,
		"expires_in": service.EmailCodeTTLSeconds(),
	}))
}

func toPublicFriendLinkDTO(link model.FriendWebsite) model.FriendLinkDTO {
	return model.FriendLinkDTO{
		ID:          link.ID,
		Name:        link.Name,
		Link:        link.Link,
		Avatar:      link.Avatar,
		Description: link.Info,
		Status:      link.Status,
		EnableRss:   link.EnableRss,
		UpdatedAt:   link.UpdatedAt,
	}
}
