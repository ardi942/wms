# Project: Warehouse Management System (WMS) — Minimal Working MVP

## 1. Role and Objective

Act as a Senior Software Architect, Senior Golang Backend Engineer, Senior React Frontend Engineer, and QA Engineer.

Design and implement a practical Warehouse Management System (WMS) for internal company operations.

Build a functional, maintainable, and production-minded MVP. Prioritize real business workflows, data integrity, usability, and reliability.

**Development principles:**
- Build only essential features.
- Avoid overengineering, unnecessary abstractions, and premature optimization.
- Do not implement microservices. Use a modular monolith.
- Prefer simple, readable, maintainable code.
- Every implemented feature must work end-to-end.
- Use realistic sample data for demonstration.
- Do not leave fake buttons, non-functional pages, or placeholder business logic.
- Do not implement features just because they are common in enterprise software.
- Make reasonable assumptions and document them rather than continuously asking questions.

The result must be a working application, not merely an architecture proposal or UI prototype.

---

## 2. Technology Stack

### Frontend
- React
- TypeScript
- Vite
- Tailwind CSS
- A lightweight component library such as shadcn/ui
- React Router
- TanStack Query for API state management
- React Hook Form + Zod for forms
- Recharts for simple reports

### Backend
- Golang (latest stable version)
- REST API
- Gin or Chi HTTP router
- GORM or sqlc for database access (choose one)
- PostgreSQL as the primary database
- JWT-based authentication
- bcrypt password hashing
- Structured logging
- Database migrations

### Infrastructure
- Docker
- Docker Compose
- PostgreSQL
- Redis for caching
- RabbitMQ for asynchronous background processing

### Architecture

Use a modular monolith:

React Frontend → Golang REST API → PostgreSQL

Supporting infrastructure:
- Redis: cache frequently accessed read-only data, such as product lists and dashboard summaries.
- RabbitMQ: asynchronous processing of non-critical tasks, such as stock movement notifications and report generation.
- Go routines: bounded background workers for RabbitMQ consumers and other asynchronous tasks.

Do not use Redis as the primary inventory database.

Do not use RabbitMQ for synchronous stock transactions. A stock transaction must be committed successfully in PostgreSQL before asynchronous events are published.

Use a transactional outbox pattern only if needed to guarantee event delivery. Keep the implementation as simple as possible while maintaining reliability.

---

## 3. User Roles and Permissions

Implement exactly three initial user roles.

### A. Supervisor

Purpose: Monitor warehouse activities and business performance.

Permissions:
- View dashboard.
- View stock levels.
- View low-stock products.
- View stock movement history.
- View incoming and outgoing stock reports.
- View department stock requests.
- View inventory valuation if unit costs are available.
- Filter reports by date, product, warehouse, and department.
- Export reports to CSV.

Restrictions:
- Cannot modify inventory.
- Cannot create or modify products.
- Cannot perform stock movements.

### B. Warehouse Admin

Purpose: Manage daily warehouse operations.

Permissions:
- View warehouse dashboard.
- Create and manage products.
- Manage product categories and units.
- Manage stock receiving.
- View incoming stock.
- Process department stock requests.
- Issue stock to departments.
- Record stock returns.
- Perform authorized stock adjustments with mandatory reasons.
- View inventory and stock movement history.
- View low-stock products.
- View department stock requests.

Restrictions:
- Cannot manage user roles.
- Cannot delete historical stock transactions.

### C. Department User

Purpose: Request products and monitor stock usage for their division.

Permissions:
- View available products and stock information.
- View their department's stock requests.
- Create stock requests.
- View request status.
- View their department's incoming and outgoing stock history.
- View their department's stock usage summary.

Restrictions:
- Cannot directly change warehouse stock.
- Cannot process stock receiving.
- Cannot perform inventory adjustments.
- Cannot view other departments' private requests.

Each department user must be associated with a department.

Enforce authorization on the backend, not only by hiding frontend menu items.

---

## 4. Core MVP Modules

Implement only the following modules.

### Module 1: Authentication and User Management

Features:
- Login and logout.
- JWT authentication.
- Password hashing.
- Role-based access control.
- Department association.
- Seed initial demo users.

No public registration is required.

### Module 2: Dashboard

Create role-specific dashboards.

