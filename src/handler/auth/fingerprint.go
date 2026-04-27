package authHandler

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"blog_api/src/repositories"
	"blog_api/src/service"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FingerprintHandler handles fingerprint related requests.
type FingerprintHandler struct {
	DB *gorm.DB
}

// NewFingerprintHandler creates a new fingerprint handler.
func NewFingerprintHandler(db *gorm.DB) *FingerprintHandler {
	return &FingerprintHandler{DB: db}
}

// CreateFingerprint handles POST /api/verify/fingerprint request.
func (h *FingerprintHandler) CreateFingerprint(c *gin.Context) {
	cfg := config.GetConfig()
	secret := cfg.Verify.Fingerprint.Secret
	if secret == "" {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "fingerprint secret is not configured"))
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	tokenService := service.NewFingerprintTokenService(secret)

	if token, ok, err := h.reuseFingerprintToken(c, tokenService, ip, userAgent, secret); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to update fingerprint"))
		return
	} else if ok {
		c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{
			"fingerprint_token": token,
		}))
		return
	}

	fingerprintValue := hashFingerprint(ip, userAgent, secret)

	record, err := repositories.GetFingerprintByValue(h.DB, fingerprintValue)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to query fingerprint"))
			return
		}

		record = &model.Fingerprint{
			Fingerprint:      fingerprintValue,
			UserAgent:        userAgent,
			IP:               ip,
			PermissionsLevel: "normal",
			CreatedAt:        time.Now().Unix(),
		}
		if err := repositories.CreateFingerprint(h.DB, record); err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to create fingerprint"))
			return
		}
	}

	token := tokenService.Sign(record.ID)

	c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{
		"fingerprint_token": token,
	}))
}

func (h *FingerprintHandler) reuseFingerprintToken(
	c *gin.Context,
	tokenService *service.FingerprintTokenService,
	ip, userAgent, secret string,
) (string, bool, error) {
	token := extractFingerprintToken(c)
	if token == "" {
		return "", false, nil
	}

	id, ok := tokenService.Verify(token)
	if !ok || id <= 0 {
		return "", false, nil
	}

	record, err := repositories.GetFingerprintByID(h.DB, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	fingerprintValue := hashFingerprint(ip, userAgent, secret)
	if record.Fingerprint == fingerprintValue && record.UserAgent == userAgent && record.IP == ip {
		return token, true, nil
	}

	if err := repositories.UpdateFingerprintIdentity(h.DB, record.ID, fingerprintValue, userAgent, ip); err != nil {
		return "", false, err
	}

	return token, true, nil
}

func hashFingerprint(ip, userAgent, secret string) string {
	sum := sha256.Sum256([]byte(ip + "|" + userAgent + "|" + secret))
	return hex.EncodeToString(sum[:])
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
