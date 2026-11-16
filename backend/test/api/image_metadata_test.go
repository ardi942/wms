package api_test

import (
	"net/http"
	"testing"

	"wms/backend/internal/testkit"
)

func TestAuthResponsesIncludeImageFields(t *testing.T) {
	env := testkit.New(t)

	rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "admin@example.com", "password": testkit.Password,
	}, "", nil)
	requireStatus(t, rec, http.StatusOK)

	var login struct {
		Data struct {
			Token string         `json:"token"`
			User  map[string]any `json:"user"`
		} `json:"data"`
	}
	mustDecode(t, rec, &login)
	if login.Data.Token == "" {
		t.Fatal("missing login token")
	}
	requireNullImageFields(t, login.Data.User)

	rec = env.Request(http.MethodGet, "/api/v1/auth/me", nil, login.Data.Token, nil)
	requireStatus(t, rec, http.StatusOK)

	var me struct {
		Data struct {
			User map[string]any `json:"user"`
		} `json:"data"`
	}
	mustDecode(t, rec, &me)
	requireNullImageFields(t, me.Data.User)
}

func requireNullImageFields(t *testing.T, user map[string]any) {
	t.Helper()
	for _, key := range []string{"image_url", "image_updated_at"} {
		value, ok := user[key]
		if !ok {
			t.Fatalf("missing %s in user response: %#v", key, user)
		}
		if value != nil {
			t.Fatalf("%s = %#v, want null", key, value)
		}
	}
}
