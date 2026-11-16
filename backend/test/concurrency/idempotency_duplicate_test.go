package concurrency_test

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

func TestConcurrentSameIdempotencyKeySingleSideEffect(t *testing.T) {
	env := testkit.New(t)
	dept := login(t, env, "dept@example.com")
	key := "idem-concurrent-" + uuid.NewString()
	body := map[string]any{
		"purpose": "Need supplies for team",
		"items":   []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 1}},
	}

	const n = 8
	type result struct {
		code int
		body []byte
	}
	results := make(chan result, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := env.Request(http.MethodPost, "/api/v1/stock-requests", body, dept, map[string]string{"Idempotency-Key": key})
			results <- result{code: rec.Code, body: append([]byte(nil), rec.Body.Bytes()...)}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	created, conflict := 0, 0
	var createdID string
	for r := range results {
		switch r.code {
		case http.StatusCreated, http.StatusOK:
			created++
			var out struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			_ = json.Unmarshal(r.body, &out)
			if createdID == "" {
				createdID = out.Data.ID
			} else if out.Data.ID != "" && createdID != out.Data.ID {
				t.Fatalf("different resource ids for same idempotency key: %s vs %s", createdID, out.Data.ID)
			}
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected status %d body=%s", r.code, string(r.body))
		}
	}
	if created < 1 || created+conflict != n {
		t.Fatalf("expected >=1 success and rest conflicts, got created=%d conflict=%d", created, conflict)
	}
	var count int64
	if err := env.DB.Table("stock_requests").Count(&count).Error; err != nil {
		t.Fatalf("count requests: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 stock request row, got %d", count)
	}
}

func TestDuplicateFulfillSameRequestSingleMovement(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	dept := login(t, env, "dept@example.com")
	env.SetInventory(env.ProductID, 10)

	rec := env.Request(http.MethodPost, "/api/v1/stock-requests", map[string]any{
		"purpose": "Need supplies", "items": []map[string]any{{"product_id": env.ProductID.String(), "requested_qty": 5}},
	}, dept, idem())
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	body := map[string]any{"items": []map[string]any{{"item_id": created.Data.Items[0].ID, "issued_quantity": 5}}}
	const n = 6
	codes := make(chan int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			rec := env.Request(http.MethodPost, "/api/v1/stock-requests/"+created.Data.ID+"/fulfill", body, admin, map[string]string{"Idempotency-Key": "fulfill-" + uuid.NewString()})
			codes <- rec.Code
		}(i)
	}
	close(start)
	wg.Wait()
	close(codes)

	ok := 0
	for c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict, http.StatusUnprocessableEntity:
		default:
			t.Fatalf("unexpected fulfill status %d", c)
		}
	}
	if ok != 1 {
		t.Fatalf("expected exactly 1 successful fulfill, got %d", ok)
	}
	if bal := env.Balance(env.ProductID); bal != 5 {
		t.Fatalf("expected balance 5, got %d", bal)
	}
	var moves int64
	if err := env.DB.Raw(`SELECT COUNT(*) FROM stock_movements WHERE reference_type = 'REQUEST' AND reference_id = ?`, created.Data.ID).Scan(&moves).Error; err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if moves != 1 {
		t.Fatalf("expected exactly 1 movement, got %d", moves)
	}
}
