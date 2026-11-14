SHELL := /bin/bash
GO := go
NPM := npm
DATABASE_URL ?= postgres://wms:wms@localhost:5432/wms?sslmode=disable

.PHONY: help install dev-up dev-down migrate seed seed-reset backend worker frontend \
	build test test-backend test-frontend lint e2e compose-up compose-down reset clean redeploy

help:
	@echo "WMS development targets"
	@echo "  make install        Install frontend dependencies"
	@echo "  make dev-up         Start postgres, redis and rabbitmq"
	@echo "  make migrate        Run migrations"
	@echo "  make seed           Seed demo data (idempotent)"
	@echo "  make seed-reset     Truncate demo data and reseed from scratch"
	@echo "  make backend        Run the API locally"
	@echo "  make worker         Run the worker locally"
	@echo "  make frontend       Run the Vite dev server"
	@echo "  make test           Run backend and frontend tests"
	@echo "  make build          Build backend binaries and frontend"
	@echo "  make compose-up     Start the full Docker Compose stack"
	@echo "  make redeploy       Commit, rebuild and redeploy (ARGS=\"-m msg -y\")"
	@echo "  make reset          Destroy compose volumes and rebuild from scratch"

install:
	cd frontend && $(NPM) install

dev-up:
	docker compose up -d postgres redis rabbitmq

dev-down:
	docker compose stop postgres redis rabbitmq

migrate:
	cd backend && DATABASE_URL="$(DATABASE_URL)" $(GO) run ./cmd/migrate

seed:
	cd backend && DATABASE_URL="$(DATABASE_URL)" SEED_DEMO=true $(GO) run ./cmd/seed --skip-migrate

seed-reset:
	docker compose run --rm --build seed /app/seed --skip-migrate --reset
	-docker compose exec -T redis redis-cli FLUSHDB

backend:
	cd backend && DATABASE_URL="$(DATABASE_URL)" $(GO) run ./cmd/api

worker:
	cd backend && DATABASE_URL="$(DATABASE_URL)" $(GO) run ./cmd/worker

frontend:
	cd frontend && $(NPM) run dev

build:
	cd backend && $(GO) build ./...
	cd frontend && $(NPM) run build

test: test-backend test-frontend

test-backend:
	cd backend && $(GO) test ./...

test-frontend:
	cd frontend && $(NPM) test

lint:
	cd frontend && $(NPM) run lint

e2e:
	@cd frontend && E2E_BASE_URL=$${E2E_BASE_URL:-http://localhost:$${WMS_FRONTEND_PORT:-3000}} E2E_API_URL=$${E2E_API_URL:-http://localhost:$${WMS_API_PORT:-8080}/api/v1} npx playwright test; \
	status=$$?; \
	$(MAKE) -C "$(CURDIR)" seed-reset; \
	exit $$status

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

redeploy:
	scripts/redeploy.sh $(ARGS)

reset:
	docker compose down -v
	docker compose up --build -d

clean:
	rm -rf frontend/dist backend/bin
