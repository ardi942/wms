package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiter(t *testing.T) {
	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	l := NewMemoryWithClock(func() time.Time { return now })
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		d, err := l.Allow(ctx, "k", 3, time.Minute)
		if err != nil || !d.Allowed {
			t.Fatalf("request %d should be allowed: %#v err=%v", i+1, d, err)
		}
	}
	d, err := l.Allow(ctx, "k", 3, time.Minute)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d.Allowed || d.Remaining != 0 {
		t.Fatalf("expected blocked with zero remaining: %#v", d)
	}
	now = now.Add(time.Minute)
	d, err = l.Allow(ctx, "k", 3, time.Minute)
	if err != nil || !d.Allowed {
		t.Fatalf("expected reset allowed: %#v err=%v", d, err)
	}
}
