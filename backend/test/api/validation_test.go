package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/config"
	"wms/backend/internal/testkit"
)

func TestUserCreateValidation(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	rec := env.Request(http.MethodPost, "/api/v1/users", map[string]any{}, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	requireErrorCode(t, rec, "VALIDATION")
}

func TestPatchUserRejectsIsActive(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	userID := env.Users["dept"].ID.String()
	rec := env.Request(http.MethodPatch, "/api/v1/users/"+userID, map[string]any{"is_active": false}, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestActivateRequiresReason(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	userID := env.Users["inactive"].ID.String()
	rec := env.Request(http.MethodPost, "/api/v1/users/"+userID+"/activate", map[string]any{"reason": "no"}, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	rec = env.Request(http.MethodPost, "/api/v1/users/"+userID+"/activate", map[string]any{"reason": "   "}, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestDateValidation(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	rec := env.Request(http.MethodGet, "/api/v1/stock-requests?date_from=2026-02-31", nil, dept, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	rec = env.Request(http.MethodGet, "/api/v1/stock-requests?date_from=2026-03-10&date_to=2026-03-01", nil, dept, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestInvalidUUIDPath(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	rec := env.Request(http.MethodGet, "/api/v1/products/not-a-uuid", nil, admin, nil)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 404/422 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestBodyLimit(t *testing.T) {
	env := testkit.New(t, testkit.WithConfig(func(c *config.Config) { c.MaxRequestBodyBytes = 1024 }))
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": strings.Repeat("a", 4096),
		"items":   []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}},
	}, dept, idemKey())
	requireStatus(t, rec, http.StatusRequestEntityTooLarge)
}

func TestCreateStockRequestValidation(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": "Need supplies",
		"items":   []any{},
	}, dept, idemKey())
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestSearchAndSortInjectionDoNot500(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	payloads := []string{"' OR 1=1 --", `"; DROP TABLE users; --`, `name;SELECT pg_sleep(1)--`}
	for _, p := range payloads {
		rec := env.Request(http.MethodGet, "/api/v1/products?search="+urlEscape(p)+"&sort="+urlEscape(p), nil, admin, nil)
		if rec.Code >= 500 {
			t.Fatalf("payload %q caused %d %s", p, rec.Code, rec.Body.String())
		}
	}
}

func urlEscape(s string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "'", "%27", ";", "%3B", "\"", "%22", "#", "%23", "=", "%3D", "-", "%2D")
	// minimal escaping adequate for query values in tests
	return r.Replace(s)
}

func TestDuplicateEmailReturns409(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	body := map[string]any{
		"email": "dept@example.com", "full_name": "Dup", "role": "DEPARTMENT_USER",
		"department_id": env.Users["dept"].DepartmentID.String(), "password": "Password123!",
	}
	rec := env.Request(http.MethodPost, "/api/v1/users", body, admin, nil)
	if rec.Code != http.StatusConflict && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 409/422 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestUnknownProductUUID(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": "Need supplies",
		"items":   []map[string]any{{"product_id": uuid.NewString(), "requested_qty": 1}},
	}, dept, idemKey())
	if rec.Code >= 500 {
		t.Fatalf("unexpected 5xx: %d %s", rec.Code, rec.Body.String())
	}
}