Supervisor:
- Total products.
- Total available stock.
- Low-stock products.
- Incoming stock summary.
- Outgoing stock summary.
- Pending department requests.
- Recent stock movements.
- Stock movement chart.

Warehouse Admin:
- Current stock overview.
- Low-stock alerts.
- Pending stock requests.
- Today's receiving transactions.
- Today's outgoing transactions.
- Recent warehouse activities.

Department User:
- Available products.
- My pending requests.
- My approved/fulfilled requests.
- My department's stock usage.
- Recent department transactions.

Use real API data. Do not hardcode dashboard statistics.

### Module 3: Product Management

Product fields:
- Product ID / SKU (unique).
- Product name.
- Description.
- Category.
- Unit of measurement.
- Minimum stock threshold.
- Optional unit cost.
- Active/inactive status.
- Created date.

Features:
- Create product.
- Edit product.
- View product details.
- Search by name or SKU.
- Filter by category.
- Pagination.
- Display current stock.

Do not allow deleting products that have transaction history. Use active/inactive status instead.

### Module 4: Inventory Management

Maintain current stock by product and warehouse.

Essential features:
- View stock list.
- Search and filter products.
- View current quantity.
- View minimum stock.
- Display low-stock warnings.
- View stock movement history.
- View product transaction history.

Stock must never be updated through arbitrary CRUD operations.

All stock changes must come from recorded inventory transactions.

Use an inventory ledger as the source of truth.

Each stock movement must record:
- Transaction ID.
- Product.
- Transaction type.
- Quantity.
- Reference number.
- Source or destination department where applicable.
- Performed by.
- Transaction date.
- Notes or reason.

Use database transactions and appropriate locking or atomic updates to prevent overselling and negative inventory.

### Module 5: Stock Receiving (IN)

Purpose: Record products entering the warehouse.

Workflow:
1. Warehouse admin opens Stock Receiving.
2. Selects a product.
3. Enters received quantity.
4. Adds supplier or source information.
5. Adds reference number and optional notes.
6. Reviews the transaction.
7. Confirms receiving.
8. System records an IN movement.
9. System increases inventory.
10. System updates the dashboard and stock history.

For MVP, supplier information can be stored as simple text. Do not build a complete procurement module.

### Module 6: Department Stock Request

Purpose: Allow other divisions to request products from the warehouse.

Workflow:

1. Department user logs in.
2. Opens Stock Request.
3. Searches available products.
4. Selects product and requested quantity.
5. Adds purpose or usage notes.
6. Submits request.
7. System creates a PENDING request.
8. Warehouse admin sees the request in the pending queue.
9. Admin reviews available stock.
10. Admin either rejects the request with a reason or fulfills it with an issued quantity.
11. System records an OUT stock movement when fulfillment is confirmed.
12. Inventory is decreased.
13. Request status changes to FULFILLED.
14. Department user can view the result.

Important rules:
- A department user cannot issue stock directly.
- A request does not reduce inventory.
- Stock is reduced only when the warehouse admin confirms fulfillment.
- The fulfilled quantity cannot exceed available stock.
- Support partial fulfillment.
- Record who processed each request.
- Prevent duplicate fulfillment.
- Maintain a complete request history.

Request statuses:
- PENDING
- PARTIALLY_FULFILLED
- FULFILLED
- REJECTED
- CANCELLED

Allow department users to cancel requests only while they are PENDING.

### Module 7: Stock Return

Purpose: Return unused products from a department to the warehouse.

Workflow:
1. Department user submits a return request with product, quantity, and reason.
2. Warehouse admin reviews the returned goods.
3. Admin confirms the actual accepted quantity.
4. System records an IN movement.
5. Inventory is increased.
6. Return history is recorded.

Do not allow department users to increase warehouse stock directly.

### Module 8: Stock Adjustment

Purpose: Correct physical inventory discrepancies.

Workflow:
1. Warehouse admin opens Stock Adjustment.
2. Selects product.
3. Enters the actual quantity adjustment (+/-).
4. Provides a mandatory reason.
5. Reviews and confirms.
6. System records an ADJUSTMENT transaction.
7. Inventory is updated atomically.

Every adjustment must be traceable.

### Module 9: Reports

Keep reporting simple and useful.

