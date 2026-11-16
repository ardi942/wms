package concurrency_test

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

func TestOversellInvariant(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	dept := login(t, env, "dept@example.com")
	env.SetInventory(env.ProductID, 10)

	// two requests of 7 each, try fulfill both concurrently for full amount -> only one should fully succeed without exceeding stock
	createReq := func(qty int) (string, string) {
		rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
			"purpose": "Need", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": qty}},
		}, dept, map[string]string{"Idempotency-Key": "idem-" + uuid.NewString()})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create request: %d %s", rec.Code, rec.Body.String())
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
		return out.Data.ID, out.Data.Items[0].ID
	}
	id1, item1 := createReq(7)
	id2, item2 := createReq(7)

	type result struct{ code int }
	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	fulfill := func(reqID, itemID string) {
		defer wg.Done()
		<-start
		rec := env.Request(http.MethodPost, "/api/v1/stock-requests/"+reqID+"/fulfill", map[string]any{
			"items": []map[string]any{{"item_id": itemID, "issued_quantity": 7}},
		}, admin, map[string]string{"Idempotency-Key": "idem-" + uuid.NewString()})
		results <- result{code: rec.Code}
	}
	wg.Add(2)
	go fulfill(id1, item1)
	go fulfill(id2, item2)
	close(start)
	wg.Wait()
	close(results)
	ok, conflict := 0, 0
	for r := range results {
		switch r.code {
		case http.StatusOK:
			ok++
		case http.StatusConflict, http.StatusUnprocessableEntity:
			conflict++
		default:
			t.Fatalf("unexpected status %d", r.code)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("expected 1 ok and 1 conflict, got ok=%d conflict=%d", ok, conflict)
	}
	if bal := env.Balance(env.ProductID); bal != 3 {
		t.Fatalf("expected final balance 3 got %d", bal)
	}
}

func TestStockMovementsAreImmutable(t *testing.T) {
	env := testkit.New(t)
	// insert minimal movement row
	env.DB.Exec(`INSERT INTO users (id,email,password_hash,full_name,role) VALUES (gen_random_uuid(),'tmp-sm@example.com',(SELECT password_hash FROM users WHERE email='admin@example.com'),'Tmp','SUPERVISOR') ON CONFLICT DO NOTHING`)
	env.DB.Exec(`INSERT INTO stock_movements (id, product_id, warehouse_id, movement_type, quantity_delta, balance_after, reference_type, reference_id, reference_number, actor_id)
		VALUES (gen_random_uuid(), ?, ?, 'IN', 1, 1, 'RECEIPT', gen_random_uuid(), 'R', (SELECT id FROM users WHERE email='admin@example.com'))`, env.ProductID, env.WarehouseID)
	if err := env.DB.Exec(`UPDATE stock_movements SET quantity_delta = 2`).Error; err == nil {
		t.Fatal("expected stock_movements update to fail")
	}
	if err := env.DB.Exec(`DELETE FROM stock_movements`).Error; err == nil {
		t.Fatal("expected stock_movements delete to fail")
	}
}
