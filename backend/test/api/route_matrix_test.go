package api_test

import (
	"net/http"
	"testing"

	"wms/backend/internal/testkit"
)

func TestRouteMatrixAuthAndRBAC(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	supervisor := loginToken(t, env, "supervisor@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)

	type tc struct {
		name   string
		method string
		path   string
		token  string
		want   int
	}
	cases := []tc{
		{"me unauth", http.MethodGet, "/api/v1/auth/me", "", http.StatusUnauthorized},
		{"users dept forbidden", http.MethodGet, "/api/v1/users", dept, http.StatusForbidden},
		{"users supervisor forbidden", http.MethodGet, "/api/v1/users", supervisor, http.StatusForbidden},
		{"users admin ok", http.MethodGet, "/api/v1/users", admin, http.StatusOK},
		{"audit dept forbidden", http.MethodGet, "/api/v1/audit", dept, http.StatusForbidden},
		{"audit supervisor ok", http.MethodGet, "/api/v1/audit", supervisor, http.StatusOK},
		{"audit admin ok", http.MethodGet, "/api/v1/audit", admin, http.StatusOK},
		{"inventory report dept forbidden", http.MethodGet, "/api/v1/reports/inventory", dept, http.StatusForbidden},
		{"inventory report supervisor ok", http.MethodGet, "/api/v1/reports/inventory", supervisor, http.StatusOK},
		{"low stock dept forbidden", http.MethodGet, "/api/v1/reports/low-stock", dept, http.StatusForbidden},
		{"movements report admin ok", http.MethodGet, "/api/v1/reports/stock-movements", admin, http.StatusOK},
		{"monitoring requests dept ok", http.MethodGet, "/api/v1/monitoring/requests", dept, http.StatusOK},
		{"monitoring returns admin ok", http.MethodGet, "/api/v1/monitoring/returns", admin, http.StatusOK},
		{"dept list authed ok", http.MethodGet, "/api/v1/departments", dept, http.StatusOK},
		{"products authed ok", http.MethodGet, "/api/v1/products", dept, http.StatusOK},
	}
	for _, c := range cases {
		rec := env.Request(c.method, c.path, nil, c.token, nil)
		if rec.Code != c.want {
			t.Errorf("%s: want %d got %d body=%s", c.name, c.want, rec.Code, rec.Body.String())
		}
	}
}
