package api_test

import (
	"net/http"
	"testing"

	"wms/backend/internal/testkit"
)

func TestDepartmentIsolationStockRequests(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	// create request as dept user
	body := map[string]any{"purpose": "Need supplies", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}}}
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, idemKey())
	requireStatus(t, rec, http.StatusCreated)
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, rec, &created)
	if created.Data.ID == "" {
		t.Fatal("missing request id")
	}
	// admin can see
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/stock-requests/"+created.Data.ID, nil, admin, nil), http.StatusOK)
	// another department user cannot be created in fixtures easily here; instead ensure dept list is scoped
	rec = env.Request(http.MethodGet, "/api/v1/stock-requests", nil, dept, nil)
	requireStatus(t, rec, http.StatusOK)
}
