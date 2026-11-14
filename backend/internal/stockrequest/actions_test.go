package stockrequest

import (
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/user"
)

func TestAllowedActionsRequest(t *testing.T) {
	dept := uuid.New()
	other := uuid.New()
	cases := []struct {
		name   string
		status string
		role   string
		actor  *uuid.UUID
		dept   uuid.UUID
		want   int
	}{
		{"admin pending", StatusPending, user.RoleWarehouseAdmin, nil, dept, 2},
		{"admin partial", StatusPartiallyFulfilled, user.RoleWarehouseAdmin, nil, dept, 2},
		{"admin terminal", StatusFulfilled, user.RoleWarehouseAdmin, nil, dept, 0},
		{"dept owner pending", StatusPending, user.RoleDepartmentUser, &dept, dept, 1},
		{"dept other pending", StatusPending, user.RoleDepartmentUser, &other, dept, 0},
		{"dept owner partial", StatusPartiallyFulfilled, user.RoleDepartmentUser, &dept, dept, 0},
		{"supervisor", StatusPending, user.RoleSupervisor, nil, dept, 0},
	}
	for _, tc := range cases {
		got := allowedActions(tc.status, tc.role, tc.actor, tc.dept)
		if len(got) != tc.want {
			t.Fatalf("%s: got %v want %d actions", tc.name, got, tc.want)
		}
	}
}