package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

type matrixCase struct {
	name      string
	method    string
	path      func(*testkit.Env, ids) string
	token     func(tokens) string
	body      func(*testkit.Env, ids) any
	headers   func() map[string]string
	want      int
	wantCodes []int
}

type tokens struct{ dept, supervisor, admin, inactive string }
type ids struct {
	productID, product2ID, userID, deptUserID, inactiveUserID uuid.UUID
	deptID, catID, unitID                                     uuid.UUID
	requestID, requestItemID                                  uuid.UUID
	returnID, returnItemID                                    uuid.UUID
	notifID                                                   uuid.UUID
}

func ok(code int, wants []int) bool {
	if len(wants) > 0 {
		for _, w := range wants {
			if code == w {
				return true
			}
		}
		return false
	}
	return code == 0
}

func TestRouteMatrixComprehensive(t *testing.T) {
	env := testkit.New(t)
	tok := tokens{
		dept:       loginToken(t, env, "dept@example.com", testkit.Password),
		supervisor: loginToken(t, env, "supervisor@example.com", testkit.Password),
		admin:      loginToken(t, env, "admin@example.com", testkit.Password),
	}
	id := ids{
		productID: env.ProductID, product2ID: env.Product2ID,
		userID: env.Users["admin"].ID, deptUserID: env.Users["dept"].ID, inactiveUserID: env.Users["inactive"].ID,
		deptID: env.ITDeptID, catID: env.CategoryID, unitID: env.UnitID,
	}
	// create request/return/notification fixtures
	id.requestID, id.requestItemID = createRequestFixture(t, env, tok.dept)
	id.returnID, id.returnItemID = createReturnFixture(t, env, tok.dept)
	id.notifID = createNotificationFixture(t, env, env.Users["dept"].ID)

	static := func(p string) func(*testkit.Env, ids) string { return func(*testkit.Env, ids) string { return p } }
	withID := func(fmtStr string, pick func(ids) uuid.UUID) func(*testkit.Env, ids) string {
		return func(_ *testkit.Env, x ids) string { return fmt.Sprintf(fmtStr, pick(x).String()) }
	}
	noBody := func(*testkit.Env, ids) any { return nil }
	noHdr := func() map[string]string { return nil }
	deptTok := func(x tokens) string { return x.dept }
	supervisorTok := func(x tokens) string { return x.supervisor }
	adminTok := func(x tokens) string { return x.admin }
	anon := func(tokens) string { return "" }

	mk := func(name, method string, path func(*testkit.Env, ids) string, token func(tokens) string, body func(*testkit.Env, ids) any, headers func() map[string]string, want int, codes ...int) matrixCase {
		return matrixCase{name: name, method: method, path: path, token: token, body: body, headers: headers, want: want, wantCodes: codes}
	}
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "mx-" + uuid.NewString()} }
	hdr := func() func() map[string]string { return func() map[string]string { return idem() } }

	cases := []matrixCase{
		// health/ready
		mk("healthz ok", "GET", static("/healthz"), anon, noBody, noHdr, 200),
		mk("readyz ok", "GET", static("/readyz"), anon, noBody, noHdr, 200),
		// auth
		mk("login invalid", "POST", static("/api/v1/auth/login"), anon, func(*testkit.Env, ids) any { return map[string]any{"email": "dept@example.com", "password": "wrong"} }, noHdr, 401),
		mk("login validation", "POST", static("/api/v1/auth/login"), anon, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 422),
		mk("login success", "POST", static("/api/v1/auth/login"), anon, func(*testkit.Env, ids) any {
			return map[string]any{"email": "dept@example.com", "password": testkit.Password}
		}, noHdr, 200),
		mk("me unauth", "GET", static("/api/v1/auth/me"), anon, noBody, noHdr, 401),
		mk("me ok", "GET", static("/api/v1/auth/me"), deptTok, noBody, noHdr, 200),
		// common reads
		mk("departments unauth", "GET", static("/api/v1/departments"), anon, noBody, noHdr, 401),
		mk("departments ok", "GET", static("/api/v1/departments"), deptTok, noBody, noHdr, 200),
		mk("categories unauth", "GET", static("/api/v1/categories"), anon, noBody, noHdr, 401),
		mk("categories ok", "GET", static("/api/v1/categories"), deptTok, noBody, noHdr, 200),
		mk("units unauth", "GET", static("/api/v1/units"), anon, noBody, noHdr, 401),
		mk("units ok", "GET", static("/api/v1/units"), deptTok, noBody, noHdr, 200),
		mk("products unauth", "GET", static("/api/v1/products"), anon, noBody, noHdr, 401),
		mk("products ok", "GET", static("/api/v1/products"), deptTok, noBody, noHdr, 200),
		mk("products invalid query", "GET", static("/api/v1/products?page=0"), deptTok, noBody, noHdr, 422),
		mk("product get unauth", "GET", withID("/api/v1/products/%s", func(x ids) uuid.UUID { return x.productID }), anon, noBody, noHdr, 401),
		mk("product get invalid id", "GET", static("/api/v1/products/not-a-uuid"), deptTok, noBody, noHdr, 404),
		mk("product get ok", "GET", withID("/api/v1/products/%s", func(x ids) uuid.UUID { return x.productID }), deptTok, noBody, noHdr, 200),
		// inventory
		mk("inventory unauth", "GET", static("/api/v1/inventory"), anon, noBody, noHdr, 401),
		mk("inventory ok", "GET", static("/api/v1/inventory"), deptTok, noBody, noHdr, 200),
		mk("inventory invalid", "GET", static("/api/v1/inventory?page=-1"), deptTok, noBody, noHdr, 422),
		mk("inventory by product ok", "GET", withID("/api/v1/inventory/%s", func(x ids) uuid.UUID { return x.productID }), deptTok, noBody, noHdr, 200),
		mk("inventory by product bad id", "GET", static("/api/v1/inventory/nope"), deptTok, noBody, noHdr, 404),
		mk("movements unauth", "GET", static("/api/v1/inventory/movements"), anon, noBody, noHdr, 401),
		mk("movements ok", "GET", static("/api/v1/inventory/movements"), deptTok, noBody, noHdr, 200),
		mk("movements invalid date", "GET", static("/api/v1/inventory/movements?date_from=2026-02-31"), deptTok, noBody, noHdr, 422),
		// requests/returns lists & gets
		mk("requests unauth", "GET", static("/api/v1/stock-requests"), anon, noBody, noHdr, 401),
		mk("requests ok", "GET", static("/api/v1/stock-requests"), deptTok, noBody, noHdr, 200),
		mk("requests invalid date", "GET", static("/api/v1/stock-requests?date_from=bad"), deptTok, noBody, noHdr, 422),
		mk("request get unauth", "GET", withID("/api/v1/stock-requests/%s", func(x ids) uuid.UUID { return x.requestID }), anon, noBody, noHdr, 401),
		mk("request get bad id", "GET", static("/api/v1/stock-requests/nope"), deptTok, noBody, noHdr, 404),
		mk("request get ok", "GET", withID("/api/v1/stock-requests/%s", func(x ids) uuid.UUID { return x.requestID }), deptTok, noBody, noHdr, 200),
		mk("returns unauth", "GET", static("/api/v1/stock-returns"), anon, noBody, noHdr, 401),
		mk("returns ok", "GET", static("/api/v1/stock-returns"), deptTok, noBody, noHdr, 200),
		mk("return get bad id", "GET", static("/api/v1/stock-returns/nope"), deptTok, noBody, noHdr, 404),
		mk("return get ok", "GET", withID("/api/v1/stock-returns/%s", func(x ids) uuid.UUID { return x.returnID }), deptTok, noBody, noHdr, 200),
		// monitoring
		mk("monitor req unauth", "GET", static("/api/v1/monitoring/requests"), anon, noBody, noHdr, 401),
		mk("monitor req ok", "GET", static("/api/v1/monitoring/requests"), deptTok, noBody, noHdr, 200),
		mk("monitor ret ok", "GET", static("/api/v1/monitoring/returns"), adminTok, noBody, noHdr, 200),
		mk("monitor req invalid", "GET", static("/api/v1/monitoring/requests?department_id=bad"), deptTok, noBody, noHdr, 422),
		// reports/dashboard/notifications
		mk("dashboard unauth", "GET", static("/api/v1/reports/dashboard"), anon, noBody, noHdr, 401),
		mk("dashboard ok", "GET", static("/api/v1/reports/dashboard"), deptTok, noBody, noHdr, 200),
		mk("recent movements ok", "GET", static("/api/v1/reports/dashboard/recent-movements"), deptTok, noBody, noHdr, 200),
		mk("dept usage ok", "GET", static("/api/v1/reports/department-usage"), deptTok, noBody, noHdr, 200),
		mk("export unauth", "GET", static("/api/v1/reports/export?report=inventory"), anon, noBody, noHdr, 401),
		mk("export invalid type", "GET", static("/api/v1/reports/export?report=nope"), adminTok, noBody, noHdr, 422),
		mk("export ok", "GET", static("/api/v1/reports/export?report=inventory"), adminTok, noBody, noHdr, 200),
		mk("notifications unauth", "GET", static("/api/v1/notifications"), anon, noBody, noHdr, 401),
		mk("notifications ok", "GET", static("/api/v1/notifications"), deptTok, noBody, noHdr, 200),
		mk("unread count ok", "GET", static("/api/v1/notifications/unread-count"), deptTok, noBody, noHdr, 200),
		mk("notif mark read unauth", "PATCH", withID("/api/v1/notifications/%s/read", func(x ids) uuid.UUID { return x.notifID }), anon, noBody, noHdr, 401),
		mk("notif mark read bad id", "PATCH", static("/api/v1/notifications/nope/read"), deptTok, noBody, noHdr, 404),
		mk("notif mark read ok", "PATCH", withID("/api/v1/notifications/%s/read", func(x ids) uuid.UUID { return x.notifID }), deptTok, noBody, noHdr, 204),
		mk("notif read all ok", "POST", static("/api/v1/notifications/read-all"), deptTok, noBody, noHdr, 204),
		// admin routes - unauth/forbidden/invalid/success
		mk("users list unauth", "GET", static("/api/v1/users"), anon, noBody, noHdr, 401),
		mk("users list forbidden dept", "GET", static("/api/v1/users"), deptTok, noBody, noHdr, 403),
		mk("users list ok", "GET", static("/api/v1/users"), adminTok, noBody, noHdr, 200),
		mk("users create forbidden", "POST", static("/api/v1/users"), deptTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 403),
		mk("users create invalid", "POST", static("/api/v1/users"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 422),
		mk("users create ok", "POST", static("/api/v1/users"), adminTok, func(_ *testkit.Env, x ids) any {
			return map[string]any{"email": "new-" + uuid.NewString() + "@example.com", "full_name": "New User", "password": "Password123!", "department_id": x.deptID.String()}
		}, noHdr, 201),
		mk("user get bad id", "GET", static("/api/v1/users/nope"), adminTok, noBody, noHdr, 404),
		mk("user get ok", "GET", withID("/api/v1/users/%s", func(x ids) uuid.UUID { return x.deptUserID }), adminTok, noBody, noHdr, 200),
		mk("user patch invalid", "PATCH", withID("/api/v1/users/%s", func(x ids) uuid.UUID { return x.deptUserID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"is_active": false} }, noHdr, 422),
		mk("user patch ok", "PATCH", withID("/api/v1/users/%s", func(x ids) uuid.UUID { return x.deptUserID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"full_name": "Dept User Updated"} }, noHdr, 200),
		mk("user activate invalid", "POST", withID("/api/v1/users/%s/activate", func(x ids) uuid.UUID { return x.deptUserID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "no"} }, noHdr, 422),
		mk("user deactivate ok", "POST", withID("/api/v1/users/%s/deactivate", func(x ids) uuid.UUID { return x.deptUserID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "deactivated for testing"} }, noHdr, 200),
		mk("user activate ok", "POST", withID("/api/v1/users/%s/activate", func(x ids) uuid.UUID { return x.deptUserID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "reactivated for testing"} }, noHdr, 200),
		// categories/units admin
		mk("category create forbidden", "POST", static("/api/v1/categories"), deptTok, func(*testkit.Env, ids) any { return map[string]any{"name": "X"} }, noHdr, 403),
		mk("category create invalid", "POST", static("/api/v1/categories"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 422),
		mk("category create ok", "POST", static("/api/v1/categories"), adminTok, func(*testkit.Env, ids) any { return map[string]any{"name": "Cat-" + uuid.NewString()[:8]} }, noHdr, 201),
		mk("category update invalid", "PATCH", withID("/api/v1/categories/%s", func(x ids) uuid.UUID { return x.catID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 422),
		mk("category update ok", "PATCH", withID("/api/v1/categories/%s", func(x ids) uuid.UUID { return x.catID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"name": "Office Supplies"} }, noHdr, 200),
		mk("unit create invalid", "POST", static("/api/v1/units"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 422),
		mk("unit create ok", "POST", static("/api/v1/units"), adminTok, func(*testkit.Env, ids) any { return map[string]any{"code": "U" + uuid.NewString()[:6], "name": "Unit"} }, noHdr, 201),
		mk("unit update ok", "PATCH", withID("/api/v1/units/%s", func(x ids) uuid.UUID { return x.unitID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"name": "Pieces"} }, noHdr, 200),
		// product admin/editor
		mk("product create forbidden", "POST", static("/api/v1/products"), deptTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 403),
		mk("product create invalid", "POST", static("/api/v1/products"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, noHdr, 422),
		mk("product create ok", "POST", static("/api/v1/products"), adminTok, func(_ *testkit.Env, x ids) any {
			return map[string]any{"sku": "SKU-" + uuid.NewString()[:8], "name": "New Product", "category_id": x.catID.String(), "unit_id": x.unitID.String(), "min_stock": 1}
		}, noHdr, 201),
		mk("product update forbidden", "PATCH", withID("/api/v1/products/%s", func(x ids) uuid.UUID { return x.productID }), deptTok, func(*testkit.Env, ids) any { return map[string]any{"name": "x"} }, noHdr, 403),
		mk("product update ok", "PATCH", withID("/api/v1/products/%s", func(x ids) uuid.UUID { return x.productID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"name": "Notebook"} }, noHdr, 200),
		// inventory mutations
		mk("receipt forbidden", "POST", static("/api/v1/inventory/receipts"), deptTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 403),
		mk("receipt invalid", "POST", static("/api/v1/inventory/receipts"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 422),
		mk("receipt ok", "POST", static("/api/v1/inventory/receipts"), adminTok, func(_ *testkit.Env, x ids) any {
			return map[string]any{"product_id": x.productID.String(), "quantity": 1, "supplier": "S", "reference_number": "R-" + uuid.NewString()}
		}, hdr(), 201),
		mk("adjustment invalid", "POST", static("/api/v1/inventory/adjustments"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 422),
		mk("adjustment ok", "POST", static("/api/v1/inventory/adjustments"), adminTok, func(_ *testkit.Env, x ids) any {
			return map[string]any{"product_id": x.productID.String(), "quantity_delta": 1, "reason": "adjustment reason"}
		}, hdr(), 201),
		// fulfill/reject request (admin)
		mk("request fulfill forbidden", "POST", withID("/api/v1/stock-requests/%s/fulfill", func(x ids) uuid.UUID { return x.requestID }), deptTok, func(*testkit.Env, ids) any { return map[string]any{"items": []any{}} }, hdr(), 403),
		mk("request fulfill invalid", "POST", withID("/api/v1/stock-requests/%s/fulfill", func(x ids) uuid.UUID { return x.requestID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"items": []any{}} }, hdr(), 422),
		mk("request reject forbidden", "POST", withID("/api/v1/stock-requests/%s/reject", func(x ids) uuid.UUID { return x.requestID }), deptTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "bad reason"} }, hdr(), 403),
		mk("request reject invalid", "POST", withID("/api/v1/stock-requests/%s/reject", func(x ids) uuid.UUID { return x.requestID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "no"} }, hdr(), 422),
		// return confirm/reject (admin)
		mk("return confirm forbidden", "POST", withID("/api/v1/stock-returns/%s/confirm", func(x ids) uuid.UUID { return x.returnID }), deptTok, func(*testkit.Env, ids) any { return map[string]any{"items": []any{}} }, hdr(), 403),
		mk("return confirm invalid", "POST", withID("/api/v1/stock-returns/%s/confirm", func(x ids) uuid.UUID { return x.returnID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"items": []any{}} }, hdr(), 422),
		mk("return reject invalid", "POST", withID("/api/v1/stock-returns/%s/reject", func(x ids) uuid.UUID { return x.returnID }), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "no"} }, hdr(), 422),
		// history/audit/reporters
		mk("audit unauth", "GET", static("/api/v1/audit"), anon, noBody, noHdr, 401),
		mk("audit forbidden", "GET", static("/api/v1/audit"), deptTok, noBody, noHdr, 403),
		mk("audit ok", "GET", static("/api/v1/audit"), supervisorTok, noBody, noHdr, 200),
		mk("audit invalid", "GET", static("/api/v1/audit?entity_type=NOPE"), supervisorTok, noBody, noHdr, 422),
		mk("user history forbidden", "GET", withID("/api/v1/users/%s/history", func(x ids) uuid.UUID { return x.deptUserID }), deptTok, noBody, noHdr, 403),
		mk("user history bad id", "GET", static("/api/v1/users/nope/history"), supervisorTok, noBody, noHdr, 404),
		mk("user history ok", "GET", withID("/api/v1/users/%s/history", func(x ids) uuid.UUID { return x.deptUserID }), supervisorTok, noBody, noHdr, 200),
		mk("product history forbidden", "GET", withID("/api/v1/products/%s/history", func(x ids) uuid.UUID { return x.productID }), deptTok, noBody, noHdr, 403),
		mk("product history ok", "GET", withID("/api/v1/products/%s/history", func(x ids) uuid.UUID { return x.productID }), supervisorTok, noBody, noHdr, 200),
		mk("report inventory forbidden", "GET", static("/api/v1/reports/inventory"), deptTok, noBody, noHdr, 403),
		mk("report inventory ok", "GET", static("/api/v1/reports/inventory"), supervisorTok, noBody, noHdr, 200),
		mk("report movements forbidden", "GET", static("/api/v1/reports/stock-movements"), deptTok, noBody, noHdr, 403),
		mk("report movements ok", "GET", static("/api/v1/reports/stock-movements"), supervisorTok, noBody, noHdr, 200),
		mk("report low stock forbidden", "GET", static("/api/v1/reports/low-stock"), deptTok, noBody, noHdr, 403),
		mk("report low stock ok", "GET", static("/api/v1/reports/low-stock"), supervisorTok, noBody, noHdr, 200),
		// dept-only create/cancel
		mk("request create unauth", "POST", static("/api/v1/stock-requests"), anon, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 401),
		mk("request create forbidden admin", "POST", static("/api/v1/stock-requests"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 403),
		mk("request create invalid", "POST", static("/api/v1/stock-requests"), deptTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 422),
		mk("request create ok", "POST", static("/api/v1/stock-requests"), deptTok, func(_ *testkit.Env, x ids) any {
			return map[string]any{"purpose": "Need supplies", "items": []map[string]any{{"product_id": x.productID.String(), "requested_qty": 1}}}
		}, hdr(), 201),
		mk("request cancel forbidden admin", "POST", withID("/api/v1/stock-requests/%s/cancel", func(x ids) uuid.UUID { return x.requestID }), adminTok, noBody, hdr(), 403),
		mk("return create forbidden admin", "POST", static("/api/v1/stock-returns"), adminTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 403),
		mk("return create invalid", "POST", static("/api/v1/stock-returns"), deptTok, func(*testkit.Env, ids) any { return map[string]any{} }, hdr(), 422),
		mk("return create ok", "POST", static("/api/v1/stock-returns"), deptTok, func(_ *testkit.Env, x ids) any {
			return map[string]any{"reason": "Return reason", "items": []map[string]any{{"product_id": x.productID.String(), "requested_qty": 1}}}
		}, hdr(), 201),
		mk("return cancel forbidden admin", "POST", withID("/api/v1/stock-returns/%s/cancel", func(x ids) uuid.UUID { return x.returnID }), adminTok, noBody, hdr(), 403),
		mk("request cancel bad id", "POST", static("/api/v1/stock-requests/nope/cancel"), deptTok, noBody, hdr(), 404),
		mk("return cancel bad id", "POST", static("/api/v1/stock-returns/nope/cancel"), deptTok, noBody, hdr(), 404),
		mk("fulfill bad id", "POST", static("/api/v1/stock-requests/nope/fulfill"), adminTok, func(*testkit.Env, ids) any { return map[string]any{"items": []any{}} }, hdr(), 404),
		mk("confirm bad id", "POST", static("/api/v1/stock-returns/nope/confirm"), adminTok, func(*testkit.Env, ids) any { return map[string]any{"items": []any{}} }, hdr(), 404),
		mk("reject request bad id", "POST", static("/api/v1/stock-requests/nope/reject"), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "reason ok"} }, hdr(), 404),
		mk("reject return bad id", "POST", static("/api/v1/stock-returns/nope/reject"), adminTok, func(*testkit.Env, ids) any { return map[string]any{"reason": "reason ok"} }, hdr(), 404),
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			path := c.path(env, id)
			token := c.token(tok)
			body := c.body(env, id)
			h := c.headers()
			rec := env.Request(c.method, path, body, token, h)
			if len(c.wantCodes) > 0 {
				if !ok(rec.Code, c.wantCodes) {
					t.Fatalf("%s %s: want one of %v got %d body=%s", c.method, path, c.wantCodes, rec.Code, rec.Body.String())
				}
				return
			}
			if rec.Code != c.want {
				t.Fatalf("%s %s: want %d got %d body=%s", c.method, path, c.want, rec.Code, rec.Body.String())
			}
		})
	}
}

func createRequestFixture(t *testing.T, env *testkit.Env, deptToken string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": "Fixture request", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}},
	}, deptToken, map[string]string{"Idempotency-Key": "fixreq-" + uuid.NewString()})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create request fixture: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return uuid.MustParse(out.Data.ID), uuid.MustParse(out.Data.Items[0].ID)
}

func createReturnFixture(t *testing.T, env *testkit.Env, deptToken string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/stock-returns", map[string]any{
		"reason": "Fixture return", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}},
	}, deptToken, map[string]string{"Idempotency-Key": "fixret-" + uuid.NewString()})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create return fixture: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return uuid.MustParse(out.Data.ID), uuid.MustParse(out.Data.Items[0].ID)
}

func createNotificationFixture(t *testing.T, env *testkit.Env, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if err := env.DB.Exec(`INSERT INTO notifications (id, user_id, type, title, body) VALUES (?, ?, 'test', 'T', 'B')`, id, userID).Error; err != nil {
		t.Fatalf("insert notification: %v", err)
	}
	return id
}
