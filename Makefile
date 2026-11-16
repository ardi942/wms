SHELL := /bin/bash
GO := go
NPM := npm
DATABASE_URL ?= postgres://wms:wms@localhost:5432/wms?sslmode=disable

.PHONY: help install dev-up dev-down migrate seed seed-reset backend worker frontend \
	build test test-unit test-integration test-race test-security test-fuzz test-load test-all \
	test-backend test-frontend lint lint-backend lint-security audit e2e compose-up compose-down reset clean redeploy

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
	@echo "  make test           Run backend unit and frontend tests (DB-free)"
	@echo "  make test-unit      Run backend unit tests"
	@echo "  make test-integration  Run backend integration/concurrency tests (TEST_DATABASE_URL)"
	@echo "  make test-race      Run integration/concurrency tests with -race -count=10"
	@echo "  make test-security  Run backend security package"
	@echo "  make test-fuzz      Run native fuzz targets (short)"
	@echo "  make test-load      Run k6 smoke against a running stack"
	@echo "  make test-all       Run unit, integration, frontend, and lint"
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

test: test-unit test-frontend

test-unit:
	cd backend && $(GO) test -short ./...

test-backend: test-unit

test-integration:
	cd backend && $(GO) test ./test/api ./test/security ./test/outbox ./test/worker ./test/concurrency

test-race:
	cd backend && $(GO) test -race -count=10 ./test/concurrency

test-security:
	cd backend && $(GO) test -count=1 ./test/security

test-fuzz:
	cd backend && $(GO) test -run '^$$' -fuzz=Fuzz -fuzztime=5s ./internal/httpx ./internal/report ./internal/middleware ./internal/idempotency

test-load:
	k6 run tests/load/k6/smoke.js

test-all: test-unit test-integration test-frontend lint

test-frontend:
	cd frontend && $(NPM) test

lint: lint-backend
	cd frontend && $(NPM) run lint

lint-backend:
	cd backend && $(GO) vet ./...
	-cd backend && command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed (go install honnef.co/go/tools/cmd/staticcheck@latest)"

lint-security:
	-cd backend && command -v gosec >/dev/null 2>&1 && gosec -exclude-generated ./... || echo "gosec not installed (go install github.com/securego/gosec/v2/cmd/gosec@latest)"
	-cd backend && command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed (go install golang.org/x/vuln/cmd/govulncheck@latest)"

audit:
	cd frontend && $(NPM) audit --audit-level=high || true

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
