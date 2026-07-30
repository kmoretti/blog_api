package handlerAction

import (
	"blog_api/src/model"
	"blog_api/src/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TestEmailReq is the payload for sending a test email.
type TestEmailReq struct {
	To        string          `json:"to" binding:"required,email"`
	EmailConf model.EmailConf `json:"email_conf" binding:"required"`
}

// TestEmail sends a test message using the provided SMTP configuration.
func (h *ConfigHandler) TestEmail(c *gin.Context) {
	var req TestEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的请求: "+err.Error()))
		return
	}

	conf := req.EmailConf
	if strings.TrimSpace(conf.Sender) == "" {
		conf.Sender = conf.UserName
	}

	content := service.EmailContent{
		Subject: "SMTP 测试邮件",
		Body:    "<p>这是一封来自 Blog API 的 SMTP 测试邮件。</p><p>如果你收到此邮件，说明邮件配置正常。</p>",
		IsHTML:  true,
	}

	if err := service.SendEmail(conf, []string{req.To}, content); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "邮件发送失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{
		"message": "测试邮件已发送，请查收收件箱（包括垃圾箱）",
	}))
}
