package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	KeyProductsVersion  = "wms:v1:products:ver"
	KeyDashboardVersion = "wms:v1:dashboard:ver"
)

type Cache struct {
	rdb *redis.Client
	log *slog.Logger
}

func New(addr, password string, log *slog.Logger) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("redis unavailable at startup; operating with database fallback", "error", err)
	}
	return &Cache{rdb: rdb, log: log}
}

func (c *Cache) Close() error { return c.rdb.Close() }

func (c *Cache) Ping(ctx context.Context) error { return c.rdb.Ping(ctx).Err() }

func (c *Cache) Version(ctx context.Context, key string) int64 {
	v, err := c.rdb.Get(ctx, key).Int64()
	if err != nil {
		return 0
	}
	return v
}

func (c *Cache) Bump(ctx context.Context, key string) {
	if err := c.rdb.Incr(ctx, key).Err(); err != nil {
		c.log.Debug("redis bump failed", "key", key, "error", err)
	}
}

func (c *Cache) GetJSON(ctx context.Context, key string, dst any) bool {
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, dst) == nil
}

func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	if err := c.rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		c.log.Debug("redis set failed", "key", key, "error", err)
	}
}

func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		c.log.Debug("redis delete failed", "error", err)
	}
}

func (c *Cache) InvalidateStock(ctx context.Context) {
	c.Bump(ctx, KeyProductsVersion)
	c.Bump(ctx, KeyDashboardVersion)
}

func (c *Cache) InvalidateProduct(ctx context.Context) {
	c.Bump(ctx, KeyProductsVersion)
	c.Bump(ctx, KeyDashboardVersion)
}

func (c *Cache) InvalidateDashboard(ctx context.Context) {
	c.Bump(ctx, KeyDashboardVersion)
}

func VersionString(v int64) string { return strconv.FormatInt(v, 10) }
