package concurrency_test

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testkit.Cleanup()
	os.Exit(code)
}

func login(t *testing.T, env *testkit.Env, email string) string {
	t.Helper()
	rec := env.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": testkit.Password}, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return out.Data.Token
}

func idem() map[string]string {
	return map[string]string{"Idempotency-Key": "idem-" + uuid.NewString()}
}

func TestConcurrentReceiptsInvariant(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	env.SetInventory(env.ProductID, 0)
	const n = 10
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := env.Request(http.MethodPost, "/api/v1/inventory/receipts", map[string]any{
				"product_id": env.ProductID.String(), "quantity": 1,
				"supplier": "Test Supplier", "reference_number": "RCPT-" + uuid.NewString(),
				"notes": "concurrent",
			}, admin, idem())
			if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
				errs <- rec.Body.String()
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("receipt failed: %s", e)
	}
	if bal := env.Balance(env.ProductID); bal != n {
		t.Fatalf("expected balance %d got %d", n, bal)
	}
}

func TestConcurrentDuplicateEmail(t *testing.T) {
	env := testkit.New(t)
	admin := login(t, env, "admin@example.com")
	const n = 6
	email := "race-dup@example.com"
	body := map[string]any{
		"email": email, "full_name": "Race User", "password": "Password123!",
		"department_id": env.ITDeptID.String(),
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := env.Request(http.MethodPost, "/api/v1/users", body, admin, nil)
			codes <- rec.Code
		}()
	}
	close(start)
	wg.Wait()
	close(codes)
	created, conflict := 0, 0
	for c := range codes {
		switch c {
		case http.StatusCreated:
			created++
		case http.StatusConflict, http.StatusUnprocessableEntity:
			conflict++
		default:
			t.Fatalf("unexpected status %d", c)
		}
	}
	if created != 1 {
		t.Fatalf("expected exactly 1 create, got created=%d conflict=%d", created, conflict)
	}
}

func TestAuditLogsAreImmutable(t *testing.T) {
	env := testkit.New(t)
	// trigger an audit row via user activation path if exists; fallback insert direct then try update/delete
	env.DB.Exec(`INSERT INTO audit_logs (id, entity_type, entity_id, action, actor_id, actor_name, actor_email, actor_role, changed_fields)
		VALUES (gen_random_uuid(), 'USER', gen_random_uuid(), 'USER_PROFILE_UPDATED', ?, 'A','a@example.com','SUPERVISOR', '{}')`, env.Users["admin"].ID)
	if err := env.DB.Exec(`UPDATE audit_logs SET action = 'USER_CREATED'`).Error; err == nil {
		t.Fatal("expected audit_logs update to fail")
	}
	if err := env.DB.Exec(`DELETE FROM audit_logs`).Error; err == nil {
		t.Fatal("expected audit_logs delete to fail")
	}
}
