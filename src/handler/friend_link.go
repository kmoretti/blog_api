package handler

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"blog_api/src/service"
	crawlerService "blog_api/src/service/crawler"
	"blog_api/src/service/fcircle"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

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
			skipHealthCheck := link.SkipHealthCheck
			dto.SkipHealthCheck = &skipHealthCheck
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
		skipHealthCheck := link.SkipHealthCheck
		dto.SkipHealthCheck = &skipHealthCheck
	}
	return dto
}

// getFriendLinks is a helper function to get friend links with common logic.
func (h *FriendLinkHandler) getFriendLinks(c *gin.Context, isPrivate bool, email string) {
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
	validStatuses := map[string]bool{
		"survival": true,
		"timeout":  true,
		"error":    true,
		"pending":  true,
		"rejected": true,
	}
	if status != "" && !validStatuses[status] {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid status parameter"))
		return
	}

	// Public friend list should only show approved links by default.
	if !isPrivate && status == "" {
		status = "survival"
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Query friend links and total count
	opts := model.FriendLinkQueryOptions{
		Status: status,
		Search: search,
		Email:  email,
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
	h.getFriendLinks(c, false, "")
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

	h.getFriendLinks(c, false, email)
}

// GetFullFriendLinks handles GET /api/action/friend/ request (authenticated)
// It returns the full friend link data, including sensitive fields.
func (h *FriendLinkHandler) GetFullFriendLinks(c *gin.Context) {
	h.getFriendLinks(c, true, "")
}

// GetFullFriendLinkByID handles GET /api/action/friend/:id request (authenticated)
func (h *FriendLinkHandler) GetFullFriendLinkByID(c *gin.Context) {
	h.getFriendLinkByID(c, true)
}

// ApplyFriendLink handles POST /api/public/friend/apply.
// It creates a pending friend link submission from an anonymous visitor.
func (h *FriendLinkHandler) ApplyFriendLink(c *gin.Context) {
	var req model.FriendLinkApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Link) == "" || strings.TrimSpace(req.Avatar) == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "name, link and avatar are required"))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid email address"))
			return
		}
	}

	link := strings.TrimRight(req.Link, "/")
	link = strings.TrimSpace(link)
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "link must start with http:// or https://"))
		return
	}

	submission := model.FriendWebsite{
		Name:           req.Name,
		Link:           link,
		Avatar:         req.Avatar,
		Info:           req.Description,
		Email:          req.Email,
		Status:         "pending",
		EnableRss:      req.EnableRss,
		Snapshot:       req.Snapshot,
		FriendLinkPage: req.FriendLinkPage,
		Feed:           req.Feed,
	}

	id, err := friendsRepositories.CreateFriendLink(h.DB, submission)
	if err != nil {
		log.Printf("[handler][friend][apply] failed to create application: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to submit friend link application"))
		return
	}

	go service.NotifyFriendLinkApplication(submission)

	c.JSON(http.StatusCreated, model.NewSuccessResponseWithCode(http.StatusCreated, gin.H{
		"id":      id,
		"status":  "pending",
		"message": "友链申请已提交，等待管理员审核",
	}))

	fcircle.ScheduleRegenerate()
}

