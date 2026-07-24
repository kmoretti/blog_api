package handlerAction

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	crawlerService "blog_api/src/service/crawler"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FriendRssHandler 处理与 friend_rss 相关的请求
type FriendRssHandler struct {
	DB *gorm.DB
}

// CreateRss 处理 POST /api/action/rss 请求，用于创建新的 RSS feed。
func (h *FriendRssHandler) CreateRss(c *gin.Context) {
	var req model.CreateRssReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的请求体: "+err.Error()))
		return
	}

	friendLinkID := req.FriendLinkID
	if friendLinkID == 0 {
		friendLinkID = -1
	}

	if friendLinkID != -1 {
		// 检查 friend_link_id 是否真实存在
		exists, err := friendsRepositories.FriendLinkExists(h.DB, friendLinkID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "检查友链是否存在时出错: "+err.Error()))
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, fmt.Sprintf("ID 为 %d 的友链不存在", friendLinkID)))
			return
		}
	}

	name := req.Name
	if name == "" {
		var err error
		name, err = crawlerService.GetRssTitle(req.RssURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "无法获取 RSS 标题: "+err.Error()))
			return
		}
	}

	createdFeed, err := friendsRepositories.CreateFriendRssFeeds(h.DB, friendLinkID, req.RssURL, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "创建 RSS 失败: "+err.Error()))
		return
	}
	if createdFeed == nil {
		c.JSON(http.StatusConflict, model.NewErrorResponse(http.StatusConflict, "RSS feed 已存在"))
		return
	}

	// 创建成功后立即在后台抓取一次文章，避免用户刷新后仍看不到内容
	go crawlerService.ParseRssFeed(h.DB, createdFeed.ID, createdFeed.RssURL)

	c.JSON(http.StatusCreated, model.NewSuccessResponse(gin.H{"id": createdFeed.ID}))
}

// DeleteFriendRss 处理 DELETE /api/action/rss/:id 请求
func (h *FriendRssHandler) DeleteFriendRss(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的 RSS ID"))
		return
	}

	rowsAffected, err := friendsRepositories.DeleteFriendRssByID(h.DB, uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "删除 RSS 失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
}

// EditRss 处理 PUT /api/action/rss/:id 请求，用于更新现有的 RSS feed。
func (h *FriendRssHandler) EditRss(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的 RSS ID"))
		return
	}

	var req model.EditFriendRssReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的请求体: "+err.Error()))
		return
	}

	rowsAffected, err := friendsRepositories.UpdateFriendRssByID(h.DB, uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "更新 RSS 失败: "+err.Error()))
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, "未找到指定 ID 的 RSS 或没有字段需要更新"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
}

// GetRss 处理 GET /api/action/rss 请求
func (h *FriendRssHandler) GetRss(c *gin.Context) {
	// 解析查询参数
	status := c.Query("status")

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

	// 限制最大页面大小
	if pageSize > 100 {
		pageSize = 100
	}

	// 如果提供了 status 参数，则进行验证
	if status != "" {
		validStatuses := map[string]bool{
			"survival": true,
			"timeout":  true,
			"error":    true,
			"pause":    true,
			"valid":    true,
		}
		if !validStatuses[status] {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的状态参数"))
			return
		}
	}

	// 查询友链和总数
	opts := model.FriendRssQueryOptions{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}
	resp, err := friendsRepositories.QueryFriendRss(h.DB, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "获取友链列表失败"))
		return
	}

	// 构建分页响应
	paginatedData := model.PaginatedResponse{
		Items:    resp.Feeds,
		Total:    int(resp.Total),
		Page:     page,
		PageSize: pageSize,
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(paginatedData))
}
