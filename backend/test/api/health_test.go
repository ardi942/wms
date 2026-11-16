package api_test

import (
	"net/http"
	"testing"

	"wms/backend/internal/testkit"
)

func TestHealthAndReadiness(t *testing.T) {
	env := testkit.New(t)
	requireStatus(t, env.Request(http.MethodGet, "/healthz", nil, "", nil), http.StatusOK)
	rec := env.Request(http.MethodGet, "/readyz", nil, "", nil)
	requireStatus(t, rec, http.StatusOK)
	var body map[string]any
	mustDecode(t, rec, &body)
	if body["status"] == "" {
		t.Fatalf("missing readiness status: %#v", body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	env := testkit.New(t)
	rec := env.Request(http.MethodDelete, "/api/v1/auth/login", nil, "", nil)
	requireStatus(t, rec, http.StatusMethodNotAllowed)
	requireErrorCode(t, rec, "METHOD_NOT_ALLOWED")
}

func TestRouteNotFound(t *testing.T) {
	env := testkit.New(t)
	rec := env.Request(http.MethodGet, "/api/v1/nope", nil, "", nil)
	requireStatus(t, rec, http.StatusNotFound)
}