// UpdateApplyFriendLink handles POST /api/public/friend/update-apply.
// It creates a pending update request. If an existing friend link with the
// same email and original link is found, the new data is stored as a pending
// update record linked to that friend link.
func (h *FriendLinkHandler) UpdateApplyFriendLink(c *gin.Context) {
	var req model.FriendLinkUpdateApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	originalLink := strings.TrimRight(strings.TrimSpace(req.OriginalURL), "/")
	if originalLink == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Link) == "" || strings.TrimSpace(req.Avatar) == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "original_url, name, link and avatar are required"))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid email address"))
			return
		}
	}

	newLink := strings.TrimRight(strings.TrimSpace(req.Link), "/")
	if !strings.HasPrefix(newLink, "http://") && !strings.HasPrefix(newLink, "https://") {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "link must start with http:// or https://"))
		return
	}

	existing, err := friendsRepositories.GetFriendLinkByLink(h.DB, originalLink)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[handler][friend][update-apply] failed to query existing link: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to query existing friend link"))
		return
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "未找到原友链记录，请检查原站点地址"))
		return
	}

	if existing.Email != "" && existing.Email != req.Email {
		c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "邮箱与原友链登记邮箱不一致"))
		return
	}

	// Store the update request as a new pending application. The admin can
	// inspect the original ID and apply the changes through the existing panel.
	submission := model.FriendWebsite{
		Name:           req.Name,
		Link:           newLink,
		Avatar:         req.Avatar,
		Info:           req.Description,
		Email:          req.Email,
		Status:         "pending",
		EnableRss:      req.EnableRss,
		Snapshot:       req.Snapshot,
		FriendLinkPage: req.FriendLinkPage,
		Feed:           req.Feed,
	}

	id, err := friendsRepositories.CreateFriendLink(h.DB, submission)
	if err != nil {
		log.Printf("[handler][friend][update-apply] failed to create update application: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to submit update application"))
		return
	}

	go service.NotifyFriendLinkApplication(submission)

	c.JSON(http.StatusCreated, model.NewSuccessResponseWithCode(http.StatusCreated, gin.H{
		"id":           id,
		"original_id":  existing.ID,
		"status":       "pending",
		"message":      "友链更新申请已提交，等待管理员审核",
	}))

	fcircle.ScheduleRegenerate()
}

// GetFriendSubmissions handles GET /api/public/friend/submissions.
// It returns a public list of pending/approved/rejected friend link applications
// without exposing sensitive fields.
func (h *FriendLinkHandler) GetFriendSubmissions(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid page parameter"))
		return
	}

	pageSizeStr := c.DefaultQuery("page_size", "12")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid page_size parameter"))
		return
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Validate status parameter if provided
	validStatuses := map[string]bool{
		"survival": true,
		"timeout":  true,
		"error":    true,
		"pending":  true,
		"rejected": true,
	}
	if status != "" && !validStatuses[status] {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid status parameter"))
		return
	}

	offset := (page - 1) * pageSize
	opts := model.FriendLinkQueryOptions{
		Status: status,
		Search: search,
		Offset: offset,
		Limit:  pageSize,
	}

	resp, err := friendsRepositories.QueryFriendLinks(h.DB, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve submissions"))
		return
	}

	submissions := make([]model.FriendLinkSubmission, 0, len(resp.Links))
	for _, link := range resp.Links {
		submissions = append(submissions, model.FriendLinkSubmission{
			ID:          link.ID,
			Name:        link.Name,
			Description: link.Info,
			Status:      link.Status,
			UpdatedAt:   link.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{
		"submissions": submissions,
		"total":       resp.Count,
		"page":        page,
		"page_size":   pageSize,
	}))
}

// RecheckFriendLink handles POST /api/action/friend/:id/recheck.
// It performs one inspection even when scheduled health checks are disabled.
func (h *FriendLinkHandler) RecheckFriendLink(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "invalid friend link ID"))
		return
	}

	link, err := friendsRepositories.GetFriendLinkByID(h.DB, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, "friend link not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "failed to retrieve friend link"))
		return
	}

	result := crawlerService.CrawlWebsite(c.Request.Context(), link.Link)
	if c.Request.Context().Err() != nil {
		return
	}
	if err := friendsRepositories.UpdateFriendLink(h.DB, link, result); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "failed to update friend link inspection"))
		return
	}

	if link.EnableRss {
		for _, rssURL := range result.RssURLs {
			name, err := crawlerService.GetRssTitle(rssURL)
			if err != nil {
				log.Printf("[handler][friend] failed to get RSS title %s: %v", rssURL, err)
				continue
			}
			if _, err := friendsRepositories.CreateFriendRssFeeds(h.DB, link.ID, rssURL, name); err != nil {
				log.Printf("[handler][friend] failed to add RSS feed %s for friend link %d: %v", rssURL, link.ID, err)
			}
		}
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}
