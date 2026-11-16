package api_test

import (
	"net/http"
	"testing"

	"wms/backend/internal/testkit"
)

func TestLoginInvalidCredentials(t *testing.T) {
	env := testkit.New(t)
	rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "admin@example.com", "password": "wrong",
	}, "", nil)
	requireStatus(t, rec, http.StatusUnauthorized)
	requireErrorCode(t, rec, "UNAUTHORIZED")
}

func TestLoginUnknownEmailDoesNotLeak(t *testing.T) {
	env := testkit.New(t)
	rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "nobody@example.com", "password": "wrong",
	}, "", nil)
	requireStatus(t, rec, http.StatusUnauthorized)
	requireErrorCode(t, rec, "UNAUTHORIZED")
}

func TestAuthRequired(t *testing.T) {
	env := testkit.New(t)
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/auth/me", nil, "", nil), http.StatusUnauthorized)
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/users", nil, "", nil), http.StatusUnauthorized)
}

func TestRBAC(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	supervisor := loginToken(t, env, "supervisor@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)

	requireStatus(t, env.Request(http.MethodGet, "/api/v1/users", nil, dept, nil), http.StatusForbidden)
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/users", nil, supervisor, nil), http.StatusForbidden)
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/users", nil, admin, nil), http.StatusOK)

	requireStatus(t, env.Request(http.MethodGet, "/api/v1/reports/inventory", nil, dept, nil), http.StatusForbidden)
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/reports/inventory", nil, supervisor, nil), http.StatusOK)

	requireStatus(t, env.Request(http.MethodGet, "/api/v1/audit", nil, dept, nil), http.StatusForbidden)
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/audit", nil, supervisor, nil), http.StatusOK)
}
