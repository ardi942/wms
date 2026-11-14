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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/auth"
	"wms/backend/internal/cache"
	"wms/backend/internal/category"
	"wms/backend/internal/config"
	"wms/backend/internal/database"
	"wms/backend/internal/department"
	"wms/backend/internal/httpx"
	"wms/backend/internal/idempotency"
	"wms/backend/internal/inventory"
	"wms/backend/internal/middleware"
	"wms/backend/internal/notification"
	"wms/backend/internal/product"
	"wms/backend/internal/report"
	"wms/backend/internal/stockrequest"
	"wms/backend/internal/stockreturn"
	"wms/backend/internal/unit"
	"wms/backend/internal/user"
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

	userRepo := user.NewRepository(db)
	jwtSvc := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := auth.NewService(userRepo, jwtSvc)
	authHandler := auth.NewHandler(authSvc)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)
	deptRepo := department.NewRepository(db)
	deptHandler := department.NewHandler(deptRepo)
	categoryRepo := category.NewRepository(db)
	categoryHandler := category.NewHandler(categoryRepo)
	unitRepo := unit.NewRepository(db)
	unitHandler := unit.NewHandler(unitRepo)
	productRepo := product.NewRepository(db)
	productSvc := product.NewService(productRepo, warehouseID)
	productSvc.SetCache(redisCache)
	productHandler := product.NewHandler(productSvc)
	invRepo := inventory.NewRepository(db, warehouseID)
	invSvc := inventory.NewService(db, invRepo)
	invSvc.SetCache(redisCache)
	invHandler := inventory.NewHandler(invSvc)
	reqRepo := stockrequest.NewRepository(db)
	reqSvc := stockrequest.NewService(db, reqRepo, invRepo)
	reqSvc.SetCache(redisCache)
	reqHandler := stockrequest.NewHandler(reqSvc)
	retRepo := stockreturn.NewRepository(db)
	retSvc := stockreturn.NewService(db, retRepo, invRepo)
	retSvc.SetCache(redisCache)
	retHandler := stockreturn.NewHandler(retSvc)
	reportSvc := report.NewService(db, invRepo, warehouseID)
	reportSvc.SetCache(redisCache)
	reportHandler := report.NewHandler(reportSvc)
	notifRepo := notification.NewRepository(db)
	notifHandler := notification.NewHandler(notifRepo)
	idemStore := idempotency.NewStore(db)
	go runIdempotencyCleanup(idemStore, log)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.Logging(log),
		middleware.CORS(cfg.CORSOrigins),
	)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		checks := gin.H{}
		status := http.StatusOK
		overall := "ok"
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			checks["database"] = "unavailable"
			status = http.StatusServiceUnavailable
			overall = "unavailable"
		} else {
			checks["database"] = "ok"
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := redisCache.Ping(ctx); err != nil {
			checks["redis"] = "unavailable"
			if overall == "ok" {
				overall = "degraded"
			}
		} else {
			checks["redis"] = "ok"
		}
		checks["rabbitmq"] = "optional"
		c.JSON(status, gin.H{"status": overall, "checks": checks})
	})

	api := router.Group("/api/v1")
	api.POST("/auth/login", authHandler.Login)

	authed := api.Group("", middleware.Auth(jwtSvc, userRepo))
	authed.GET("/auth/me", authHandler.Me)
	authed.GET("/departments", deptHandler.List)
	authed.GET("/categories", categoryHandler.List)
	authed.GET("/units", unitHandler.List)
	authed.GET("/products", productHandler.List)
	authed.GET("/products/:id", productHandler.Get)
	authed.GET("/inventory", invHandler.List)
	authed.GET("/inventory/movements", invHandler.Movements)
	authed.GET("/inventory/:productId", invHandler.Get)
	authed.GET("/stock-requests", reqHandler.List)
	authed.GET("/stock-requests/:id", reqHandler.Get)
	authed.GET("/stock-returns", retHandler.List)
	authed.GET("/stock-returns/:id", retHandler.Get)
	authed.GET("/reports/dashboard", reportHandler.Dashboard)
	authed.GET("/reports/department-usage", reportHandler.DepartmentUsage)
	authed.GET("/reports/export", reportHandler.Export)
	authed.GET("/notifications", notifHandler.List)
	authed.GET("/notifications/unread-count", notifHandler.UnreadCount)
	authed.PATCH("/notifications/:id/read", notifHandler.MarkRead)
	authed.POST("/notifications/read-all", notifHandler.MarkAllRead)

	admin := authed.Group("", middleware.RequireRoles(user.RoleWarehouseAdmin))
	admin.GET("/users", userHandler.List)
	admin.POST("/users", userHandler.Create)
	admin.GET("/users/:id", userHandler.Get)
	admin.PATCH("/users/:id", userHandler.Update)
	admin.POST("/categories", categoryHandler.Create)
	admin.PATCH("/categories/:id", categoryHandler.Update)
	admin.POST("/units", unitHandler.Create)
	admin.PATCH("/units/:id", unitHandler.Update)
	admin.POST("/products", productHandler.Create)
	admin.POST("/inventory/receipts",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "inventory.receipt.create"), invHandler.Receive)
	admin.POST("/inventory/adjustments",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "inventory.adjustment.create"), invHandler.Adjust)
	admin.POST("/stock-requests/:id/fulfill",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockrequest.fulfill"), reqHandler.Fulfill)
	admin.POST("/stock-requests/:id/reject",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockrequest.reject"), reqHandler.Reject)
	admin.POST("/stock-returns/:id/confirm",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockreturn.confirm"), retHandler.Confirm)
	admin.POST("/stock-returns/:id/reject",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockreturn.reject"), retHandler.Reject)

	productEditors := authed.Group("", middleware.RequireRoles(user.RoleSupervisor, user.RoleWarehouseAdmin))
	productEditors.PATCH("/products/:id", productHandler.Update)

	reporters := authed.Group("", middleware.RequireRoles(user.RoleSupervisor, user.RoleWarehouseAdmin))
	reporters.GET("/reports/inventory", reportHandler.Inventory)
	reporters.GET("/reports/stock-movements", reportHandler.Movements)
	reporters.GET("/reports/low-stock", reportHandler.LowStock)

	dept := authed.Group("", middleware.RequireRoles(user.RoleDepartmentUser))
	dept.POST("/stock-requests",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockrequest.create"), reqHandler.Create)
	dept.POST("/stock-requests/:id/cancel",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockrequest.cancel"), reqHandler.Cancel)
	dept.POST("/stock-returns",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockreturn.create"), retHandler.Create)
	dept.POST("/stock-returns/:id/cancel",
		middleware.BodyHashAndIdempotencyKey(), middleware.Idempotency(idemStore, "stockreturn.cancel"), retHandler.Cancel)

	router.NoRoute(func(c *gin.Context) {
		httpx.WriteError(c, httpx.NotFound("route not found"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
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
