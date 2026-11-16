package api_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

func TestIdempotencyReplayAndConflict(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	key := "idem-fixed-" + uuid.NewString()
	body := map[string]any{"purpose": "Need supplies for team", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}}}
	rec1 := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, idemHeaders(key))
	rec2 := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, idemHeaders(key))
	requireStatus(t, rec1, http.StatusCreated)
	requireStatus(t, rec2, http.StatusCreated)
	if !jsonEqual(t, rec1.Body.Bytes(), rec2.Body.Bytes()) {
		t.Fatalf("expected replayed body equal\n1=%s\n2=%s", rec1.Body.String(), rec2.Body.String())
	}
	body["items"] = []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 2}}
	rec3 := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, idemHeaders(key))
	requireStatus(t, rec3, http.StatusConflict)
	requireErrorCode(t, rec3, "IDEMPOTENCY_KEY_REUSED")
}

func TestIdempotencyKeyCannotBeReplayedByAnotherUser(t *testing.T) {
	env := testkit.New(t)
	// create another department user in IT by inserting directly
	var count int
	if err := env.DB.Raw(`SELECT COUNT(*) FROM users WHERE email = 'dept2@example.com'`).Scan(&count).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	hash := env.DB.Exec(`INSERT INTO users (id,email,password_hash,full_name,role,department_id) VALUES (gen_random_uuid(),'dept2@example.com',(SELECT password_hash FROM users WHERE email='dept@example.com'),'Dept Two','DEPARTMENT_USER',?)`, env.ITDeptID)
	if hash.Error != nil {
		t.Fatalf("insert user: %v", hash.Error)
	}
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	dept2 := loginToken(t, env, "dept2@example.com", testkit.Password)
	key := "idem-cross-" + uuid.NewString()
	body := map[string]any{"purpose": "Need supplies for team", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}}}
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, idemHeaders(key)), http.StatusCreated)
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept2, idemHeaders(key))
	requireStatus(t, rec, http.StatusConflict)
	requireErrorCode(t, rec, "IDEMPOTENCY_KEY_REUSED")
}

func TestIdempotencyKeyValidation(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	body := map[string]any{"purpose": "Need supplies for team", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}}}
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, map[string]string{"Idempotency-Key": "short"})
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	rec = env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, map[string]string{"Idempotency-Key": "bad key with space"})
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestRequestIDSanitized(t *testing.T) {
	env := testkit.New(t)
	rec := env.Request(http.MethodGet, "/healthz", nil, "", map[string]string{"X-Request-ID": "<script>"})
	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("X-Request-Id"); got == "<script>" {
		t.Fatalf("unsafe request id echoed: %q", got)
	}
}

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("unmarshal a: %v", err)
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		t.Fatalf("unmarshal b: %v", err)
	}
	return reflect.DeepEqual(av, bv)
}
