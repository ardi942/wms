# WMS — Warehouse Management System

A multi-department warehouse management MVP: a React SPA, a Go REST API, PostgreSQL, Redis, RabbitMQ and a Go worker.

Features include product catalogue and inventory, receiving/adjustments, multi-line stock requests with partial fulfilment, guarded stock returns, role-specific dashboards, CSV reports, notifications and audited append-only stock movements.

## Stack

| Layer | Technology |
|---|---|
| Frontend | React 19, TypeScript, Vite, Tailwind CSS v4, React Router v7, TanStack Query v5, React Hook Form + Zod, Recharts, Sonner |
| Backend | Go 1.24, Gin, GORM, golang-migrate, JWT (HS256), bcrypt |
| Data | PostgreSQL 17, Redis 7 (cache), RabbitMQ 4 (transactional outbox + worker) |
| Runtime | Docker Compose, nginx |

## Quick start (Docker Compose)

```bash
cp .env.example .env
# Edit JWT_SECRET (at least 32 characters) and database passwords.
# If host ports 3000/8080 are already in use, set WMS_FRONTEND_PORT/WMS_API_PORT.
docker compose up --build -d
docker compose ps
# Examples below use the default API port 8080 and app port 3000.
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
```

Open `http://localhost:3000` (or `WMS_FRONTEND_PORT`). Substitute the ports configured in `.env` if you changed them. If login shows a different application or a 404, see [docs/PORT_CONFIGURATION.md](docs/PORT_CONFIGURATION.md).

Startup order is enforced with healthchecks: `migrate` → `seed` → `backend`/`worker` → `frontend`. The seed is idempotent, so restarts do not duplicate data.

### Demo accounts

| Role | Email | Password |
|---|---|---|
| Supervisor | `supervisor@wms.local` | `Supervisor123!` |
| Warehouse Admin | `warehouse@wms.local` | `Warehouse123!` |
| Department User (IT) | `staff@wms.local` | `Staff123!` |
| Department User (FIN) | `staff.fin@wms.local` | `Staff123!` |

### Services

| Service | URL |
|---|---|
| App | `http://localhost:3000` or `WMS_FRONTEND_PORT` |
| API | `http://localhost:8080` or `WMS_API_PORT` |
| RabbitMQ management | http://localhost:15672 (`wms` / `wms` by default) |
| PostgreSQL | `localhost:5432` (`wms` / `wms`) |

## Local development

Prerequisites: Go 1.24+, Node 22+, Docker.

```bash
# Infrastructure only
docker compose up -d postgres redis rabbitmq

# Backend
cd backend
export DATABASE_URL='postgres://wms:wms@localhost:5432/wms?sslmode=disable'
export JWT_SECRET='replace-this-development-secret-at-least-32'
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api        # terminal 1
go run ./cmd/worker     # terminal 2

# Frontend
cd frontend
npm install
npm run dev             # http://localhost:5173, proxies /api to :8080
```

To proxy to a non-default API port, run `VITE_API_PROXY_TARGET=http://localhost:18081 npm run dev`.

The repository `Makefile` wraps the common commands: `make dev-up`, `make migrate`, `make seed`, `make backend`, `make worker`, `make frontend`, `make test`.

## Configuration

See `.env.example` for all variables. Important ones:

- `JWT_SECRET` — required, minimum 32 characters; startup fails otherwise.
- `DATABASE_URL` — API/worker/seed database connection.
- `REDIS_ADDR` — cache address; if Redis is unavailable the API falls back to PostgreSQL and readiness reports `degraded`.
- `RABBITMQ_URL` — outbox publishing and notification worker.
- `SEED_DEMO` — set `false` in production to skip demo data.
- `CORS_ORIGINS` — comma-separated allowed origins for local development.
- `WMS_FRONTEND_PORT` / `WMS_API_PORT` — host ports published by Compose. They default to `3000` and `8080`; change them when those ports are occupied. Container-internal ports stay fixed (`frontend` 8080, `backend` 8080).

## Architecture

```
frontend (React SPA) ──/api──> backend (Gin) ──> PostgreSQL
                                   │  │
                                   │  └──> Redis (versioned cache)
                                   └──> outbox_events ──> RabbitMQ ──> worker ──> notifications
```

- **Stock is append-only.** Every change inserts a `stock_movements` row and updates `inventory_balances` in the same transaction. A database trigger rejects updates/deletes on movements.
- **No oversell.** Fulfilment locks balances with `SELECT ... FOR UPDATE`, validates remaining stock, then applies movements.
- **Returns are capped** by the department's net issued quantity (`issued - returned`) for each product, enforced under advisory locks.
- **Transactional outbox** guarantees events are published after commit. The relay uses `FOR UPDATE SKIP LOCKED`, retry backoff, and `PENDING`/`PUBLISHED`/`FAILED` states.
- **Idempotency** is enforced by middleware for mutating document endpoints: the response is stored and replayed for the same `Idempotency-Key`; conflicting payloads return `409`.
- **RBAC** is authoritative on the backend; the UI hides actions by role but is never trusted.

