package handlerAction

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 处理与配置相关的请求
type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// UpdateConfig 处理 PUT /api/action/config 请求，用于更新系统配置
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var req []model.UpdateConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的请求体,请传入一个数组: "+err.Error()))
		return
	}
	if err := config.UpdateAndSaveConfigs(req); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "更新配置失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{
		"message": "配置更新成功，重启服务后生效",
	}))
}
