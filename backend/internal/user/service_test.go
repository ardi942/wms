package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/audit"
	"wms/backend/internal/httpx"
)

func TestUpdateRejectsDirectStatusChange(t *testing.T) {
	active := false
	svc := &Service{}
	_, err := svc.Update(context.Background(), uuid.Nil, UpdateInput{IsActive: &active}, audit.Actor{})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION" {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
	if _, ok := appErr.Details.(map[string]string)["is_active"]; !ok {
		t.Fatalf("expected is_active field error, got %#v", appErr.Details)
	}
}

func TestStatusChangeRequiresReason(t *testing.T) {
	svc := &Service{}
	_, _, err := svc.Deactivate(context.Background(), uuid.Nil, "   ", audit.Actor{})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION" {
		t.Fatalf("expected validation AppError, got %#v", err)
	}
	if _, ok := appErr.Details.(map[string]string)["reason"]; !ok {
		t.Fatalf("expected reason field error, got %#v", appErr.Details)
	}
}

func TestUserSnapshotNeverIncludesPasswordData(t *testing.T) {
	departmentID := uuid.Must(uuid.NewV7())
	snapshot := userSnapshot(&User{
		Email:        "staff@wms.local",
		PasswordHash: "super-secret-hash",
		FullName:     "Budi Santoso",
		Role:         RoleDepartmentUser,
		DepartmentID: &departmentID,
		IsActive:     true,
	})
	for key := range snapshot {
		if strings.Contains(strings.ToLower(key), "password") {
			t.Fatalf("snapshot contains password field %q", key)
		}
	}
	for _, value := range snapshot {
		if text, ok := value.(string); ok && strings.Contains(text, "super-secret-hash") {
			t.Fatal("snapshot contains password hash")
		}
	}
	if snapshot["email"] != "staff@wms.local" || snapshot["full_name"] != "Budi Santoso" || snapshot["is_active"] != true {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}
