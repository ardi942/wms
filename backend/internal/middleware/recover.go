package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms/backend/internal/httpx"
)

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "request_id", GetRequestID(c), "panic", r)
				httpx.WriteRawError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			}
		}()
		c.Next()
	}
}
