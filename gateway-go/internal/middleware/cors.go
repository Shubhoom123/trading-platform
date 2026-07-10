package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows a browser frontend (served from a different origin during dev,
// e.g. Vite on :5173) to call the gateway. `allowedOrigin` is echoed back;
// pass "*" to allow any origin. Preflight OPTIONS requests are short-circuited.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := allowedOrigin
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "600")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