Supervisor reports:
- Current inventory report.
- Incoming stock report.
- Outgoing stock report.
- Stock movement history.
- Low-stock report.
- Department stock usage report.

Provide:
- Date filters.
- Product filters.
- Department filters.
- CSV export.

Department users can only access their own department's reports.

Avoid implementing a complicated reporting engine.

---

## 5. Essential Business Rules

These rules are mandatory.

1. Inventory must never become negative.
2. All stock movements must be recorded in an immutable transaction ledger.
3. No direct stock quantity editing.
4. Every stock operation must be atomic.
5. Use database transactions for inventory updates.
6. Prevent duplicate submissions and duplicate fulfillment.
7. Validate quantities on both frontend and backend.
8. Department users must only access their own department data.
9. Every important operation must record the acting user and timestamp.
10. Use soft deactivation for products with transaction history.
11. Use proper error responses and meaningful validation messages.
12. Use Redis only for cacheable data, never as the source of truth for stock.
13. Invalidate relevant caches after successful stock changes.
14. Use database constraints to protect critical business rules.
15. Ensure RabbitMQ failure does not silently invalidate a successfully committed stock transaction.

For concurrent stock operations, use appropriate row-level locking or atomic SQL updates with sufficient constraints.

---

## 6. Database Design

Create a minimal relational schema.

Suggested tables:

- users
- roles (or a simple role enum)
- departments
- products
- categories
- units
- warehouses (optional if initially operating one warehouse)
- inventory_balances
- stock_movements
- stock_requests
- stock_request_items
- stock_returns
- stock_return_items
- notifications (only if required)
- outbox_events (only if using transactional outbox)

Use:
- UUID or consistent integer primary keys.
- Foreign keys.
- Unique constraints.
- Appropriate indexes.
- created_at and updated_at.
- Database migrations.

Do not create unnecessary tables.

Before implementation, explain the relationship between inventory balances and stock movements.

The stock movement ledger must remain the historical record of all inventory changes.

---

## 7. Backend API Design

Use RESTful API conventions.

Suggested endpoints:

Authentication:
- POST /api/v1/auth/login
- GET /api/v1/auth/me

Users:
- GET /api/v1/users
- POST /api/v1/users
- PATCH /api/v1/users/:id

Products:
- GET /api/v1/products
- POST /api/v1/products
- GET /api/v1/products/:id
- PATCH /api/v1/products/:id

Inventory:
- GET /api/v1/inventory
- GET /api/v1/inventory/:productId
- GET /api/v1/inventory/movements
- POST /api/v1/inventory/receipts
- POST /api/v1/inventory/adjustments

Stock Requests:
- GET /api/v1/stock-requests
- POST /api/v1/stock-requests
- GET /api/v1/stock-requests/:id
- POST /api/v1/stock-requests/:id/fulfill
- POST /api/v1/stock-requests/:id/reject
- POST /api/v1/stock-requests/:id/cancel

Stock Returns:
- POST /api/v1/stock-returns
- GET /api/v1/stock-returns
- POST /api/v1/stock-returns/:id/confirm

Reports:
- GET /api/v1/reports/dashboard
- GET /api/v1/reports/inventory
- GET /api/v1/reports/stock-movements
- GET /api/v1/reports/department-usage
- GET /api/v1/reports/low-stock
- GET /api/v1/reports/export

Use pagination, filtering, consistent response formats, and meaningful HTTP status codes.

Do not expose internal database models directly as API responses. Use request and response DTOs.

---

## 8. Golang Project Structure

Use a simple modular monolith structure.

Suggested structure:

backend/
  cmd/
    api/
      main.go
    worker/
      main.go
  internal/
    auth/
    user/
    department/
    product/
    inventory/
    stockrequest/
    stockreturn/
    report/
    messaging/
    cache/
    worker/
    database/
    middleware/
    config/
  migrations/
  tests/
  Dockerfile
  go.mod

Keep business logic separate from HTTP handlers and database access.

Use dependency injection through constructors and interfaces where they provide real value.

Do not introduce a complex dependency injection framework.

### Goroutine requirements

Use Go routines appropriately:
- RabbitMQ background consumers.
- Bounded worker pools.
- Asynchronous notification processing.
- Background cache refresh only when necessary.

