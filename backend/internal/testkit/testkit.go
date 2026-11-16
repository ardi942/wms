package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"wms/backend/internal/app"
	"wms/backend/internal/auth"
	"wms/backend/internal/config"
	"wms/backend/internal/database"
	"wms/backend/internal/ratelimit"
	"wms/backend/internal/user"
)

const Password = "Password123!"

type TestUser struct {
	ID           uuid.UUID
	Email        string
	Password     string
	Role         string
	DepartmentID *uuid.UUID
	Active       bool
	Token        string
}

type Env struct {
	T            *testing.T
	DB           *gorm.DB
	Router       *gin.Engine
	App          *app.App
	Config       *config.Config
	Schema       string
	WarehouseID  uuid.UUID
	ITDeptID     uuid.UUID
	FacilitiesID uuid.UUID
	CategoryID   uuid.UUID
	UnitID       uuid.UUID
	ProductID    uuid.UUID
	Product2ID   uuid.UUID
	Users        map[string]TestUser
}

type options struct {
	limiter ratelimit.Limiter
	config  func(*config.Config)
}

type Option func(*options)

func WithLimiter(limiter ratelimit.Limiter) Option {
	return func(o *options) { o.limiter = limiter }
}

func WithConfig(mutate func(*config.Config)) Option {
	return func(o *options) { o.config = mutate }
}

var (
	testDBMu   sync.Mutex
	testDBName string
	baseDBURL  string
)

