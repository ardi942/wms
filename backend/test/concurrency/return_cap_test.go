package concurrency_test

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"wms/backend/internal/testkit"
)

func TestConcurrentReturnsRespectNetIssuedCap(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	dept := login(t, env, "dept@example.com")

	// establish net issued = 2
	env.SetInventory(env.ProductID, 2)
	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": "Need", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 2}},
	}, dept, idem())
	if rec.Code != http.StatusCreated {
		t.Fatalf("create req: %d %s", rec.Code, rec.Body.String())
	}
	var req struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &req)
	rec = env.Request(http.MethodPost, "/api/v1/stock-requests/"+req.Data.ID+"/fulfill", map[string]any{
		"items": []map[string]any{{"item_id": req.Data.Items[0].ID, "issued_quantity": 2}},
	}, admin, idem())
	if rec.Code != http.StatusOK {
		t.Fatalf("fulfill: %d %s", rec.Code, rec.Body.String())
	}
	if bal := env.Balance(env.ProductID); bal != 0 {
		t.Fatalf("expected balance 0 after issue, got %d", bal)
	}

	createRet := func() (string, string) {
		rec := env.Request(http.MethodPost, "/api/v1/stock-returns", map[string]any{
			"reason": "Return reason", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 2}},
		}, dept, idem())
		if rec.Code != http.StatusCreated {
			t.Fatalf("create return: %d %s", rec.Code, rec.Body.String())
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
	ret1, item1 := createRet()
	ret2, item2 := createRet()

	results := make(chan int, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	confirm := func(retID, itemID string) {
		defer wg.Done()
		<-start
		rec := env.Request(http.MethodPost, "/api/v1/stock-returns/"+retID+"/confirm", map[string]any{
			"items": []map[string]any{{"item_id": itemID, "accepted_qty": 2}},
		}, admin, idem())
		results <- rec.Code
	}
	wg.Add(2)
	go confirm(ret1, item1)
	go confirm(ret2, item2)
	close(start)
	wg.Wait()
	close(results)

	ok, conflict := 0, 0
	for c := range results {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict, http.StatusUnprocessableEntity:
			conflict++
		default:
			t.Fatalf("unexpected status %d", c)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("expected 1 ok 1 conflict, got ok=%d conflict=%d", ok, conflict)
	}
	if bal := env.Balance(env.ProductID); bal != 2 {
		t.Fatalf("expected balance 2 after one accepted return, got %d", bal)
	}
}
