package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"blog_api/src/model"
	momentRepositories "blog_api/src/repositories/moment"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MomentReactionHandler handles reactions to moments.
type MomentReactionHandler struct {
	DB *gorm.DB
}

// AddReaction handles POST /api/public/moments/:id/reactions request.
func (h *MomentReactionHandler) AddReaction(c *gin.Context) {
	momentID, ok := parseMomentID(c)
	if !ok {
		return
	}

	var req model.MomentReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if !isValidReaction(req.Reaction) {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid reaction"))
		return
	}

	fingerprintID, ok := c.Get("fingerprint_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "fingerprint token is required"))
		return
	}

	reaction := &model.MomentReaction{
		MomentID:      momentID,
		FingerprintID: fingerprintID.(int),
		Reaction:      req.Reaction,
		CreatedAt:     time.Now().Unix(),
	}

	if err := momentRepositories.CreateMomentReaction(h.DB, reaction); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, model.NewErrorResponse(409, "reaction already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to add reaction"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

// DeleteReaction handles DELETE /api/public/moments/:id/reactions request.
func (h *MomentReactionHandler) DeleteReaction(c *gin.Context) {
	momentID, ok := parseMomentID(c)
	if !ok {
		return
	}

	var req model.MomentReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid request body"))
		return
	}

	if !isValidReaction(req.Reaction) {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid reaction"))
		return
	}

	fingerprintID, ok := c.Get("fingerprint_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "fingerprint token is required"))
		return
	}

	if err := momentRepositories.DeleteMomentReaction(h.DB, momentID, fingerprintID.(int), req.Reaction); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewErrorResponse(404, "reaction not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to delete reaction"))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(nil))
}

func parseMomentID(c *gin.Context) (int, bool) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid moment id"))
		return 0, false
	}
	return id, true
}

func isValidReaction(reaction string) bool {
	switch reaction {
	case "👍", "👎", "❤", "👀", "💩":
		return true
	default:
		return false
	}
}
