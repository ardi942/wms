package main

import (
	"flag"
	"log/slog"
	"os"

	"wms/backend/internal/database"
)

func main() {
	skipMigrate := flag.Bool("skip-migrate", false, "skip running database migrations")
	reset := flag.Bool("reset", false, "truncate demo tables before seeding (destructive)")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	databaseURL := env("DATABASE_URL", "postgres://wms:wms@localhost:5432/wms?sslmode=disable")
	migrationsPath := env("MIGRATIONS_PATH", "./migrations")

	if !*skipMigrate {
		if err := database.Migrate(databaseURL, migrationsPath); err != nil {
			log.Error("migration failed", "error", err)
			os.Exit(1)
		}
		log.Info("migrations applied")
	}
	db, err := database.Open(databaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	enabled := env("SEED_DEMO", "true") != "false"
	if *reset {
		if !enabled {
			log.Error("refusing to reset demo data while SEED_DEMO is disabled")
			os.Exit(1)
		}
		log.Warn("resetting demo data; all existing rows will be deleted")
		if err := database.Reset(db); err != nil {
			log.Error("reset failed", "error", err)
			os.Exit(1)
		}
		log.Info("demo tables truncated")
	}
	if err := database.Run(db, enabled); err != nil {
		log.Error("seed failed", "error", err)
		os.Exit(1)
	}
	log.Info("seed completed", "demo_enabled", enabled)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