Requirements:
- Use context.Context.
- Graceful shutdown.
- Proper goroutine lifecycle management.
- Bounded concurrency.
- Handle worker errors.
- Avoid goroutine leaks.
- Avoid spawning unbounded goroutines for every request.

Do not use goroutines for simple synchronous database operations.

### Redis requirements

Use Redis for:
- Dashboard summary caching.
- Product list caching where useful.
- Frequently accessed read-only reference data.

Requirements:
- Define TTL.
- Invalidate affected keys after stock changes.
- Graceful fallback to PostgreSQL if Redis is unavailable.
- Never return stale inventory as authoritative data for stock fulfillment.

### RabbitMQ requirements

Use RabbitMQ for:
- Stock movement notifications.
- Asynchronous notification processing.
- Optional report generation.

Requirements:
- Acknowledge messages only after successful processing.
- Handle retries and failed messages.
- Avoid infinite retry loops.
- Use durable queues where appropriate.
- Make consumers idempotent.
- Keep stock transactions independent from consumer availability.

Do not introduce a complicated event-driven architecture.

---

## 9. Frontend UI/UX Requirements

Build a clean, modern, professional warehouse management interface.

Design principles:
- Simple and intuitive.
- Minimal navigation.
- Responsive desktop and tablet layouts.
- Consistent spacing and typography.
- Clear status indicators.
- Accessible forms and controls.
- Fast interactions.
- Avoid unnecessary animations.
- Avoid overly complicated dashboards.

### Main layout

Sidebar:
- Dashboard
- Products
- Inventory
- Stock Receiving
- Stock Requests
- Stock Returns
- Stock Adjustments
- Reports
- User Management (only where authorized)

Top navigation:
- Application name.
- Current user.
- Department.
- Notification indicator if implemented.
- Logout.

Menus must be filtered by user role.

### Important interactive components

Implement working:
- Searchable product selection.
- Data tables with pagination.
- Filters.
- Date range selectors.
- Stock request forms.
- Receiving forms.
- Confirmation dialogs.
- Status badges.
- Loading states.
- Empty states.
- Error states.
- Success notifications.
- CSV download.
- Dashboard charts.

Forms must validate inputs and display useful errors.

All data must be connected to the real backend APIs.

Do not use static mock data in production pages. Mock data is permitted only for development and automated tests.

### Role-specific experience

Supervisor:
A dashboard-first experience focused on reports and monitoring.

Warehouse Admin:
An operations-first experience focused on receiving, requests, fulfillment, and inventory.

Department User:
A simple self-service experience focused on product availability, requesting stock, and tracking requests.

---

## 10. Testing Strategy

Testing is mandatory. Do not consider the MVP complete just because the application compiles.

### Backend unit tests

Test:
- Authentication.
- Role-based authorization.
- Product validation.
- Stock receiving.
- Stock adjustment.
- Stock request creation.
- Request fulfillment.
- Partial fulfillment.
- Request rejection.
- Stock returns.
- Negative stock prevention.
- Duplicate transaction prevention.
- Department data isolation.

### Backend integration tests

Test with PostgreSQL:
- Database transactions.
- Inventory consistency.
- Concurrent fulfillment.
- Concurrent stock receiving.
- Rollback after transaction failure.
- Database constraints.

Test Redis cache invalidation and fallback.

Test RabbitMQ message processing, retries, and idempotency.

### Frontend tests

Test:
- Login.
- Role-based navigation.
- Product creation.
- Stock receiving.
- Stock request submission.
- Request fulfillment workflow.
- Form validation.
- API error handling.
- Unauthorized page access.

Use suitable React testing tools and API mocking for frontend tests.

### End-to-end tests

Implement at least these scenarios:

Scenario 1: Initial inventory
1. Warehouse admin logs in.
2. Creates a product.
3. Receives 100 units.
4. Verifies inventory is 100.

Scenario 2: Department stock request
1. Department user logs in.
2. Requests 20 units.
3. Warehouse admin logs in.
4. Fulfills the request.
5. Verifies inventory is 80.
6. Department user verifies the request is fulfilled.

Scenario 3: Partial fulfillment
1. Department requests 30 units.
2. Only 15 units are available.
3. Admin fulfills 15 units.
4. System records PARTIALLY_FULFILLED.
5. Verify inventory is zero.

