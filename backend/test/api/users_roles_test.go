package api_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

type userResponse struct {
	Data struct {
		ID         uuid.UUID `json:"id"`
		Email      string    `json:"email"`
		Role       string    `json:"role"`
		Department *struct {
			ID uuid.UUID `json:"id"`
		} `json:"department"`
		IsActive bool `json:"is_active"`
	} `json:"data"`
}

func createUser(t *testing.T, env *testkit.Env, token string, body map[string]any) userResponse {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/users", body, token, nil)
	requireStatus(t, rec, http.StatusCreated)
	var res userResponse
	mustDecode(t, rec, &res)
	return res
}

func elevatedBody(email, role string) map[string]any {
	return map[string]any{
		"email": email, "full_name": "Elevated User", "password": "Password123!", "role": role,
	}
}

func TestCreateElevatedUsers(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)

	supervisor := createUser(t, env, admin, elevatedBody("new.supervisor@example.com", "SUPERVISOR"))
	if supervisor.Data.Role != "SUPERVISOR" || supervisor.Data.Department != nil {
		t.Fatalf("unexpected supervisor: %+v", supervisor.Data)
	}
	warehouseAdmin := createUser(t, env, admin, elevatedBody("new.admin@example.com", "WAREHOUSE_ADMIN"))
	if warehouseAdmin.Data.Role != "WAREHOUSE_ADMIN" || warehouseAdmin.Data.Department != nil {
		t.Fatalf("unexpected admin: %+v", warehouseAdmin.Data)
	}
	lowercase := createUser(t, env, admin, elevatedBody("lower.supervisor@example.com", "supervisor"))
	if lowercase.Data.Role != "SUPERVISOR" {
		t.Fatalf("expected normalized role, got %+v", lowercase.Data)
	}
	department := createUser(t, env, admin, map[string]any{
		"email": "new.department@example.com", "full_name": "Department User", "password": "Password123!",
		"department_id": env.ITDeptID.String(),
	})
	if department.Data.Role != "DEPARTMENT_USER" || department.Data.Department == nil {
		t.Fatalf("expected department user, got %+v", department.Data)
	}
}

func TestCreateUserRoleValidation(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"invalid role", elevatedBody("bad.role@example.com", "OWNER")},
		{"department user without department", map[string]any{
			"email": "no.dept@example.com", "full_name": "No Department", "password": "Password123!",
		}},
		{"elevated with department", map[string]any{
			"email": "admin.with.dept@example.com", "full_name": "Admin With Dept", "password": "Password123!",
			"role": "WAREHOUSE_ADMIN", "department_id": env.ITDeptID.String(),
		}},
		{"elevated with empty department", map[string]any{
			"email": "admin.empty.dept@example.com", "full_name": "Admin Empty Dept", "password": "Password123!",
			"role": "SUPERVISOR", "department_id": "",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.Request(http.MethodPost, "/api/v1/users", tc.body, admin, nil)
			requireStatus(t, rec, http.StatusUnprocessableEntity)
		})
	}
}

func TestListUsersByRole(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	createUser(t, env, admin, elevatedBody("listed.supervisor@example.com", "SUPERVISOR"))

	var all struct {
		Data []struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	rec := env.Request(http.MethodGet, "/api/v1/users?page_size=100", nil, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	mustDecode(t, rec, &all)
	if all.Meta.Total < 5 {
		t.Fatalf("expected all seeded and created users, got %d", all.Meta.Total)
	}
	seen := map[string]string{}
	for _, u := range all.Data {
		seen[u.Email] = u.Role
	}
	if seen["admin@example.com"] != "WAREHOUSE_ADMIN" || seen["supervisor@example.com"] != "SUPERVISOR" ||
		seen["dept@example.com"] != "DEPARTMENT_USER" || seen["listed.supervisor@example.com"] != "SUPERVISOR" {
		t.Fatalf("missing roles in default list: %#v", seen)
	}

	var filtered struct {
		Data []struct {
			Role string `json:"role"`
		} `json:"data"`
	}
	rec = env.Request(http.MethodGet, "/api/v1/users?role=supervisor&page_size=100", nil, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	mustDecode(t, rec, &filtered)
	if len(filtered.Data) == 0 {
		t.Fatal("expected supervisor rows")
	}
	for _, u := range filtered.Data {
		if u.Role != "SUPERVISOR" {
			t.Fatalf("unexpected role %s in filtered list", u.Role)
		}
	}

	rec = env.Request(http.MethodGet, "/api/v1/users?role=OWNER", nil, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestManageElevatedUser(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	created := createUser(t, env, admin, elevatedBody("managed.supervisor@example.com", "SUPERVISOR"))
	id := created.Data.ID.String()

	rec := env.Request(http.MethodPatch, "/api/v1/users/"+id, map[string]any{"full_name": "Managed Supervisor"}, admin, nil)
	requireStatus(t, rec, http.StatusOK)

	rec = env.Request(http.MethodPatch, "/api/v1/users/"+id, map[string]any{"department_id": env.ITDeptID.String()}, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	rec = env.Request(http.MethodPatch, "/api/v1/users/"+id, map[string]any{"password": "NewPassword123!"}, admin, nil)
	requireStatus(t, rec, http.StatusOK)

	rec = env.Request(http.MethodPost, "/api/v1/users/"+id+"/deactivate", map[string]any{"reason": "no longer needed"}, admin, nil)
	requireStatus(t, rec, http.StatusOK)

	var history struct {
		Data []any `json:"data"`
	}
	rec = env.Request(http.MethodGet, "/api/v1/users/"+id+"/history", nil, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	mustDecode(t, rec, &history)
	if len(history.Data) < 3 {
		t.Fatalf("expected creation, profile, password and status history, got %d", len(history.Data))
	}

	rec = env.Request(http.MethodPost, "/api/v1/users/"+id+"/activate", map[string]any{"reason": "account needed again"}, admin, nil)
	requireStatus(t, rec, http.StatusOK)
}

func TestConcurrentAdminDeactivationKeepsAdmin(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	second := createUser(t, env, admin, elevatedBody("second.admin@example.com", "WAREHOUSE_ADMIN"))
	secondToken := loginToken(t, env, "second.admin@example.com", "Password123!")

	var wg sync.WaitGroup
	start := make(chan struct{})
	codes := make(chan int, 2)
	targets := []struct {
		token string
		id    string
	}{
		{admin, second.Data.ID.String()},
		{secondToken, env.Users["admin"].ID.String()},
	}
	for _, target := range targets {
		wg.Add(1)
		go func(token, id string) {
			defer wg.Done()
			<-start
			rec := env.Request(http.MethodPost, "/api/v1/users/"+id+"/deactivate", map[string]any{"reason": "concurrent test"}, token, nil)
			codes <- rec.Code
		}(target.token, target.id)
	}
	close(start)
	wg.Wait()
	close(codes)

	for code := range codes {
		switch code {
		case http.StatusOK, http.StatusConflict, http.StatusUnauthorized:
		default:
			t.Fatalf("unexpected status %d", code)
		}
	}
	var active int64
	if err := env.DB.Raw(`SELECT COUNT(*) FROM users WHERE role = 'WAREHOUSE_ADMIN' AND is_active = true`).Scan(&active).Error; err != nil {
		t.Fatalf("count active admins: %v", err)
	}
	if active < 1 {
		t.Fatal("expected at least one active warehouse admin")
	}
}
