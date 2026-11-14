package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/auth"
	"wms/backend/internal/httpx"
	"wms/backend/internal/user"
)

func Auth(jwtSvc *auth.JWT, users *user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			httpx.WriteError(c, httpx.Unauthorized("authentication required"))
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := jwtSvc.Parse(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Unauthorized("invalid or expired token"))
			return
		}
		uid, err := uuid.Parse(claims.Subject)
		if err != nil {
			httpx.WriteError(c, httpx.Unauthorized("invalid token subject"))
			return
		}
		u, err := users.GetByID(c.Request.Context(), uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				httpx.WriteError(c, httpx.Unauthorized("user no longer exists"))
				return
			}
			httpx.WriteError(c, err)
			return
		}
		if !u.IsActive {
			httpx.WriteError(c, httpx.Unauthorized("user is inactive"))
			return
		}
		id := &httpx.Identity{
			ID:           u.ID,
			Email:        u.Email,
			FullName:     u.FullName,
			Role:         u.Role,
			DepartmentID: u.DepartmentID,
		}
		if u.DepartmentID != nil {
			if d, err := users.GetDepartment(c.Request.Context(), *u.DepartmentID); err == nil {
				id.DepartmentCode = d.Code
				id.DepartmentName = d.Name
			}
		}
		httpx.SetIdentity(c, id)
		c.Next()
	}
}
