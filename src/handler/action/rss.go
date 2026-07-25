package handlerAction

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	crawlerService "blog_api/src/service/crawler"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FriendRssHandler 处理与 friend_rss 相关的请求
type FriendRssHandler struct {
	DB *gorm.DB
}

func normalizeFriendLinkID(value interface{}) (int, error) {
	var id int
	switch value := value.(type) {
	case int:
		id = value
	case float64:
		const maxSafeInteger = float64(1<<53 - 1)
		if math.Trunc(value) != value || value > maxSafeInteger || value < -maxSafeInteger {
			return 0, fmt.Errorf("friend_link_id must be an integer")
		}
		id = int(value)
	case json.Number:
		parsed, err := strconv.Atoi(value.String())
		if err != nil {
			return 0, fmt.Errorf("friend_link_id must be an integer")
		}
		id = parsed
	default:
		return 0, fmt.Errorf("friend_link_id must be an integer")
	}
	if id == 0 {
		return -1, nil
	}
	if id < -1 {
		return 0, fmt.Errorf("friend_link_id must be -1 or a positive integer")
	}
	return id, nil
}

func validateFriendLinkID(db *gorm.DB, friendLinkID int) error {
	if friendLinkID == -1 {
		return nil
	}
	exists, err := friendsRepositories.FriendLinkExists(db, friendLinkID)
	if err != nil {
		return fmt.Errorf("check friend link: %w", err)
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CreateRss 处理 POST /api/action/rss 请求，用于创建新的 RSS feed。
func (h *FriendRssHandler) CreateRss(c *gin.Context) {
	var req model.CreateRssReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的请求体: "+err.Error()))
		return
	}

	friendLinkID, err := normalizeFriendLinkID(req.FriendLinkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	if err := validateFriendLinkID(h.DB, friendLinkID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, fmt.Sprintf("ID 为 %d 的友链不存在", friendLinkID)))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "检查友链是否存在时出错"))
		return
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
	go crawlerService.ParseRssFeed(c.Request.Context(), h.DB, createdFeed.ID, createdFeed.RssURL)

	c.JSON(http.StatusCreated, model.NewSuccessResponseWithCode(http.StatusCreated, gin.H{"id": createdFeed.ID}))
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

	var existing model.FriendRss
	if err := h.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "查询 RSS 失败: "+err.Error()))
		return
	}

	// 校验归属友链
	if value, ok := req.Data["friend_link_id"]; ok {
		friendLinkID, err := normalizeFriendLinkID(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, err.Error()))
			return
		}
		if err := validateFriendLinkID(h.DB, friendLinkID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, fmt.Sprintf("ID 为 %d 的友链不存在", friendLinkID)))
				return
			}
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "检查友链是否存在时出错"))
			return
		}
		req.Data["friend_link_id"] = friendLinkID
	}

	// 判断更新后是否需要立即重新抓取文章
	urlChanged := false
	if newURL, ok := req.Data["rss_url"].(string); ok && newURL != existing.RssURL {
		urlChanged = true
	}
	revived := false
	if newIsDied, ok := req.Data["is_died"].(bool); ok && existing.IsDied && !newIsDied {
		revived = true
	}
	unpaused := false
	if newStatus, ok := req.Data["status"].(string); ok && existing.Status == "pause" && newStatus != "pause" {
		unpaused = true
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

	if urlChanged || revived || unpaused {
		parseURL := existing.RssURL
		if urlChanged {
			parseURL = req.Data["rss_url"].(string)
		}
		go crawlerService.ParseRssFeed(c.Request.Context(), h.DB, int(id), parseURL)
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
}

// FetchRss fetches and persists articles from one RSS feed immediately.
func (h *FriendRssHandler) FetchRss(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的 RSS ID"))
		return
	}

	feed, err := friendsRepositories.GetFriendRssByID(h.DB, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, "未找到指定 RSS"))
			return
		}
		log.Printf("[handler][rss] failed to retrieve RSS %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "获取 RSS 信息失败"))
		return
	}

	result, err := crawlerService.ParseRssFeed(c.Request.Context(), h.DB, feed.ID, feed.RssURL)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		if errors.Is(err, crawlerService.ErrRssSource) {
			c.JSON(http.StatusBadGateway, model.NewErrorResponse(http.StatusBadGateway, "获取或解析 RSS 失败"))
			return
		}
		log.Printf("[handler][rss] failed to persist RSS %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "保存 RSS 获取结果失败"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(result))
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
