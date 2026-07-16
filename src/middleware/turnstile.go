package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"blog_api/src/config"
	"blog_api/src/model"
	"blog_api/src/service"

	"github.com/gin-gonic/gin"
)

type turnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
}

const maxTurnstileRequestBytes = int64(1 << 20)

// TurnstileVerify validates a Turnstile token before continuing.
func TurnstileVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.GetConfig()
		if !cfg.Verify.Turnstile.Enable || cfg.Verify.Turnstile.Secret == "" {
			c.Next()
			return
		}

		token := extractTurnstileToken(c)
		if token == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "turnstile token is required"))
			c.Abort()
			return
		}

		if ok := verifyTurnstile(c, cfg.Verify.Turnstile.Secret, token); !ok {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "turnstile verification failed"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// AntiBotAuth validates the short-lived anti-bot token.
func AntiBotAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractAntiBotToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "antibot token is required"))
			c.Abort()
			return
		}

		if !service.ValidateAntiBotToken(token) {
			c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "antibot token is invalid"))
			c.Abort()
			return
		}

		c.Set("antibot_token", token)
		c.Next()
	}
}

func extractAntiBotToken(c *gin.Context) string {
	if token := c.GetHeader("X-Antibot-Token"); token != "" {
		return token
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	return ""
}

func verifyTurnstile(c *gin.Context, secret, token string) bool {
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if ip := c.ClientIP(); ip != "" {
		form.Set("remoteip", ip)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", form)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	return result.Success
}

func extractTurnstileToken(c *gin.Context) string {
	if token := c.GetHeader("X-Turnstile-Token"); token != "" {
		return token
	}
	if token := c.GetHeader("CF-Turnstile-Token"); token != "" {
		return token
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxTurnstileRequestBytes)
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if token, ok := payload["turnstile_token"].(string); ok {
		return token
	}
	if token, ok := payload["token"].(string); ok {
		return token
	}

	return ""
}
