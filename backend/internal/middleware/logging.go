package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"wms/backend/internal/httpx"
)

func Logging(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		attrs := []any{
			"request_id", GetRequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if id := httpx.CurrentIdentity(c); id != nil {
			attrs = append(attrs, "user_id", id.ID.String())
		}
		if status >= 500 {
			log.Error("request", attrs...)
			return
		}
		log.Info("request", attrs...)
	}
}
