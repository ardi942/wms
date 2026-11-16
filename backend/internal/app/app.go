package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/audit"
	"wms/backend/internal/auth"
	"wms/backend/internal/cache"
	"wms/backend/internal/category"
	"wms/backend/internal/config"
	"wms/backend/internal/department"
	"wms/backend/internal/httpx"
	"wms/backend/internal/idempotency"
	"wms/backend/internal/inventory"
	"wms/backend/internal/media"
	"wms/backend/internal/middleware"
	"wms/backend/internal/notification"
	"wms/backend/internal/product"
	"wms/backend/internal/ratelimit"
	"wms/backend/internal/report"
	"wms/backend/internal/stockrequest"
	"wms/backend/internal/stockreturn"
	"wms/backend/internal/unit"
	"wms/backend/internal/user"
)

type Deps struct {
	DB          *gorm.DB
	Cache       *cache.Cache
	Config      *config.Config
	Logger      *slog.Logger
	WarehouseID uuid.UUID
	Limiter     ratelimit.Limiter
	Media       *media.Service
}

type App struct {
	Router      *gin.Engine
	Idempotency *idempotency.Store
	Limiter     ratelimit.Limiter
}

func New(deps Deps) (*App, error) {
	cfg := deps.Config
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	mediaSvc := deps.Media
	if mediaSvc == nil {
		store, err := media.NewLocalStore(cfg.UploadDir)
		if err != nil {
			return nil, err
		}
		mediaSvc = media.NewService(store, cfg.ImageMaxBytes)
	}
	limiter := deps.Limiter
	if limiter == nil && cfg.RateLimitEnabled {
		if redisLimiter, err := ratelimit.NewRedis(cfg.RedisAddr, cfg.RedisPassword); err == nil {
			limiter = &ratelimit.Fallback{Primary: redisLimiter, Backup: ratelimit.NewMemory(), Log: log}
		} else {
			log.Warn("redis rate limiter unavailable; using in-memory limiter", "error", err)
			limiter = ratelimit.NewMemory()
		}
	}

	userRepo := user.NewRepository(deps.DB)
	jwtSvc := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := auth.NewService(userRepo, jwtSvc)
	authHandler := auth.NewHandler(authSvc)
	auditRepo := audit.NewRepository(deps.DB)
	auditSvc := audit.NewService(auditRepo)
	auditHandler := audit.NewHandler(auditSvc)
	userSvc := user.NewService(deps.DB, userRepo, auditSvc, mediaSvc)
	userHandler := user.NewHandler(userSvc)
	deptRepo := department.NewRepository(deps.DB)
	deptHandler := department.NewHandler(deptRepo)
	categoryRepo := category.NewRepository(deps.DB)
	categoryHandler := category.NewHandler(categoryRepo)
	unitRepo := unit.NewRepository(deps.DB)
	unitHandler := unit.NewHandler(unitRepo)
	productRepo := product.NewRepository(deps.DB)
	productSvc := product.NewService(deps.DB, productRepo, deps.WarehouseID, auditSvc, mediaSvc)
	productSvc.SetCache(deps.Cache)
	productHandler := product.NewHandler(productSvc)
	invRepo := inventory.NewRepository(deps.DB, deps.WarehouseID)
	invSvc := inventory.NewService(deps.DB, invRepo)
	invSvc.SetCache(deps.Cache)
	invHandler := inventory.NewHandler(invSvc)
	reqRepo := stockrequest.NewRepository(deps.DB)
	reqSvc := stockrequest.NewService(deps.DB, reqRepo, invRepo)
	reqSvc.SetCache(deps.Cache)
	reqHandler := stockrequest.NewHandler(reqSvc)
	retRepo := stockreturn.NewRepository(deps.DB)
	retSvc := stockreturn.NewService(deps.DB, retRepo, invRepo)
	retSvc.SetCache(deps.Cache)
	retHandler := stockreturn.NewHandler(retSvc)
	reportSvc := report.NewService(deps.DB, invRepo, deps.WarehouseID)
	reportSvc.SetCache(deps.Cache)
	reportHandler := report.NewHandler(reportSvc)
	notifRepo := notification.NewRepository(deps.DB)
	notifHandler := notification.NewHandler(notifRepo)
	idemStore := idempotency.NewStore(deps.DB)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if len(cfg.TrustedProxies) == 0 {
		_ = router.SetTrustedProxies(nil)
	} else if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Warn("invalid trusted proxies; trusting none", "error", err)
		_ = router.SetTrustedProxies(nil)
	}
	router.HandleMethodNotAllowed = true
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.BodyLimitResolver(cfg.MaxRequestBodyBytes, cfg.ImageMaxBytes+(512<<10)),
		middleware.SecurityHeaders(),
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
		sqlDB, err := deps.DB.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			checks["database"] = "unavailable"
			status = http.StatusServiceUnavailable
			overall = "unavailable"
		} else {
			checks["database"] = "ok"
		}
		if deps.Cache == nil {
			checks["redis"] = "unavailable"
			if overall == "ok" {
				overall = "degraded"
			}
		} else {
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			if err := deps.Cache.Ping(ctx); err != nil {
				checks["redis"] = "unavailable"
				if overall == "ok" {
					overall = "degraded"
				}
			} else {
				checks["redis"] = "ok"
			}
		}
		checks["rabbitmq"] = "optional"
		c.JSON(status, gin.H{"status": overall, "checks": checks})
	})

	api := router.Group("/api/v1")
	loginHandlers := []gin.HandlerFunc{}
	if limiter != nil {
		loginHandlers = append(loginHandlers,
			middleware.RateLimit(middleware.RateLimitOptions{
				Limiter: limiter, Log: log, Prefix: "login-ip",
				Limit: cfg.RateLimitLoginPerMinute, Window: time.Minute, Key: middleware.RateLimitKeyIP,
			}),
			middleware.RateLimit(middleware.RateLimitOptions{
				Limiter: limiter, Log: log, Prefix: "login-email",
				Limit: cfg.RateLimitLoginPerMinute, Window: time.Minute, Key: middleware.RateLimitKeyLoginEmail,
			}),
		)
	}
	loginHandlers = append(loginHandlers, authHandler.Login)
	api.POST("/auth/login", loginHandlers...)

	authedHandlers := []gin.HandlerFunc{middleware.Auth(jwtSvc, userRepo)}
	if limiter != nil {
		authedHandlers = append(authedHandlers,
			middleware.RateLimit(middleware.RateLimitOptions{
				Limiter: limiter, Log: log, Prefix: "api",
				Limit: cfg.RateLimitAPIPerMinute, Window: time.Minute, Key: middleware.RateLimitKeyIdentity,
			}),
			middleware.RateLimit(middleware.RateLimitOptions{
				Limiter: limiter, Log: log, Prefix: "mutation",
				Limit: cfg.RateLimitMutationPerMin, Window: time.Minute, Key: middleware.RateLimitKeyIdentity,
				Skip: middleware.SkipReadOnly,
			}),
		)
	}
	authed := api.Group("", authedHandlers...)
	authed.GET("/auth/me", authHandler.Me)
	authed.GET("/departments", deptHandler.List)
	authed.GET("/categories", categoryHandler.List)
	authed.GET("/units", unitHandler.List)
	authed.GET("/products", productHandler.List)
	authed.GET("/products/:id", productHandler.Get)
	authed.GET("/products/:id/image", productHandler.GetImage)
	authed.GET("/users/:id/image", userHandler.GetImage)
	authed.GET("/inventory", invHandler.List)
	authed.GET("/inventory/movements", invHandler.Movements)
	authed.GET("/inventory/:productId", invHandler.Get)
	authed.GET("/stock-requests", reqHandler.List)
	authed.GET("/stock-requests/:id", reqHandler.Get)
	authed.GET("/stock-returns", retHandler.List)
	authed.GET("/stock-returns/:id", retHandler.Get)
	authed.GET("/monitoring/requests", reqHandler.MonitorList)
	authed.GET("/monitoring/returns", retHandler.MonitorList)
	authed.GET("/reports/dashboard", reportHandler.Dashboard)
	authed.GET("/reports/dashboard/recent-movements", reportHandler.RecentMovements)
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
	admin.POST("/users/:id/image", userHandler.UploadImage)
	admin.DELETE("/users/:id/image", userHandler.DeleteImage)
	admin.POST("/users/:id/activate", userHandler.Activate)
	admin.POST("/users/:id/deactivate", userHandler.Deactivate)
	admin.POST("/categories", categoryHandler.Create)
	admin.PATCH("/categories/:id", categoryHandler.Update)
	admin.POST("/units", unitHandler.Create)
	admin.PATCH("/units/:id", unitHandler.Update)
	admin.POST("/products", productHandler.Create)
	admin.POST("/products/:id/image", productHandler.UploadImage)
	admin.DELETE("/products/:id/image", productHandler.DeleteImage)
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

	history := authed.Group("", middleware.RequireRoles(user.RoleSupervisor, user.RoleWarehouseAdmin))
	history.GET("/audit", auditHandler.List)
	history.GET("/users/:id/history", userHandler.History)
	history.GET("/products/:id/history", productHandler.History)

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
	router.NoMethod(func(c *gin.Context) {
		httpx.WriteError(c, httpx.Err(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed"))
	})

	return &App{Router: router, Idempotency: idemStore, Limiter: limiter}, nil
}

func (a *App) Close() error {
	if a.Limiter != nil {
		return a.Limiter.Close()
	}
	return nil
}
