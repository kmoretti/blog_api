package handlerAction

import (
	"blog_api/src/model"
	momentRepositories "blog_api/src/repositories/moment"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MediaHandler handles media related actions
type MediaHandler struct {
	DB *gorm.DB
}

// CreateMedia handles POST /api/action/moments/media request
func (h *MediaHandler) CreateMedia(c *gin.Context) {
	var req model.CreateMomentMediaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if req.MomentID <= 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid moment_id"))
		return
	}
	if req.MediaURL == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "media_url is required"))
		return
	}

	if req.MediaType == "" {
		req.MediaType = "image"
	}
	validMediaTypes := map[string]bool{
		"image": true,
		"video": true,
	}
	if !validMediaTypes[req.MediaType] {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid media_type"))
		return
	}
	if req.IsLocal != 0 && req.IsLocal != 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid is_local"))
		return
	}

	media := model.MomentMedia{
		MomentID:  req.MomentID,
		Name:      req.Name,
		MediaURL:  req.MediaURL,
		MediaType: req.MediaType,
		IsLocal:   req.IsLocal,
		IsDeleted: 0,
	}

	if err := momentRepositories.CreateMomentMedia(h.DB, &media); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to create media"))
		return
	}

	c.JSON(http.StatusCreated, model.NewSuccessResponse(media))
}

// DeleteMedia handles DELETE /api/action/moments/media/:id request
func (h *MediaHandler) DeleteMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid media id"))
		return
	}

	if err := momentRepositories.DeleteMomentMedia(h.DB, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "media not found"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete media"))
		}
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// UpdateMedia handles PUT /api/action/moments/media/:id request
func (h *MediaHandler) UpdateMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid media id"))
		return
	}

	var req model.UpdateMomentMediaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if req.MomentID != nil && *req.MomentID <= 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid moment_id"))
		return
	}
	if req.MediaType != nil {
		validMediaTypes := map[string]bool{
			"image": true,
			"video": true,
		}
		if !validMediaTypes[*req.MediaType] {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid media_type"))
			return
		}
	}
	if req.IsLocal != nil && *req.IsLocal != 0 && *req.IsLocal != 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid is_local"))
		return
	}

	updates := map[string]interface{}{}
	media := model.MomentMedia{ID: id}
	if req.MomentID != nil {
		updates["moment_id"] = *req.MomentID
		media.MomentID = *req.MomentID
	}
	if req.Name != nil {
		updates["name"] = *req.Name
		media.Name = *req.Name
	}
	if req.MediaURL != nil {
		updates["media_url"] = *req.MediaURL
		media.MediaURL = *req.MediaURL
	}
	if req.MediaType != nil {
		updates["media_type"] = *req.MediaType
		media.MediaType = *req.MediaType
	}
	if req.IsLocal != nil {
		updates["is_local"] = *req.IsLocal
		media.IsLocal = *req.IsLocal
	}

	if err := momentRepositories.UpdateMomentMedia(h.DB, id, updates); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "media not found"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update media"))
		}
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(media))
}

// GetMedia handles GET /api/action/moments/media request
func (h *MediaHandler) GetMedia(c *gin.Context) {
	var req struct {
		Page      int    `form:"page"`
		PageSize  int    `form:"page_size"`
		MomentID  int    `form:"moment_id"`
		MediaType string `form:"type"`
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

	media, total, err := momentRepositories.QueryMedia(h.DB, req.Page, req.PageSize, req.MomentID, req.MediaType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to get media"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(&model.QueryMediaResponse{
		Media: media,
		Total: total,
	}))
}
