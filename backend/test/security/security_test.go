package security_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode %d %s: %v", rec.Code, rec.Body.String(), err)
	}
}

func login(t *testing.T, env *testkit.Env, email string) string {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": testkit.Password}, "", nil)
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var res struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	mustDecode(t, rec, &res)
	return res.Data.Token
}

func TestSQLInjectionPayloads(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"admin@example.com'; DROP TABLE users; --",
		"%' UNION SELECT password_hash FROM users --",
	}
	for _, p := range payloads {
		rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": p, "password": "nope"}, "", nil)
		if rec.Code >= 500 {
			t.Fatalf("login payload %q caused %d %s", p, rec.Code, rec.Body.String())
		}
		if strings.Contains(strings.ToLower(rec.Body.String()), "sqlstate") || strings.Contains(rec.Body.String(), "pq:") {
			t.Fatalf("login leaked db error for %q: %s", p, rec.Body.String())
		}
		rec = env.Request(http.MethodGet, "/api/v1/products?search="+strings.ReplaceAll(p, " ", "%20"), nil, admin, nil)
		if rec.Code >= 500 {
			t.Fatalf("search payload %q caused %d %s", p, rec.Code, rec.Body.String())
		}
	}
	var userCount int64
	if err := env.DB.Table("users").Count(&userCount).Error; err != nil || userCount == 0 {
		t.Fatalf("users table damaged: count=%d err=%v", userCount, err)
	}
}

func TestJWTTampering(t *testing.T) {
	env := testkit.New(t)
	token := login(t, env, "admin@example.com")
	bad := token[:len(token)-2] + "aa"
	rec := env.Request(http.MethodGet, "/api/v1/auth/me", nil, bad, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestXSSStoredAsText(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	name := `<script>alert(1)</script>`
	rec := env.Request(http.MethodPost, "/api/v1/products", map[string]any{
		"sku": "XSS-001", "name": name, "category_id": env.CategoryID.String(), "unit_id": env.UnitID.String(),
	}, admin, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: %d %s", rec.Code, rec.Body.String())
	}
	var res struct {
		Data struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		} `json:"data"`
	}
	mustDecode(t, rec, &res)
	if res.Data.Name != name {
		t.Fatalf("name mutated: got %q", res.Data.Name)
	}
	rec = env.Request(http.MethodGet, "/api/v1/products/"+res.Data.ID.String(), nil, admin, nil)
	if !strings.Contains(rec.Body.String(), `<script>alert(1)</script>`) && !strings.Contains(rec.Body.String(), `\u003cscript\u003e`) {
		t.Fatalf("expected payload in json body: %s", rec.Body.String())
	}
}