Scenario 4: Negative stock prevention
1. Attempt to fulfill a request exceeding available stock.
2. System rejects the operation.
3. Verify inventory remains unchanged.

Scenario 5: Department isolation
1. Department A logs in.
2. Attempts to access Department B requests.
3. Backend rejects access.

Scenario 6: Stock return
1. Department submits a return request.
2. Warehouse admin confirms accepted quantity.
3. Verify inventory increases correctly.

---

## 11. Docker Deployment

Create Dockerfiles for frontend and backend.

Create docker-compose.yml containing:
- frontend
- backend API
- backend worker (if using a separate worker process)
- PostgreSQL
- Redis
- RabbitMQ

Requirements:
- Persistent PostgreSQL volume.
- Persistent RabbitMQ volume where appropriate.
- Environment-based configuration.
- Health checks.
- Service dependency management.
- Database migration execution.
- Restart policies.
- Docker network.
- No hardcoded production secrets.

Provide:
- .env.example
- Docker Compose configuration.
- Database migration instructions.
- Seed data instructions.
- Local development instructions.
- Production deployment instructions.

The application must be runnable with a simple command:

docker compose up --build -d

Provide a documented way to run database migrations and seed initial users.

Do not assume Docker Compose dependency ordering guarantees service readiness.

---

## 12. Demo Users and Seed Data

Create three demo accounts.

Use the following example credentials in development only:

| Role | Email | Password |
|---|---|---|
| Supervisor | supervisor@wms.local | Supervisor123! |
| Warehouse Admin | warehouse@wms.local | Warehouse123! |
| Department User | staff@wms.local | Staff123! |

Seed:
- At least 3 departments.
- At least 10 products.
- Several product categories.
- Initial inventory.
- Sample stock movement history.

Department User should belong to the IT department.

Warehouse Admin should have warehouse-wide access.

Supervisor should have read-only access to reports and inventory.

Never use demo passwords in production.

---

## 13. Documentation — Keep Markdown Minimal

Create only these essential Markdown files:

### README.md
Include:
- Project overview.
- Technology stack.
- Architecture summary.
- Prerequisites.
- Docker startup instructions.
- Environment configuration.
- Migration and seeding instructions.
- Test commands.
- Demo accounts.
- Main application URL.

### docs/IMPLEMENTATION_PLAN.md
Include:
- Implementation phases.
- Task checklist.
- Dependencies between tasks.
- Acceptance criteria.
- Testing requirements.
- Completion status.

### docs/USER_GUIDE.md
Write a practical step-by-step guide for actual users.

Include separate instructions for:

Supervisor:
1. Login.
2. Understand dashboard metrics.
3. View current inventory.
4. View incoming and outgoing stock.
5. Filter reports.
6. Export reports.

Warehouse Admin:
1. Login.
2. Create products.
3. Receive new stock.
4. Review department requests.
5. Fulfill or partially fulfill requests.
6. Reject requests.
7. Process stock returns.
8. Perform stock adjustments.
9. View transaction history.

Department User:
1. Login.
2. Find available products.
3. Submit a stock request.
4. Track request status.
5. Cancel a pending request.
6. Submit a stock return.
7. View department stock usage.

Explain each workflow clearly with numbered steps and expected system results.

Avoid lengthy theoretical documentation.

---

## 14. Implementation Phases

Work in the following sequence.

### Phase 1 — Planning and Design

Before coding:
1. Analyze requirements.
2. Define business workflows.
3. Identify MVP scope and exclusions.
4. Design the database schema.
5. Explain inventory consistency rules.
6. Define API contracts.
7. Design role permissions.
8. Create the implementation plan.

Deliver:
- docs/IMPLEMENTATION_PLAN.md
- Database ERD (Mermaid).
- API endpoint overview.
- Business workflow diagrams (Mermaid).

Do not overdesign.

### Phase 2 — Backend Foundation

Implement:
1. Golang project structure.
2. Configuration.
3. PostgreSQL connection.
4. Database migrations.
5. Authentication.
6. Authorization middleware.
7. Seed data.
8. Health check endpoint.
9. Unit tests.

Verify backend functionality before moving forward.

### Phase 3 — Core Warehouse Features

Implement:
1. Product management.
2. Inventory management.
3. Stock receiving.
4. Department stock requests.
5. Request fulfillment.
6. Stock returns.
7. Stock adjustments.
8. Stock movement ledger.

