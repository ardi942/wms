package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const identityKey = "wms.identity"

type Identity struct {
	ID             uuid.UUID  `json:"id"`
	Email          string     `json:"email"`
	FullName       string     `json:"full_name"`
	Role           string     `json:"role"`
	DepartmentID   *uuid.UUID `json:"-"`
	DepartmentCode string     `json:"-"`
	DepartmentName string     `json:"-"`
}

func SetIdentity(c *gin.Context, id *Identity) { c.Set(identityKey, id) }

func CurrentIdentity(c *gin.Context) *Identity {
	v, ok := c.Get(identityKey)
	if !ok {
		return nil
	}
	id, _ := v.(*Identity)
	return id
}

func MustIdentity(c *gin.Context) *Identity {
	id := CurrentIdentity(c)
	if id == nil {
		panic("identity missing: auth middleware not applied")
	}
	return id
}
