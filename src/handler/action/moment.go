package handlerAction

import (
	"blog_api/src/model"
	momentRepositories "blog_api/src/repositories/moment"
	"blog_api/src/service"
	botService "blog_api/src/service/bot"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MomentHandler handles moment related actions
type MomentHandler struct {
	DB *gorm.DB
}

// CreateMoment handles POST /api/action/moments request
func (h *MomentHandler) CreateMoment(c *gin.Context) {
	var req model.CreateMomentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}
	if req.GuildID != nil && *req.GuildID < 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid guild_id"))
		return
	}
	if req.ChannelID != nil && *req.ChannelID < 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid channel_id"))
		return
	}
	if req.MessageID != nil && *req.MessageID < 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid message_id"))
		return
	}

	if err := service.CreateMoment(h.DB, req); err != nil {
		log.Printf("[moments] create moment failed: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to create moment"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// GetMoments handles GET /api/action/moments request
func (h *MomentHandler) GetMoments(c *gin.Context) {
	var req struct {
		Page     int    `form:"page"`
		PageSize int    `form:"page_size"`
		Status   string `form:"status"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid query parameters"))
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	resp, err := service.GetMomentsWithMedia(h.DB, req.Page, req.PageSize, req.Status, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to get moments"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(resp))
}

// DeleteMoment handles DELETE /api/action/moments/:id request
func (h *MomentHandler) DeleteMoment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid moment id"))
		return
	}

	if err := botService.DeleteMomentWithSync(h.DB, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "moment not found"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete moment"))
		}
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// UpdateMoment handles PUT /api/action/moments/:id request
func (h *MomentHandler) UpdateMoment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid moment id"))
		return
	}

	var req model.UpdateMomentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if req.Status != nil {
		validStatuses := map[string]bool{
			"visible": true,
			"hidden":  true,
			"deleted": true,
		}
		if !validStatuses[*req.Status] {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid status"))
			return
		}
	}
	if req.GuildID != nil && *req.GuildID < 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid guild_id"))
		return
	}
	if req.ChannelID != nil && *req.ChannelID < 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid channel_id"))
		return
	}
	if req.MessageID != nil && *req.MessageID < 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid message_id"))
		return
	}

	updates := map[string]interface{}{}
	moment := model.Moment{ID: id}
	if req.Content != nil {
		updates["content"] = *req.Content
		moment.Content = *req.Content
	}
	if req.Status != nil {
		updates["status"] = *req.Status
		moment.Status = *req.Status
	}
	if req.GuildID != nil {
		updates["guild_id"] = *req.GuildID
		moment.GuildID = *req.GuildID
	}
	if req.ChannelID != nil {
		updates["channel_id"] = *req.ChannelID
		moment.ChannelID = *req.ChannelID
	}
	if req.MessageID != nil {
		updates["message_id"] = *req.MessageID
		moment.MessageID = *req.MessageID
	}
	if req.MessageLink != nil {
		updates["message_link"] = *req.MessageLink
		moment.MessageLink = *req.MessageLink
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "no fields to update"))
		return
	}

	if err := momentRepositories.UpdateMoment(h.DB, id, updates); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "moment not found"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update moment"))
		}
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(moment))
}

// DeleteMomentReaction handles DELETE /api/action/moments/:id/reactions request
func (h *MomentHandler) DeleteMomentReaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid moment id"))
		return
	}

	reaction := c.Query("reaction")
	if reaction == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "reaction query parameter is required"))
		return
	}

	if err := momentRepositories.ClearReactionsByType(h.DB, id, reaction); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete reactions"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}
