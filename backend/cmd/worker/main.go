package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"wms/backend/internal/config"
	"wms/backend/internal/database"
	"wms/backend/internal/messaging"
	"wms/backend/internal/worker"
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
	conn, err := connectRabbit(cfg.RabbitMQURL, log)
	if err != nil {
		log.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	pub, err := messaging.NewRabbitPublisher(conn)
	if err != nil {
		log.Error("rabbitmq publisher failed", "error", err)
		os.Exit(1)
	}
	defer pub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	relay := messaging.NewRelay(db, pub, log, cfg.OutboxPollInterval)
	go relay.Run(ctx)

	notifier := worker.NewNotifier(db, log)
	consumer, err := messaging.NewConsumer(conn, messaging.QueueName, notifier.Handle, cfg.WorkerConcurrency, log)
	if err != nil {
		log.Error("consumer setup failed", "error", err)
		os.Exit(1)
	}
	go func() {
		if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("consumer stopped", "error", err)
			cancel()
		}
	}()

	log.Info("worker started", "concurrency", cfg.WorkerConcurrency)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Info("shutting down worker")
	case <-ctx.Done():
	}
	cancel()
	time.Sleep(500 * time.Millisecond)
}

func connectRabbit(url string, log *slog.Logger) (*amqp.Connection, error) {
	var lastErr error
	for i := 0; i < 30; i++ {
		conn, err := messaging.Connect(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Warn("waiting for rabbitmq", "attempt", i+1, "error", err)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
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
