package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"wms/backend/internal/user"
)

func TestJWTSignAndVerify(t *testing.T) {
	jwt := NewJWT("0123456789abcdef0123456789abcdef", time.Hour)
	u := &user.User{ID: uuid.Must(uuid.NewV7()), Role: user.RoleWarehouseAdmin}
	token, err := jwt.Sign(u)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := jwt.Parse(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != u.ID.String() || claims.Role != user.RoleWarehouseAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTRejectsBadToken(t *testing.T) {
	jwt := NewJWT("0123456789abcdef0123456789abcdef", time.Hour)
	if _, err := jwt.Parse("not-a-token"); err == nil {
		t.Fatal("expected verification error")
	}
}