## Project layout

```
backend/
  cmd/{api,worker,seed,migrate}
  internal/{auth,user,department,category,unit,product,inventory,stockrequest,stockreturn,report,notification}
  internal/{cache,messaging,worker,idempotency,middleware,httpx,database,config}
  migrations/0001..0008_*.{up,down}.sql
frontend/
  src/{api,auth,components,features,lib,types,test}
  e2e/
docs/
  IMPLEMENTATION_PLAN.md
  USER_GUIDE.md
```

## Tests

```bash
# Backend unit tests
cd backend && go test ./...
go vet ./...

# Frontend tests and build
cd frontend
npm run lint
npm test
npm run build
```

Playwright E2E requires a running stack and a browser. On a host without browser system libraries, use the official Playwright image:

```bash
cd frontend
docker run --rm --network host --user "$(id -u):$(id -g)" -e HOME=/tmp \
  -e E2E_BASE_URL=http://localhost:3000 -e E2E_API_URL=http://localhost:8080/api/v1 \
  -v "$PWD":/work -w /work mcr.microsoft.com/playwright:v1.65.0-noble npx playwright test
```

Set `E2E_BASE_URL`/`E2E_API_URL` to the configured host ports when they differ. The Makefile target uses `WMS_FRONTEND_PORT`/`WMS_API_PORT` from the environment:

```bash
export WMS_FRONTEND_PORT=13000 WMS_API_PORT=18081
make e2e
```

On a workstation with `npx playwright install --with-deps chromium`, run `npm run test:e2e`. The suite is serial and mutates the demo database; rerun against a fresh seed when you need a clean baseline.

## API

All responses use `{ "data": ..., "meta": ... }`; errors use `{ "error": { "code", "message", "details" } }`.

- `POST /api/v1/auth/login`, `GET /api/v1/auth/me`
- `/api/v1/products`, `/inventory`, `/inventory/movements`, `/inventory/receipts`, `/inventory/adjustments`
- `/api/v1/stock-requests` (+ `/:id/fulfill`, `/:id/reject`, `/:id/cancel`)
- `/api/v1/stock-returns` (+ `/:id/confirm`, `/:id/reject`, `/:id/cancel`)
- `/api/v1/reports/{dashboard,inventory,stock-movements,department-usage,low-stock,export}`
- `/api/v1/users`, `/departments`, `/categories`, `/units`
- `/api/v1/notifications`

Mutating document endpoints require `Idempotency-Key` (8–128 characters). The frontend supplies one automatically.

## Operations

```bash
docker compose logs -f migrate seed backend worker
docker compose down          # stop
docker compose down -v       # stop and delete data
docker compose up --build -d # rebuild and start
make reset                   # down -v + up --build
```

If the worker cannot reach RabbitMQ it retries for up to 60 seconds at startup. Outbox events remain `PENDING` while the broker is down and are published after recovery.

## Troubleshooting login and ports

If the login page does not appear, or login returns a 404/HTML error:

1. Confirm the WMS containers are the ones publishing the ports:
   ```bash
   docker compose ps
   ss -ltn | grep -E ':(3000|8080|13000|18081) '
   ```
2. If `3000` or `8080` is owned by another project, set free ports in `.env`:
   ```bash
   WMS_FRONTEND_PORT=13000
   WMS_API_PORT=18081
   docker compose up -d --build
   ```
3. Verify the API directly:
   ```bash
   curl -fsS http://localhost:18081/readyz
   curl -fsS -X POST http://localhost:13000/api/v1/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"warehouse@wms.local","password":"Warehouse123!"}'
   ```

Full diagnosis and an agent-oriented checklist: [docs/PORT_CONFIGURATION.md](docs/PORT_CONFIGURATION.md).

## Known deviations from the plan

- Idempotency is implemented as HTTP middleware rather than inside each service transaction. It reserves a key, executes the handler, stores the response on success, and releases the key on error.
- RabbitMQ delivery retries use explicit republishing with an `x-retry` header and a `wms.dead-letter` queue instead of a broker DLX policy.
- The frontend uses locally authored Tailwind components instead of generated shadcn/ui files.
- CSV exports stream from the API; the frontend downloads them as authenticated blobs.
- Compose host ports are configurable via `WMS_FRONTEND_PORT`/`WMS_API_PORT`; the plan's 3000/8080 remain the defaults.
