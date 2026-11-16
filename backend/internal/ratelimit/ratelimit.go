package ratelimit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "wms:ratelimit:"

type Decision struct {
	Allowed   bool
	Limit     int
	Remaining int
	Reset     time.Time
}

type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error)
	Close() error
}

type bucket struct {
	count int
	reset time.Time
}

type Memory struct {
	mu      sync.Mutex
	buckets map[string]bucket
	now     func() time.Time
}

func NewMemory() *Memory {
	return NewMemoryWithClock(time.Now)
}

func NewMemoryWithClock(now func() time.Time) *Memory {
	return &Memory{buckets: map[string]bucket{}, now: now}
}

func (m *Memory) Allow(_ context.Context, key string, limit int, window time.Duration) (Decision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if len(m.buckets) > 10000 {
		for k, b := range m.buckets {
			if !now.Before(b.reset) {
				delete(m.buckets, k)
			}
		}
	}
	b, ok := m.buckets[key]
	if !ok || !now.Before(b.reset) {
		b = bucket{reset: now.Add(window)}
	}
	b.count++
	m.buckets[key] = b
	remaining := limit - b.count
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:   b.count <= limit,
		Limit:     limit,
		Remaining: remaining,
		Reset:     b.reset,
	}, nil
}

func (m *Memory) Close() error { return nil }

type Redis struct {
	client *redis.Client
}

func NewRedis(addr, password string) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &Redis{client: client}, nil
}

var redisScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {current, ttl}
`)

func (r *Redis) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	values, err := redisScript.Run(ctx, r.client, []string{redisKeyPrefix + key}, window.Milliseconds()).Int64Slice()
	if err != nil {
		return Decision{}, err
	}
	current := int(values[0])
	ttl := time.Duration(values[1]) * time.Millisecond
	if ttl <= 0 {
		ttl = window
	}
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:   current <= limit,
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Now().Add(ttl),
	}, nil
}

func (r *Redis) Close() error { return r.client.Close() }

type Fallback struct {
	Primary Limiter
	Backup  Limiter
	Log     *slog.Logger
}

func (f *Fallback) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	decision, err := f.Primary.Allow(ctx, key, limit, window)
	if err == nil {
		return decision, nil
	}
	if f.Log != nil {
		f.Log.Warn("rate limiter unavailable; using in-memory fallback", "error", err)
	}
	return f.Backup.Allow(ctx, key, limit, window)
}

func (f *Fallback) Close() error {
	err := f.Primary.Close()
	if berr := f.Backup.Close(); err == nil {
		err = berr
	}
	return err
}
