package handler

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	crawlerService "blog_api/src/service/crawler"
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RssPostHandler handles RSS post related requests
type RssPostHandler struct {
	DB *gorm.DB
}

// DeleteRssPost handles deletion of one persisted RSS article.
func (h *RssPostHandler) DeleteRssPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的 RSS 文章 ID"))
		return
	}

	rowsAffected, err := friendsRepositories.DeleteRssPostByID(h.DB, id)
	if err != nil {
		log.Printf("[rss_post] failed to delete post %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "删除 RSS 文章失败"))
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, "未找到指定 RSS 文章"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
}

// GetRssPosts handles GET /api/rss request
// Query parameters:
//   - rss_id: filter by rss_id (optional)
//   - friend_link_id: filter by friend_link_id (optional)
//   - page: for pagination (optional, default: 1)
//   - page_size: for pagination (optional, default: 10)
func (h *RssPostHandler) GetRssPosts(c *gin.Context) {
	var query model.PostQuery

	// Bind query parameters
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid query parameters"))
		return
	}

	// Set default pagination values
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	posts, total, err := friendsRepositories.GetPosts(h.DB, &query)
	if err != nil {
		log.Printf("[rss_post] failed to retrieve posts: %+v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve posts"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(&model.PaginatedResponse{
		Items:    posts,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}))
}

// RefreshRssPosts triggers background parsing for all valid RSS feeds.
func (h *RssPostHandler) RefreshRssPosts(c *gin.Context) {
	opts := model.FriendRssQueryOptions{Status: "valid"}
	resp, err := friendsRepositories.QueryFriendRss(h.DB, opts)
	if err != nil {
		log.Printf("[rss_post] failed to query valid rss feeds: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "获取 RSS 列表失败"))
		return
	}

	if len(resp.Feeds) == 0 {
		c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"message": "没有需要抓取的 RSS 源"}))
		return
	}

	go func(feeds []model.FriendRss) {
		result := crawlerService.ParseRssFeedsConcurrently(context.Background(), h.DB, feeds)
		if result.DatabaseFailures > 0 {
			log.Printf("[rss_post] background refresh finished with %d database failures", result.DatabaseFailures)
		}
	}(resp.Feeds)

	c.JSON(http.StatusAccepted, model.NewSuccessResponseWithCode(http.StatusAccepted, gin.H{
		"message":    "已开始后台抓取 RSS",
		"feed_count": len(resp.Feeds),
	}))
}
