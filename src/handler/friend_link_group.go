package handler

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"blog_api/src/service/fcircle"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FriendLinkGroupHandler handles friend link group related requests.
type FriendLinkGroupHandler struct {
	DB *gorm.DB
}

// GetFriendLinkGroups handles GET /api/action/friend/group.
func (h *FriendLinkGroupHandler) GetFriendLinkGroups(c *gin.Context) {
	groups, err := friendsRepositories.ListFriendLinkGroups(h.DB)
	if err != nil {
		log.Printf("[handler][friend_group] failed to list groups: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link groups"))
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse(groups))
}

// CreateFriendLinkGroup handles POST /api/action/friend/group.
func (h *FriendLinkGroupHandler) CreateFriendLinkGroup(c *gin.Context) {
	var req model.FriendLinkGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "group name is required"))
		return
	}
	if err := friendsRepositories.CreateFriendLinkGroup(h.DB, &req); err != nil {
		log.Printf("[handler][friend_group] failed to create group: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to create friend link group"))
		return
	}
	fcircle.ScheduleRegenerate()
	c.JSON(http.StatusCreated, model.NewSuccessResponseWithCode(http.StatusCreated, req))
}

// UpdateFriendLinkGroup handles PUT /api/action/friend/group/:id.
func (h *FriendLinkGroupHandler) UpdateFriendLinkGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid group id"))
		return
	}
	var req model.FriendLinkGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "group name is required"))
		return
	}
	if err := friendsRepositories.UpdateFriendLinkGroup(h.DB, id, &req); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "friend link group not found"))
			return
		}
		log.Printf("[handler][friend_group] failed to update group: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update friend link group"))
		return
	}
	fcircle.ScheduleRegenerate()
	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// DeleteFriendLinkGroup handles DELETE /api/action/friend/group/:id.
func (h *FriendLinkGroupHandler) DeleteFriendLinkGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid group id"))
		return
	}
	if err := friendsRepositories.DeleteFriendLinkGroup(h.DB, id); err != nil {
		log.Printf("[handler][friend_group] failed to delete group: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete friend link group"))
		return
	}
	fcircle.ScheduleRegenerate()
	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// SetFriendLinkGroups handles PUT /api/action/friend/:id/groups.
func (h *FriendLinkGroupHandler) SetFriendLinkGroups(c *gin.Context) {
	linkID, err := strconv.Atoi(c.Param("id"))
	if err != nil || linkID < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid friend link id"))
		return
	}
	var req struct {
		GroupIDs []int `json:"group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	exists, err := friendsRepositories.FriendLinkExists(h.DB, linkID)
	if err != nil {
		log.Printf("[handler][friend_group] failed to check friend link existence: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update friend link groups"))
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "friend link not found"))
		return
	}

	// If no groups specified, move to default group so the link is always visible.
	groupIDs := req.GroupIDs
	if len(groupIDs) == 0 {
		defaultID, err := friendsRepositories.EnsureDefaultFriendLinkGroup(h.DB)
		if err != nil {
			log.Printf("[handler][friend_group] failed to ensure default group: %v", err)
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update friend link groups"))
			return
		}
		groupIDs = []int{defaultID}
	}

	if err := friendsRepositories.SetFriendLinkGroups(h.DB, linkID, groupIDs); err != nil {
		log.Printf("[handler][friend_group] failed to set groups: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update friend link groups"))
		return
	}
	fcircle.ScheduleRegenerate()
	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// GetFriendLinkGroupIDs handles GET /api/action/friend/:id/groups.
func (h *FriendLinkGroupHandler) GetFriendLinkGroupIDs(c *gin.Context) {
	linkID, err := strconv.Atoi(c.Param("id"))
	if err != nil || linkID < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid friend link id"))
		return
	}
	ids, err := friendsRepositories.GetFriendLinkGroupIDs(h.DB, linkID)
	if err != nil {
		log.Printf("[handler][friend_group] failed to get group ids: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link groups"))
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"group_ids": ids}))
}

// GetFriendLinkJSON handles GET /friend.json.
// It returns survival friend links grouped by their assigned groups.
func (h *FriendLinkGroupHandler) GetFriendLinkJSON(c *gin.Context) {
	data, err := fcircle.GenerateFriendJSON(h.DB)
	if err != nil {
		log.Printf("[handler][friend_group] failed to generate friend.json: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to generate friend.json"))
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, data)
}

// MigrateFriendLinkGroups handles POST /api/action/friend/group/migrate.
// It ensures the default group exists and moves all ungrouped friend links into it.
func (h *FriendLinkGroupHandler) MigrateFriendLinkGroups(c *gin.Context) {
	if err := friendsRepositories.MigrateExistingFriendLinksToDefaultGroup(h.DB); err != nil {
		log.Printf("[handler][friend_group] migration failed: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to migrate friend link groups"))
		return
	}
	fcircle.ScheduleRegenerate()
	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"message": "migration completed"}))
}

// FriendLinkGroupReq represents a create/update request for a friend link group.
// Kept for backward compatibility with any callers that may expect this name.
type FriendLinkGroupReq = model.FriendLinkGroup

func nowUnix() int64 {
	return time.Now().Unix()
}
