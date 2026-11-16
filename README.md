# WMS — Warehouse Management System

A multi-department warehouse management MVP: a React SPA, a Go REST API, PostgreSQL, Redis, RabbitMQ and a Go worker.

Features include product catalogue and inventory, receiving/adjustments, multi-line stock requests with partial fulfilment, guarded stock returns, cross-division status monitoring with live polling, role-specific dashboards, CSV reports, notifications, warehouse-admin user and role management, audited append-only stock movements, and an immutable account/product audit history with required reasons for user activation changes.

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

Open `http://localhost:3000` (or `WMS_FRONTEND_PORT`). Substitute the ports configured in `.env` if you changed them. If login shows a different application or a 404, see [Troubleshooting login and ports](#troubleshooting-login-and-ports).

Startup order is enforced with healthchecks: `migrate` → `seed` → `backend`/`worker` → `frontend`. The seed is idempotent, so restarts do not duplicate data.

### Demo accounts

| Role | Email | Password |
|---|---|---|
| Supervisor | `supervisor@wms.local` | `Supervisor123!` |
| Warehouse Admin | `warehouse@wms.local` | `Warehouse123!` |
| Department User (IT) | `staff@wms.local` | `Staff123!` |
| Department User (FIN) | `staff.fin@wms.local` | `Staff123!` |

The Warehouse Admin account can create additional department users, supervisors, and warehouse admins from the Users page.

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

The repository `Makefile` wraps the common commands: `make dev-up`, `make migrate`, `make seed`, `make seed-reset`, `make backend`, `make worker`, `make frontend`, `make test`.

## Configuration

See `.env.example` for all variables. Important ones:

- `JWT_SECRET` — required, minimum 32 characters; startup fails otherwise.
- `DATABASE_URL` — API/worker/seed database connection.
- `REDIS_ADDR` — cache address; if Redis is unavailable the API falls back to PostgreSQL and readiness reports `degraded`.
- `RABBITMQ_URL` — outbox publishing and notification worker.
- `SEED_DEMO` — set `false` in production to skip demo data.
- `CORS_ORIGINS` — comma-separated allowed origins for local development.
- `UPLOAD_DIR` — local directory for product and user images. Compose defaults to the `uploads` volume at `/app/uploads`; local development defaults to `./uploads`.
- `IMAGE_MAX_BYTES` — maximum image size in bytes (64 KiB–8 MiB, default `2097152` = 2 MiB).
- `WMS_FRONTEND_PORT` / `WMS_API_PORT` — host ports published by Compose. They default to `3000` and `8080`; change them when those ports are occupied. Container-internal ports stay fixed (`frontend` 8080, `backend` 8080).

## Architecture

```
frontend (React SPA) ──/api──> backend (Gin) ──> PostgreSQL
                                   │  │
                                   │  └──> Redis (versioned cache)
                                   └──> outbox_events ──> RabbitMQ ──> worker ──> notifications
```

- **Stock is append-only.** Every change inserts a `stock_movements` row and updates `inventory_balances` in the same transaction. A database trigger rejects updates/deletes on movements.
- **Audit history is append-only.** User creation/status/profile/password changes and product cost/status/profile changes write an `audit_logs` row in the same transaction. Actor snapshots, reasons, before/after values and request IDs are retained; a trigger rejects updates/deletes. Product cost values are redacted for Warehouse Admin readers.
- **No oversell.** Fulfilment locks balances with `SELECT ... FOR UPDATE`, validates remaining stock, then applies movements.
- **Returns are capped** by the department's net issued quantity (`issued - returned`) for each product, enforced under advisory locks.
- **Transactional outbox** guarantees events are published after commit. The relay uses `FOR UPDATE SKIP LOCKED`, retry backoff, and `PENDING`/`PUBLISHED`/`FAILED` states.
- **Idempotency** is enforced by middleware for mutating document endpoints: the response is stored and replayed for the same `Idempotency-Key`; conflicting payloads return `409`.
- **User management.** Warehouse Admins can list and filter users, create department users, supervisors, or warehouse admins, edit profiles/passwords, activate/deactivate accounts, and view account history. Roles are immutable after creation. Status changes are serialized with an advisory lock, and the last active warehouse admin cannot be deactivated.
- **Live polling.** Monitoring and request/return lists refresh every 15 seconds while visible; notifications and the unread badge refresh every 30 seconds. Lists keep the previous page visible while refetching.
- **RBAC** is authoritative on the backend; the UI hides actions by role but is never trusted.
- **Images are optional** for products and users. JPEG/PNG files up to 2 MiB and 4096 px per side (16 MP total) are stored on the local filesystem behind a storage-neutral `image_key`; delivery is JWT-protected and streamed with range support. The storage interface leaves room for a future S3-compatible driver.

## Project layout

```
backend/
  cmd/{api,worker,seed,migrate}
  internal/{auth,user,department,category,unit,product,inventory,stockrequest,stockreturn,report,notification,audit}
  internal/{cache,messaging,worker,idempotency,middleware,httpx,database,config,media}
  migrations/0001..0010_*.{up,down}.sql
  uploads/                  # local image storage (git-ignored)
frontend/
  src/{api,auth,components,features,lib,types,test}
  e2e/
```

## Tests

The DB-gated suites cover integration, concurrency, security, outbox, and worker behavior, plus fuzz, load, CI, and tooling.

```bash
# Fast/local (DB-free unit + frontend)
make test

# Backend static/unit checks
cd backend && go vet ./... && go test -short ./...

# DB-backed suites (requires TEST_DATABASE_URL)
cd backend && TEST_DATABASE_URL='postgres://wms:wms@localhost:5432/wms?sslmode=disable' \
  go test ./test/api ./test/security ./test/outbox ./test/worker ./test/concurrency -count=1

# Concurrency/race proof
make test-race

# Tooling (install tools locally; CI runs them)
make lint-backend
make audit
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

On a workstation with `npx playwright install --with-deps chromium`, run `npm run test:e2e`. The suite is serial and mutates the demo database. `make e2e` automatically runs `make seed-reset` afterwards, even when tests fail; when invoking Playwright directly, run `make seed-reset` manually to restore a clean baseline.

## API

All responses use `{ "data": ..., "meta": ... }`; errors use `{ "error": { "code", "message", "details" } }`.

- `POST /api/v1/auth/login`, `GET /api/v1/auth/me`
- `/api/v1/products`, `/inventory`, `/inventory/movements`, `/inventory/receipts`, `/inventory/adjustments`
- `/api/v1/stock-requests` (+ `/:id/fulfill`, `/:id/reject`, `/:id/cancel`)
- `/api/v1/stock-returns` (+ `/:id/confirm`, `/:id/reject`, `/:id/cancel`)
- `/api/v1/monitoring/requests`, `/api/v1/monitoring/returns` — read-only cross-division status; the frontend refreshes every 15 seconds while visible
- `/api/v1/reports/{dashboard,inventory,stock-movements,department-usage,low-stock,export}`
- `/api/v1/users` — Warehouse Admin only; optional `?role=DEPARTMENT_USER|SUPERVISOR|WAREHOUSE_ADMIN` filter. Create accepts `role` (defaults to `DEPARTMENT_USER`); elevated roles must omit `department_id`
- `/api/v1/departments`, `/categories`, `/units`
- `/api/v1/users/:id/{activate,deactivate,history}`, `/api/v1/products/:id/history` — roles are immutable after creation; activation/deactivation requires a reason and the last active warehouse admin is protected
- `/api/v1/audit` — read-only audit history for Warehouse Admin and Supervisor
- `/api/v1/notifications`

### Product and user images

| Method | Path | Access |
|---|---|---|
| `GET` | `/api/v1/products/:id/image` | Any authenticated role; Department Users cannot see inactive products |
| `POST` | `/api/v1/products/:id/image` | Warehouse Admin only |
| `DELETE` | `/api/v1/products/:id/image` | Warehouse Admin only |
| `GET` | `/api/v1/users/:id/image` | Self or Warehouse Admin |
| `POST` | `/api/v1/users/:id/image` | Warehouse Admin only |
| `DELETE` | `/api/v1/users/:id/image` | Warehouse Admin only |

Uploads use `multipart/form-data` with the field name `image`, accept JPEG/PNG only, and return the updated product/user DTO. Deletes return `204`, including repeated deletes. Image reads are authenticated, streamed from `UPLOAD_DIR`, and returned with `Cache-Control: private, max-age=3600`; invalid formats return `422`, oversized bodies or files return `413`, and missing entities return `404`. Image changes reuse the `PRODUCT_UPDATED` / `USER_PROFILE_UPDATED` audit actions with the `image` field.

Mutating document endpoints require `Idempotency-Key` (8–128 characters). The frontend supplies one automatically. Image uploads do not use idempotency. User activation/deactivation requires a trimmed reason of 5–500 characters; `PATCH /users/:id` rejects `is_active`.

## Operations

```bash
docker compose logs -f migrate seed backend worker
docker compose down          # stop
docker compose down -v       # stop and delete data
docker compose up --build -d # rebuild and start
make reset                   # down -v + up --build
```

If the worker cannot reach RabbitMQ it retries for up to 60 seconds at startup. Outbox events remain `PENDING` while the broker is down and are published after recovery.

Images live in the Compose `uploads` volume. Back up PostgreSQL and the `uploads` volume together; restoring one without the other can leave image metadata pointing at missing files (or orphan files without metadata).

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

## Known deviations from the plan

- Idempotency is implemented as HTTP middleware rather than inside each service transaction. It reserves a key, executes the handler, stores the response on success, and releases the key on error.
- RabbitMQ delivery retries use explicit republishing with an `x-retry` header and a `wms.dead-letter` queue instead of a broker DLX policy.
- The frontend uses locally authored Tailwind components instead of generated shadcn/ui files.
- CSV exports stream from the API; the frontend downloads them as authenticated blobs.
- Compose host ports are configurable via `WMS_FRONTEND_PORT`/`WMS_API_PORT`; the plan's 3000/8080 remain the defaults.
