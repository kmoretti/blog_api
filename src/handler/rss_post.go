package handler

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RssPostHandler handles RSS post related requests
type RssPostHandler struct {
	DB *gorm.DB
}

// NewRssPostHandler creates a new RSS post handler
func NewRssPostHandler(db *gorm.DB) *RssPostHandler {
	return &RssPostHandler{DB: db}
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
