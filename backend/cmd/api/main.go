package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"wms/backend/internal/app"
	"wms/backend/internal/cache"
	"wms/backend/internal/config"
	"wms/backend/internal/database"
	"wms/backend/internal/idempotency"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	log := newLogger(cfg.LogLevel)
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	var warehouseID uuid.UUID
	if err := db.Raw(`SELECT id FROM warehouses WHERE code = 'MAIN'`).Row().Scan(&warehouseID); err != nil || warehouseID == uuid.Nil {
		log.Error("warehouse MAIN is missing; run migrations and seed first")
		os.Exit(1)
	}
	redisCache := cache.New(cfg.RedisAddr, cfg.RedisPassword, log)
	defer redisCache.Close()

	application := app.New(app.Deps{
		DB:          db,
		Cache:       redisCache,
		Config:      cfg,
		Logger:      log,
		WarehouseID: warehouseID,
	})
	defer application.Close()
	go runIdempotencyCleanup(application.Idempotency, log)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("api listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down api")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Error("api shutdown error", "error", err)
	}
}

func runIdempotencyCleanup(store *idempotency.Store, log *slog.Logger) {
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := store.Cleanup(ctx); err != nil {
			log.Warn("idempotency cleanup failed", "error", err)
		}
	}
	cleanup()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		cleanup()
	}
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
