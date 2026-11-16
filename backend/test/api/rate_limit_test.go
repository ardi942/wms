package api_test

import (
	"net/http"
	"testing"
	"time"

	"wms/backend/internal/config"
	"wms/backend/internal/ratelimit"
	"wms/backend/internal/testkit"
)

func TestRateLimitLogin(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	limiter := ratelimit.NewMemoryWithClock(func() time.Time { return now })
	env := testkit.New(t,
		testkit.WithLimiter(limiter),
		testkit.WithConfig(func(c *config.Config) {
			c.RateLimitEnabled = true
			c.RateLimitLoginPerMinute = 2
			c.RateLimitAPIPerMinute = 100
			c.RateLimitMutationPerMin = 100
		}),
	)
	body := map[string]string{"email": "admin@example.com", "password": "wrong"}
	for i := 0; i < 2; i++ {
		requireStatus(t, env.Request(http.MethodPost, "/api/v1/auth/login", body, "", nil), http.StatusUnauthorized)
	}
	rec := env.Request(http.MethodPost, "/api/v1/auth/login", body, "", nil)
	requireStatus(t, rec, http.StatusTooManyRequests)
	if rec.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header")
	}
}
