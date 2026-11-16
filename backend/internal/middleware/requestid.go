package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/httpx"
)

const RequestIDKey = httpx.RequestIDKey

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !validHeaderValue(id, 128) {
			id = uuid.NewString()
		}
		c.Set(RequestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func validHeaderValue(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == ':':
		default:
			return false
		}
	}
	return true
}

func GetRequestID(c *gin.Context) string {
	return httpx.RequestID(c)
}
