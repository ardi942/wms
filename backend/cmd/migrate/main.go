package main

import (
	"log/slog"
	"os"

	"wms/backend/internal/database"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := env("DATABASE_URL", "postgres://wms:wms@localhost:5432/wms?sslmode=disable")
	migrationsPath := env("MIGRATIONS_PATH", "./migrations")
	if err := database.Migrate(databaseURL, migrationsPath); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
	log.Info("migrations applied")
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
