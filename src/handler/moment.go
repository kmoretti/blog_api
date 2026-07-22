package handler

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"blog_api/src/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MomentHandler handles moment related requests
type MomentHandler struct {
	DB *gorm.DB
}

// GetMoments handles GET /api/moments request
func (h *MomentHandler) GetMoments(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid page parameter"))
		return
	}

	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "invalid page_size parameter"))
		return
	}

	if pageSize > 100 {
		pageSize = 100
	}

	var fingerprintID *int
	if id, ok := parseFingerprintID(c); ok {
		fingerprintID = &id
	}

	resp, err := service.GetMomentsWithMedia(h.DB, page, pageSize, "visible", fingerprintID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to retrieve moments"))
		return
	}

	publicMoments := make([]model.PublicMomentWithMedia, len(resp.Moments))
	for i, moment := range resp.Moments {
		publicMoments[i] = model.PublicMomentWithMedia{
			ID:               moment.ID,
			Content:          moment.Content,
			Status:           moment.Status,
			Tags:             moment.Tags,
			PinnedOrder:      moment.PinnedOrder,
			IsAd:             moment.IsAd,
			Extension:        moment.Extension,
			MessageLink:      moment.MessageLink,
			CreatedAt:        moment.CreatedAt,
			UpdatedAt:        moment.UpdatedAt,
			Media:            moment.Media,
			Reactions:        moment.Reactions,
			SelectedReaction: moment.SelectedReaction,
		}
	}

	paginatedData := model.PaginatedResponse{
		Items:    publicMoments,
		Total:    int(resp.Total),
		Page:     page,
		PageSize: pageSize,
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(paginatedData))
}

func parseFingerprintID(c *gin.Context) (int, bool) {
	secret := config.GetConfig().Verify.Fingerprint.Secret
	if secret == "" {
		return 0, false
	}

	token := extractFingerprintToken(c)
	if token == "" {
		return 0, false
	}

	tokenService := service.NewFingerprintTokenService(secret)
	id, ok := tokenService.Verify(token)
	if !ok || id <= 0 {
		return 0, false
	}

	return id, true
}

func extractFingerprintToken(c *gin.Context) string {
	if token := c.GetHeader("X-Fingerprint-Token"); token != "" {
		return token
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Fingerprint") {
			return parts[1]
		}
	}

	return ""
}
