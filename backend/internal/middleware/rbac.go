package middleware

import (
	"github.com/gin-gonic/gin"
	"wms/backend/internal/httpx"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		id := httpx.CurrentIdentity(c)
		if id == nil {
			httpx.WriteError(c, httpx.Unauthorized("authentication required"))
			return
		}
		if !allowed[id.Role] {
			httpx.WriteError(c, httpx.Forbidden("you do not have permission to perform this action"))
			return
		}
		c.Next()
	}
}