Prioritize inventory correctness over additional features.

### Phase 4 — Redis and RabbitMQ

Implement:
1. Redis caching.
2. Cache invalidation.
3. RabbitMQ publisher.
4. Background consumer.
5. Bounded Go worker pool.
6. Retry and idempotency handling.
7. Graceful shutdown.

Keep asynchronous functionality independent of critical inventory transactions.

### Phase 5 — Frontend

Implement:
1. Authentication.
2. Role-based layouts.
3. Supervisor dashboard.
4. Warehouse Admin dashboard.
5. Department User dashboard.
6. Product management.
7. Inventory.
8. Receiving.
9. Stock requests.
10. Stock returns.
11. Adjustments.
12. Reports.

Connect every screen to real API endpoints.

### Phase 6 — Integration and Testing

Implement and execute:
1. Backend unit tests.
2. Backend integration tests.
3. Frontend component tests.
4. End-to-end business workflows.
5. Role permission tests.
6. Concurrent stock transaction tests.
7. Redis and RabbitMQ failure tests.

Fix identified issues before proceeding.

### Phase 7 — Docker Deployment and Documentation

Implement:
1. Dockerfiles.
2. Docker Compose.
3. Environment configuration.
4. Health checks.
5. Database migration and seed workflow.
6. README.
7. User guide.
8. Final deployment verification.

---

## 15. MVP Exclusions — Do Not Implement Yet

To prevent scope creep, explicitly exclude:
- Procurement and purchase order management.
- Supplier management module.
- Sales and POS integration.
- Accounting and financial management.
- Barcode and QR scanning.
- RFID integration.
- Mobile applications.
- Advanced warehouse bin optimization.
- Automated replenishment.
- Multi-company support.
- Complex approval hierarchies.
- Forecasting and AI analytics.
- Advanced BI dashboards.
- Microservices.
- Kubernetes.
- Complex event sourcing.
- Multi-tenant architecture.

Keep the architecture extensible enough to add these features later without implementing them now.

---

## 16. Definition of Done

The project is complete only when:

- [x] Frontend and backend run successfully.
- [x] Docker Compose starts all required services.
- [x] Database migrations work.
- [x] Demo users can log in.
- [x] All three roles have correct permissions.
- [x] Warehouse admin can create products.
- [x] Warehouse admin can receive stock.
- [x] Department users can submit requests.
- [x] Warehouse admin can fulfill requests.
- [x] Partial fulfillment works.
- [x] Stock returns work.
- [x] Stock adjustments work.
- [x] Inventory never becomes negative.
- [x] All stock changes are recorded in the ledger.
- [x] Supervisor can view accurate reports.
- [x] Department data is isolated.
- [x] Redis caching and invalidation work.
- [x] RabbitMQ background processing works.
- [x] Go workers shut down gracefully.
- [x] Automated tests pass.
- [x] No critical frontend buttons are non-functional.
- [x] README and USER_GUIDE are complete.
- [x] A clean Docker deployment has been verified.

---

## 17. Instructions to the AI Coding Agent

Follow these rules throughout development:

1. Start with planning, not coding.
2. Inspect the existing repository before creating or modifying files.
3. If the repository is empty, initialize the project structure.
4. Implement one phase at a time.
5. Do not generate the entire project as one enormous code dump.
6. After each phase, run the relevant tests.
7. Fix errors before proceeding.
8. Keep implementation aligned with the documented plan.
9. Update task completion status in IMPLEMENTATION_PLAN.md.
10. Never claim a test passed without actually running it.
11. Never leave critical functionality as TODO or mocked behavior.
12. Prefer straightforward solutions over unnecessary design patterns.
13. Explain important architectural decisions briefly.
14. If a technical choice is uncertain, choose the simplest reliable option and document it.
15. Preserve existing working code unless a change is necessary.

At the end of each phase, provide:
- What was implemented.
- Important files created or changed.
- Commands executed.
- Test results.
- Known limitations.
- Next phase.

**Final priority: Deliver a genuinely working, easy-to-use Warehouse Management System MVP with reliable inventory transactions, clear user workflows, and a maintainable codebase. Simplicity and correctness are more important than feature count or architectural complexity.**