package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Meta  json.RawMessage `json:"meta"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type authData struct {
	Token string `json:"token"`
	User  struct {
		Role string `json:"role"`
	} `json:"user"`
}

func loginToken(t *testing.T, env *testkit.Env, email, password string) string {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	}, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d %s", email, rec.Code, rec.Body.String())
	}
	var res struct {
		Data authData `json:"data"`
	}
	mustDecode(t, rec, &res)
	if res.Data.Token == "" {
		t.Fatalf("missing token for %s", email)
	}
	return res.Data.Token
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response %d %s: %v", rec.Code, rec.Body.String(), err)
	}
}

func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected %d got %d: %s", want, rec.Code, rec.Body.String())
	}
}

func requireErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var res envelope
	mustDecode(t, rec, &res)
	if res.Error == nil || res.Error.Code != want {
		t.Fatalf("expected error %s got %+v body=%s", want, res.Error, rec.Body.String())
	}
}

func idemKey() map[string]string {
	return map[string]string{"Idempotency-Key": "idem-" + uuid.NewString()}
}

func idemHeaders(key string) map[string]string {
	return map[string]string{"Idempotency-Key": key}
}

/* helper reserved for streaming/body tests */
