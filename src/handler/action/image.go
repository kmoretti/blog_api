package handlerAction

import (
	"blog_api/src/model"
	imageRepositories "blog_api/src/repositories/image"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ImageHandler handles image related requests
type ImageHandler struct {
	DB *gorm.DB
}

// GetImages handles GET /api/action/image request
func (h *ImageHandler) GetImages(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的页面参数"))
		return
	}

	pageSizeStr := c.DefaultQuery("page_size", "20")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的页面大小参数"))
		return
	}

	if pageSize > 100 {
		pageSize = 100
	}

	opts := model.ImageQueryOptions{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Name:     c.Query("search"),
	}

	resp, err := imageRepositories.QueryImages(h.DB, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "获取图片列表失败"))
		return
	}

	paginatedData := model.PaginatedResponse{
		Items:    resp.Images,
		Total:    int(resp.Total),
		Page:     page,
		PageSize: pageSize,
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse(paginatedData))
}

// CreateImage handles POST /api/action/image request
func (h *ImageHandler) CreateImage(c *gin.Context) {
	var image model.Image
	if err := c.ShouldBindJSON(&image); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的请求体: "+err.Error()))
		return
	}

	if image.URL == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "图片 URL 不能为空"))
		return
	}
	// 忽略前端参数，默认状态为正常
	image.Status = "normal"

	if err := imageRepositories.CreateImage(h.DB, &image); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "创建图片配置失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusCreated, model.NewSuccessResponse(image))
}

// UpdateImage handles PUT /api/action/image/:id request
func (h *ImageHandler) UpdateImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的图片 ID"))
		return
	}

	var req model.UpdateImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的请求体: "+err.Error()))
		return
	}
	if req.Status != nil {
		validStatuses := map[string]bool{
			"normal":  true,
			"pause":   true,
			"broken":  true,
			"pending": true,
		}
		if !validStatuses[*req.Status] {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的状态参数"))
			return
		}
	}

	image := model.Image{ID: id}
	if req.Name != nil {
		image.Name = *req.Name
	}
	if req.URL != nil {
		image.URL = *req.URL
	}
	if req.LocalPath != nil {
		image.LocalPath = *req.LocalPath
	}
	if req.IsLocal != nil {
		image.IsLocal = *req.IsLocal
	}
	if req.IsOss != nil {
		image.IsOss = *req.IsOss
	}
	if req.Status != nil {
		image.Status = *req.Status
	}

	if err := imageRepositories.UpdateImage(h.DB, &image); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "图片配置不存在"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "更新图片配置失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(image))
}

// DeleteImage handles DELETE /api/action/image/:id request
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的图片 ID"))
		return
	}

	if err := imageRepositories.DeleteImage(h.DB, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "图片配置不存在"))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "删除图片配置失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}
