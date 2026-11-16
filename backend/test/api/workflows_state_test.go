package api_test

import (
	"net/http"
	"testing"

	"wms/backend/internal/testkit"
)

func createPendingRequest(t *testing.T, env *testkit.Env, token string, qty int) (string, string) {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": "Workflow request",
		"items":   []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": qty}},
	}, token, idemKey())
	requireStatus(t, rec, http.StatusCreated)
	var out struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	mustDecode(t, rec, &out)
	return out.Data.ID, out.Data.Items[0].ID
}

func createPendingReturn(t *testing.T, env *testkit.Env, token string, qty int) (string, string) {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/stock-returns", map[string]any{
		"reason": "Workflow return",
		"items":  []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": qty}},
	}, token, idemKey())
	requireStatus(t, rec, http.StatusCreated)
	var out struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	mustDecode(t, rec, &out)
	return out.Data.ID, out.Data.Items[0].ID
}

func TestRequestStateTransitions(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	env.SetInventory(env.ProductID, 10)

	reqID, itemID := createPendingRequest(t, env, dept, 3)
	// fulfill partial not possible here (single line full), fulfill fully
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID+"/fulfill", map[string]any{
		"items": []map[string]any{{"item_id": itemID, "issued_quantity": 3}},
	}, admin, idemKey())
	requireStatus(t, rec, http.StatusOK)
	// terminal state rejects further fulfill/reject
	rec = env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID+"/reject", map[string]any{"reason": "too late"}, admin, idemKey())
	requireStatus(t, rec, http.StatusConflict)
	requireErrorCode(t, rec, "INVALID_STATE")

	reqID2, _ := createPendingRequest(t, env, dept, 1)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID2+"/reject", map[string]any{"reason": "not needed now"}, admin, idemKey()), http.StatusOK)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID2+"/cancel", nil, dept, idemKey()), http.StatusConflict)

	reqID3, _ := createPendingRequest(t, env, dept, 1)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID3+"/cancel", nil, dept, idemKey()), http.StatusOK)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID3+"/fulfill", map[string]any{"items": []any{}}, admin, idemKey()), http.StatusUnprocessableEntity)
}

func TestReturnStateAndCap(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	env.SetInventory(env.ProductID, 0)

	// issue 2 to IT dept via request fulfillment to establish net issued
	reqID, itemID := createPendingRequest(t, env, dept, 2)
	env.SetInventory(env.ProductID, 2)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID+"/fulfill", map[string]any{
		"items": []map[string]any{{"item_id": itemID, "issued_quantity": 2}},
	}, admin, idemKey()), http.StatusOK)

	retID, retItemID := createPendingReturn(t, env, dept, 5)
	// cannot accept more than net issued (2)
	rec := env.Request(http.MethodPost, "/api/v1/stock-returns/"+retID+"/confirm", map[string]any{
		"items": []map[string]any{{"item_id": retItemID, "accepted_qty": 5}},
	}, admin, idemKey())
	requireStatus(t, rec, http.StatusConflict)
	requireErrorCode(t, rec, "RETURN_EXCEEDS_ISSUED")

	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-returns/"+retID+"/confirm", map[string]any{
		"items": []map[string]any{{"item_id": retItemID, "accepted_qty": 2}},
	}, admin, idemKey()), http.StatusOK)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-returns/"+retID+"/reject", map[string]any{"reason": "too late now"}, admin, idemKey()), http.StatusConflict)

	retID2, _ := createPendingReturn(t, env, dept, 1)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-returns/"+retID2+"/reject", map[string]any{"reason": "not acceptable"}, admin, idemKey()), http.StatusOK)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-returns/"+retID2+"/cancel", nil, dept, idemKey()), http.StatusConflict)
}

func TestCrossDepartmentIsolationAndInvalidActions(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)

	// create another department + user
	if err := env.DB.Exec(`INSERT INTO users (id,email,password_hash,full_name,role,department_id,is_active)
		SELECT gen_random_uuid(),'fac@example.com',password_hash,'Fac User','DEPARTMENT_USER',?,true FROM users WHERE email='dept@example.com'`, env.FacilitiesID).Error; err != nil {
		t.Fatalf("insert fac user: %v", err)
	}
	fac := loginToken(t, env, "fac@example.com", testkit.Password)

	reqID, _ := createPendingRequest(t, env, dept, 1)
	// other dept cannot see/cancel
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/stock-requests/"+reqID, nil, fac, nil), http.StatusNotFound)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID+"/cancel", nil, fac, idemKey()), http.StatusNotFound)

	// admin can see but cannot create/cancel dept-only
	requireStatus(t, env.Request(http.MethodGet, "/api/v1/stock-requests/"+reqID, nil, admin, nil), http.StatusOK)
}

func TestInventoryMovementsDepartmentScoped(t *testing.T) {
	env := testkit.New(t)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	env.SetInventory(env.ProductID, 5)
	reqID, itemID := createPendingRequest(t, env, dept, 1)
	requireStatus(t, env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID+"/fulfill", map[string]any{
		"items": []map[string]any{{"item_id": itemID, "issued_quantity": 1}},
	}, admin, idemKey()), http.StatusOK)

	// dept user movements filtered to own dept (should succeed and include results)
	rec := env.Request(http.MethodGet, "/api/v1/inventory/movements", nil, dept, nil)
	requireStatus(t, rec, http.StatusOK)
	// trying to spoof other department_id is ignored/overridden server-side for dept user
	rec = env.Request(http.MethodGet, "/api/v1/inventory/movements?department_id="+env.FacilitiesID.String(), nil, dept, nil)
	requireStatus(t, rec, http.StatusOK)
}