func New(t *testing.T, opts ...Option) *Env {
	t.Helper()
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	testURL, name := prepareDatabase(t, baseURL)
	db, err := database.Open(testURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	reset(t, db)

	cfg := &config.Config{
		Port:                    "0",
		DatabaseURL:             testURL,
		JWTSecret:               strings.Repeat("test-secret-", 4),
		JWTTTL:                  time.Hour,
		RedisAddr:               "",
		CORSOrigins:             []string{"http://localhost"},
		LogLevel:                "error",
		WorkerConcurrency:       1,
		OutboxPollInterval:      time.Second,
		MigrationsPath:          migrationsPath(),
		MaxRequestBodyBytes:     1 << 20,
		RateLimitEnabled:        false,
		RateLimitLoginPerMinute: 30,
		RateLimitAPIPerMinute:   600,
		RateLimitMutationPerMin: 120,
	}
	if o.config != nil {
		o.config(cfg)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	warehouseID := seed(t, db)
	application := app.New(app.Deps{
		DB:          db,
		Config:      cfg,
		Logger:      log,
		WarehouseID: warehouseID,
		Limiter:     o.limiter,
	})
	t.Cleanup(func() {
		_ = application.Close()
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	e := loadEnv(t, db, application, cfg)
	e.Schema = name
	return e
}

func Cleanup() {
	testDBMu.Lock()
	defer testDBMu.Unlock()
	if testDBName == "" || baseDBURL == "" {
		return
	}
	admin, err := database.Open(baseDBURL)
	if err == nil {
		// terminate connections then drop
		_ = admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`, testDBName).Error
		_ = admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(testDBName))).Error
		sqlDB, _ := admin.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
	testDBName = ""
}

func prepareDatabase(t *testing.T, baseURL string) (string, string) {
	t.Helper()
	testDBMu.Lock()
	defer testDBMu.Unlock()
	if testDBName != "" {
		return withDatabase(baseURL, testDBName), testDBName
	}
	name := fmt.Sprintf("wms_test_%d", os.Getpid())
	admin, err := database.Open(baseURL)
	if err != nil {
		t.Fatalf("open database for test setup: %v", err)
	}
	defer func() {
		sqlDB, _ := admin.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}()
	_ = admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`, name).Error
	_ = admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(name))).Error
	if err := admin.Exec(fmt.Sprintf(`CREATE DATABASE %s`, quoteIdent(name))).Error; err != nil {
		t.Fatalf("create test database: %v", err)
	}
	dsn := withDatabase(baseURL, name)
	if err := database.Migrate(dsn, migrationsPath()); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	testDBName = name
	baseDBURL = baseURL
	return dsn, name
}

func withDatabase(baseURL, dbName string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	u.Path = "/" + dbName
	return u.String()
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func migrationsPath() string {
	if p := os.Getenv("MIGRATIONS_PATH"); p != "" {
		return p
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "./migrations"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))
}

func reset(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"idempotency_keys", "processed_events", "outbox_events", "notifications",
		"stock_movements", "inventory_balances", "stock_return_items", "stock_returns",
		"stock_request_items", "stock_requests", "audit_logs", "products", "users",
		"units", "categories", "departments", "warehouses",
	}
	if err := db.Exec("TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}
}

func seed(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	warehouseID := uuid.Must(uuid.NewV7())
	itDeptID := uuid.Must(uuid.NewV7())
	facilitiesID := uuid.Must(uuid.NewV7())
	categoryID := uuid.Must(uuid.NewV7())
	unitID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	product2ID := uuid.Must(uuid.NewV7())
	hash, err := bcrypt.GenerateFromPassword([]byte(Password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash fixture password: %v", err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if err := db.Exec(query, args...).Error; err != nil {
			t.Fatalf("seed fixture: %v", err)
		}
	}
	exec(`INSERT INTO warehouses (id, code, name) VALUES (?, 'MAIN', 'Main Warehouse')`, warehouseID)
	exec(`INSERT INTO departments (id, code, name) VALUES (?, 'IT', 'Information Technology'), (?, 'FAC', 'Facilities')`, itDeptID, facilitiesID)
	exec(`INSERT INTO categories (id, name) VALUES (?, 'Office Supplies')`, categoryID)
	exec(`INSERT INTO units (id, code, name) VALUES (?, 'PCS', 'Pieces')`, unitID)
	adminID := uuid.Must(uuid.NewV7())
	supervisorID := uuid.Must(uuid.NewV7())
	deptUserID := uuid.Must(uuid.NewV7())
	inactiveID := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO users (id, email, password_hash, full_name, role) VALUES
		(?, 'admin@example.com', ?, 'Admin User', 'WAREHOUSE_ADMIN'),
		(?, 'supervisor@example.com', ?, 'Supervisor User', 'SUPERVISOR'),
		(?, 'inactive@example.com', ?, 'Inactive User', 'SUPERVISOR')`,
		adminID, string(hash), supervisorID, string(hash), inactiveID, string(hash))
	exec(`INSERT INTO users (id, email, password_hash, full_name, role, department_id) VALUES
		(?, 'dept@example.com', ?, 'Department User', 'DEPARTMENT_USER', ?)`,
		deptUserID, string(hash), itDeptID)
	exec(`UPDATE users SET is_active = false WHERE id = ?`, inactiveID)
	exec(`INSERT INTO products (id, sku, name, category_id, unit_id, min_stock) VALUES
		(?, 'SKU-001', 'Notebook', ?, ?, 10),
		(?, 'SKU-002', 'Pen', ?, ?, 20)`, productID, categoryID, unitID, product2ID, categoryID, unitID)
	exec(`INSERT INTO inventory_balances (product_id, warehouse_id, quantity) VALUES (?, ?, 100), (?, ?, 50)`, productID, warehouseID, product2ID, warehouseID)
	return warehouseID
}

func loadEnv(t *testing.T, db *gorm.DB, application *app.App, cfg *config.Config) *Env {
	t.Helper()
	e := &Env{T: t, DB: db, Router: application.Router, App: application, Config: cfg}
	loadID := func(query string, args ...any) uuid.UUID {
		t.Helper()
		var id string
		if err := db.Raw(query, args...).Scan(&id).Error; err != nil {
			t.Fatalf("load fixture: %v", err)
		}
		parsed, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("parse fixture uuid %q: %v", id, err)
		}
		return parsed
	}
	e.WarehouseID = loadID(`SELECT id::text FROM warehouses WHERE code = 'MAIN'`)
	e.ITDeptID = loadID(`SELECT id::text FROM departments WHERE code = 'IT'`)
	e.FacilitiesID = loadID(`SELECT id::text FROM departments WHERE code = 'FAC'`)
	e.CategoryID = loadID(`SELECT id::text FROM categories WHERE name = 'Office Supplies'`)
	e.UnitID = loadID(`SELECT id::text FROM units WHERE code = 'PCS'`)
	e.ProductID = loadID(`SELECT id::text FROM products WHERE sku = 'SKU-001'`)
	e.Product2ID = loadID(`SELECT id::text FROM products WHERE sku = 'SKU-002'`)
	e.Users = map[string]TestUser{}
	jwt := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	var rows []struct {
		ID           string
		Email        string
		Role         string
		DepartmentID *string `gorm:"column:department_id"`
		IsActive     bool
	}
	if err := db.Raw(`SELECT id::text AS id, email, role, department_id::text AS department_id, is_active FROM users ORDER BY email`).Scan(&rows).Error; err != nil {
		t.Fatalf("load users: %v", err)
	}
	for _, row := range rows {
		id := uuid.MustParse(row.ID)
		var deptID *uuid.UUID
		if row.DepartmentID != nil && *row.DepartmentID != "" {
			parsed := uuid.MustParse(*row.DepartmentID)
			deptID = &parsed
		}
		tok, err := jwt.Sign(&user.User{ID: id, Role: row.Role, DepartmentID: deptID})
		if err != nil {
			t.Fatalf("sign fixture token: %v", err)
		}
		key := strings.Split(row.Email, "@")[0]
		e.Users[key] = TestUser{
			ID: id, Email: row.Email, Password: Password, Role: row.Role,
			DepartmentID: deptID, Active: row.IsActive, Token: tok,
		}
	}
	return e
}

func (e *Env) Request(method, path string, body any, token string, headers map[string]string) *httptest.ResponseRecorder {
	e.T.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			e.T.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.Router.ServeHTTP(rec, req)
	return rec
}

func (e *Env) SetInventory(productID uuid.UUID, quantity int) {
	e.T.Helper()
	if err := e.DB.Exec(`INSERT INTO inventory_balances (product_id, warehouse_id, quantity)
		VALUES (?, ?, ?)
		ON CONFLICT (product_id, warehouse_id) DO UPDATE SET quantity = EXCLUDED.quantity`,
		productID, e.WarehouseID, quantity).Error; err != nil {
		e.T.Fatalf("set inventory: %v", err)
	}
}

func (e *Env) Balance(productID uuid.UUID) int {
	e.T.Helper()
	var quantity int
	if err := e.DB.Raw(`SELECT quantity FROM inventory_balances WHERE product_id = ? AND warehouse_id = ?`,
		productID, e.WarehouseID).Scan(&quantity).Error; err != nil {
		e.T.Fatalf("read inventory balance: %v", err)
	}
	return quantity
}

func Decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func ErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode error response %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	return out.Error.Code
}

func RequireStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, rec.Code, rec.Body.String())
	}
}
