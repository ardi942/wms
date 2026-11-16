package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"wms/backend/internal/httpx"
	"wms/backend/internal/ratelimit"
)

type RateLimitOptions struct {
	Limiter ratelimit.Limiter
	Log     *slog.Logger
	Prefix  string
	Limit   int
	Window  time.Duration
	Key     func(*gin.Context) string
	Skip    func(*gin.Context) bool
}

func RateLimit(opts RateLimitOptions) gin.HandlerFunc {
	window := opts.Window
	if window <= 0 {
		window = time.Minute
	}
	return func(c *gin.Context) {
		if opts.Skip != nil && opts.Skip(c) {
			c.Next()
			return
		}
		key := opts.Prefix + ":" + opts.Key(c)
		decision, err := opts.Limiter.Allow(c.Request.Context(), key, opts.Limit, window)
		if err != nil {
			if opts.Log != nil {
				opts.Log.Warn("rate limiter failed open", "error", err)
			}
			c.Next()
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(decision.Reset.Unix(), 10))
		if decision.Allowed {
			c.Next()
			return
		}
		retry := int(math.Ceil(time.Until(decision.Reset).Seconds()))
		if retry < 1 {
			retry = 1
		}
		c.Header("Retry-After", strconv.Itoa(retry))
		httpx.WriteError(c, httpx.Err(http.StatusTooManyRequests, "RATE_LIMITED", "too many requests"))
	}
}

func RateLimitKeyIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

func RateLimitKeyIdentity(c *gin.Context) string {
	if id := httpx.CurrentIdentity(c); id != nil {
		return "user:" + id.ID.String()
	}
	return RateLimitKeyIP(c)
}

func RateLimitKeyLoginEmail(c *gin.Context) string {
	email := loginEmail(c)
	if email == "" {
		return RateLimitKeyIP(c)
	}
	return "email:" + email
}

func SkipReadOnly(c *gin.Context) bool {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func loginEmail(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var payload struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(payload.Email))
}
