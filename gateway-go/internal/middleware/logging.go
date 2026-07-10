package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger emits one structured line per request. No request bodies or
// Authorization headers are logged.
func RequestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"client_ip", c.ClientIP(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}
