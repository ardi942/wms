package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	JWTTTL             time.Duration
	RedisAddr          string
	RedisPassword      string
	RabbitMQURL        string
	CORSOrigins        []string
	LogLevel           string
	WorkerConcurrency  int
	OutboxPollInterval time.Duration
	MigrationsPath     string
	SeedDemo           bool
}

func Load() (*Config, error) {
	c := &Config{
		Port:               env("API_PORT", "8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres://wms:wms@localhost:5432/wms?sslmode=disable"),
		JWTSecret:          env("JWT_SECRET", ""),
		RedisAddr:          env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      env("REDIS_PASSWORD", ""),
		RabbitMQURL:        env("RABBITMQ_URL", "amqp://wms:wms@localhost:5672/"),
		LogLevel:           env("LOG_LEVEL", "info"),
		MigrationsPath:     env("MIGRATIONS_PATH", "./migrations"),
		WorkerConcurrency:  envInt("WORKER_CONCURRENCY", 8),
		OutboxPollInterval: envDuration("OUTBOX_POLL_INTERVAL", time.Second),
		JWTTTL:             envDuration("JWT_TTL", 12*time.Hour),
		CORSOrigins:        splitCSV(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		SeedDemo:           envBool("SEED_DEMO", true),
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.WorkerConcurrency < 1 || c.WorkerConcurrency > 64 {
		return nil, fmt.Errorf("WORKER_CONCURRENCY must be between 1 and 64")
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
