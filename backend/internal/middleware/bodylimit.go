package middleware

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var imageUploadPath = regexp.MustCompile(`^/api/v1/(products|users)/[0-9a-fA-F-]+/image$`)

func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

func BodyLimitResolver(defaultMaxBytes, imageMaxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		maxBytes := defaultMaxBytes
		if c.Request.Method == http.MethodPost && imageUploadPath.MatchString(c.Request.URL.Path) {
			maxBytes = imageMaxBytes
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
