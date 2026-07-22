package handler

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FriendLinkHandler handles friend link related requests
type FriendLinkHandler struct {
	DB *gorm.DB
}

// toFriendLinkDTOs converts a slice of FriendWebsite models to a slice of FriendLinkDTOs.
// If isPrivate is true, it includes sensitive fields like Email and Times.
func toFriendLinkDTOs(links []model.FriendWebsite, isPrivate bool) []model.FriendLinkDTO {
	dtoLinks := make([]model.FriendLinkDTO, 0, len(links))
	for _, link := range links {
		dto := model.FriendLinkDTO{
			ID:             link.ID,
			Name:           link.Name,
			Link:           link.Link,
			Avatar:         link.Avatar,
			Description:    link.Info,
			Status:         link.Status,
			EnableRss:      link.EnableRss,
			UpdatedAt:      link.UpdatedAt,
			Snapshot:       link.Snapshot,
			FriendLinkPage: link.FriendLinkPage,
			Feed:           link.Feed,
		}
		if isPrivate {
			dto.Email = link.Email
			dto.Times = link.Times
			dto.IsDied = link.IsDied
		}
		dtoLinks = append(dtoLinks, dto)
	}
	return dtoLinks
}

func toFriendLinkDTO(link model.FriendWebsite, isPrivate bool) model.FriendLinkDTO {
	dto := model.FriendLinkDTO{
		ID:             link.ID,
		Name:           link.Name,
		Link:           link.Link,
		Avatar:         link.Avatar,
		Description:    link.Info,
		Status:         link.Status,
		EnableRss:      link.EnableRss,
		UpdatedAt:      link.UpdatedAt,
		Snapshot:       link.Snapshot,
		FriendLinkPage: link.FriendLinkPage,
		Feed:           link.Feed,
	}
	if isPrivate {
		dto.Email = link.Email
		dto.Times = link.Times
		dto.IsDied = link.IsDied
	}
	return dto
}

// getFriendLinks is a helper function to get friend links with common logic.
func (h *FriendLinkHandler) getFriendLinks(c *gin.Context, isPrivate bool) {
	// Parse query parameters
	status := c.Query("status")
	search := c.Query("search")
	isDiedStr := c.Query("is_died")
	var isDied *bool
	if isDiedStr != "" {
		val, err := strconv.ParseBool(isDiedStr)
		if err == nil {
			isDied = &val
		}
	}

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid page parameter"))
		return
	}

	pageSizeStr := c.DefaultQuery("page_size", "20")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid page_size parameter"))
		return
	}

	// Limit maximum page size
	if pageSize > 1000 {
		pageSize = 1000
	}

	// Validate status parameter if provided
	if status != "" {
		validStatuses := map[string]bool{
			"survival": true,
			"timeout":  true,
			"error":    true,
			"pending":  true,
		}
		if !validStatuses[status] {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid status parameter"))
			return
		}
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Query friend links and total count
	opts := model.FriendLinkQueryOptions{
		Status: status,
		Search: search,
		Offset: offset,
		Limit:  pageSize,
		IsDied: isDied,
	}
	resp, err := friendsRepositories.QueryFriendLinks(h.DB, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend links"))
		return
	}

	// Convert to DTO based on the context (public or private)
	dtoLinks := toFriendLinkDTOs(resp.Links, isPrivate)

	// Build paginated response
	paginatedData := model.PaginatedResponse{
		Items:    dtoLinks,
		Total:    int(resp.Count),
		Page:     page,
		PageSize: pageSize,
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(paginatedData))
}

func (h *FriendLinkHandler) getFriendLinkByID(c *gin.Context, isPrivate bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid friend link ID"))
		return
	}

	link, err := friendsRepositories.GetFriendLinkByID(h.DB, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "friend link not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link"))
		return
	}

	dto := toFriendLinkDTO(link, isPrivate)
	c.JSON(http.StatusOK, model.NewSuccessResponse(dto))
}

// GetAllFriendLinks handles GET /api/friend/ request
func (h *FriendLinkHandler) GetAllFriendLinks(c *gin.Context) {
	h.getFriendLinks(c, false)
}

// GetFriendLinkByID handles GET /api/public/friend/:id request
func (h *FriendLinkHandler) GetFriendLinkByID(c *gin.Context) {
	h.getFriendLinkByID(c, false)
}

// GetFriendLinkByEmailToken handles GET /api/public/friend/self request (email token).
func (h *FriendLinkHandler) GetFriendLinkByEmailToken(c *gin.Context) {
	authType, ok := c.Get("auth_type")
	if !ok || authType != "email" {
		c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email token is required"))
		return
	}

	authEmail, _ := c.Get("auth_email")
	email, _ := authEmail.(string)
	if email == "" {
		c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email token is invalid"))
		return
	}

	link, err := friendsRepositories.GetFriendLinkByEmail(h.DB, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "friend link not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link"))
		return
	}

	dto := toFriendLinkDTO(link, false)
	c.JSON(http.StatusOK, model.NewSuccessResponse(dto))
}

// GetFullFriendLinks handles GET /api/action/friend/ request (authenticated)
// It returns the full friend link data, including sensitive fields.
func (h *FriendLinkHandler) GetFullFriendLinks(c *gin.Context) {
	h.getFriendLinks(c, true)
}

// GetFullFriendLinkByID handles GET /api/action/friend/:id request (authenticated)
func (h *FriendLinkHandler) GetFullFriendLinkByID(c *gin.Context) {
	h.getFriendLinkByID(c, true)
}
