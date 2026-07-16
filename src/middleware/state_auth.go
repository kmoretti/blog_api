package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"blog_api/src/model"

	"github.com/gin-gonic/gin"
)

// StateMasterAuth authenticates server-to-server state API requests.
//
// Requests must carry the configured password as an Authorization Bearer token.
func StateMasterAuth(password string) gin.HandlerFunc {
	expected := sha256.Sum256([]byte(password))
	return func(c *gin.Context) {
		const prefix = "Bearer "
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, prefix) || len(header) == len(prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewErrorResponse(http.StatusUnauthorized, "unauthorized"))
			return
		}

		actual := sha256.Sum256([]byte(header[len(prefix):]))
		if subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewErrorResponse(http.StatusUnauthorized, "unauthorized"))
			return
		}
		c.Next()
	}
}
