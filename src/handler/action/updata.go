package handlerAction

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdataHandler handles updata related requests
type UpdataHandler struct {
	DB *gorm.DB
}

// CreateFriendLink handles POST /api/updata/friend request
func (h *UpdataHandler) CreateFriendLink(c *gin.Context) {
	log.Println("[handler][updata] Received friend link creation request")
	var req model.FriendWebsite
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[handler][updata][ERR] JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	log.Printf("[handler][updata] Received friend link data: %+v", req)

	if authType, ok := c.Get("auth_type"); ok && authType == "email" {
		authEmail, _ := c.Get("auth_email")
		email, _ := authEmail.(string)
		if email == "" || (req.Email != "" && req.Email != email) {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email does not match the token"))
			return
		}
		req.Email = email
		req.Status = "pending"
	} else {
		req.Status = "survival"
	}

	// Insert into database
	id, err := friendsRepositories.CreateFriendLink(h.DB, req)
	if err != nil {
		log.Printf("[handler][updata][ERR] 创建友情链接失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to create friend link"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"id": id}))
}

// DeleteOwnedFriendLink handles deletion by an email-authenticated owner.
func (h *UpdataHandler) DeleteOwnedFriendLink(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid friend link ID"))
		return
	}
	authEmail, _ := c.Get("auth_email")
	email, _ := authEmail.(string)
	if email == "" {
		c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email token is invalid"))
		return
	}

	link, err := friendsRepositories.GetFriendLinkByID(h.DB, int(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "friend link not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link"))
		return
	}
	if link.Email != email {
		c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "friend link does not belong to this email"))
		return
	}

	deleted, err := friendsRepositories.DeleteFriendLinkByOwner(h.DB, uint(id), email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "friend link ownership changed"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete friend link"))
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"deleted_link": gin.H{
		"id":          deleted.ID,
		"name":        deleted.Name,
		"link":        deleted.Link,
		"avatar":      deleted.Avatar,
		"description": deleted.Info,
		"status":      deleted.Status,
		"enable_rss":  deleted.EnableRss,
		"updated_at":  deleted.UpdatedAt,
	}}))
}

// DeleteFriendLink handles DELETE /api/action/friend/:id request
func (h *UpdataHandler) DeleteFriendLink(c *gin.Context) {
	log.Println("[handler][updata] Received friend link deletion request")
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid friend link ID"))
		return
	}

	log.Printf("[handler][updata] Received friend link deletion request for ID: %d", id)

	deletedLink, err := friendsRepositories.DeleteFriendLinkByID(h.DB, uint(id))
	if err != nil {
		log.Printf("[handler][updata][ERR] 删除友情链接失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete friend link"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"deleted_link": deletedLink}))
}

// EditFriendLink handles PUT /api/action/friend/:id request
func (h *UpdataHandler) EditFriendLink(c *gin.Context) {
	log.Println("[handler][updata] Received friend link edit request")
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid friend link ID"))
		return
	}

	var req model.EditFriendLinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[handler][updata][ERR] JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	log.Printf("[handler][updata] Received friend link edit data for ID %d: %+v", id, req)

	if authType, ok := c.Get("auth_type"); ok && authType == "email" {
		ownerFields := map[string]bool{
			"website_name":     true,
			"website_url":      true,
			"website_icon_url": true,
			"description":      true,
			"enable_rss":       true,
		}
		for field := range req.Data {
			if !ownerFields[field] {
				c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "field cannot be updated with this token: "+field))
				return
			}
		}
		authEmail, _ := c.Get("auth_email")
		email, _ := authEmail.(string)
		link, err := friendsRepositories.GetFriendLinkByID(h.DB, int(id))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "friend link not found"))
				return
			}
			log.Printf("[handler][updata][ERR] 查询友情链接失败: %v", err)
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve friend link"))
			return
		}
		if link.Email == "" {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email is not set for this friend link"))
			return
		}
		if email == "" || link.Email != email {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email does not match the token"))
			return
		}

		rowsAffected, err := friendsRepositories.UpdateFriendLinkByOwner(h.DB, uint(id), email, req)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "friend link ownership changed"))
				return
			}
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
		return
	}

	if value, exists := req.Data["skip_health_check"]; exists {
		if _, ok := value.(bool); !ok {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "skip_health_check must be a boolean"))
			return
		}
	}

	rowsAffected, err := friendsRepositories.UpdateFriendLinkByID(h.DB, uint(id), req)
	if err != nil {
		log.Printf("[handler][updata][ERR] 更新友情链接失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update friend link"))
		return
	}

	if rowsAffected == 0 {
		log.Printf("[handler][updata] No friend link found with ID %d or no fields to update", id)
		c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "no friend link found with the given ID or no fields needed update"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
}
