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
		if req.Email == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "email is required"))
			return
		}
		if email == "" || req.Email != email {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email does not match the token"))
			return
		}
		if _, err := friendsRepositories.GetFriendLinkByEmail(h.DB, req.Email); err == nil {
			c.JSON(http.StatusConflict, model.NewErrorResponse(409, "friend link already exists for this email"))
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[handler][updata][ERR] 查询友情链接失败: %v", err)
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to validate friend link"))
			return
		}
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
		if _, exists := req.Data["email"]; exists {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "email cannot be updated with this token"))
			return
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

	if authType, ok := c.Get("auth_type"); ok && authType == "email" {
		if err := friendsRepositories.DeleteRssDataByFriendLinkID(h.DB, int(id)); err != nil {
			log.Printf("[handler][updata][ERR] 删除 RSS 失败: %v", err)
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete rss data"))
			return
		}
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}))
}
