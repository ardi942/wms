package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"wms/backend/internal/httpx"
	"wms/backend/internal/idempotency"
)

const (
	idempotencyKeyKey  = "wms.idempotency_key"
	idempotencyHashKey = "wms.idempotency_hash"
)

func BodyHashAndIdempotencyKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if len(key) < 8 || !validHeaderValue(key, 128) {
			httpx.WriteError(c, httpx.Validation("validation failed", map[string]string{
				"Idempotency-Key": "header is required and must be 8-128 characters using letters, digits, dot, dash, underscore or colon",
			}))
			return
		}
		var body []byte
		if c.Request.Body != nil {
			var err error
			body, err = io.ReadAll(c.Request.Body)
			if err != nil {
				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) {
					httpx.WriteError(c, httpx.Err(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large"))
					return
				}
				httpx.WriteError(c, httpx.Validation("invalid request body", nil))
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		sum := sha256.Sum256(body)
		c.Set(idempotencyKeyKey, key)
		c.Set(idempotencyHashKey, hex.EncodeToString(sum[:]))
		c.Next()
	}
}

func Idempotency(store *idempotency.Store, operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := httpx.CurrentIdentity(c)
		if id == nil {
			httpx.WriteError(c, httpx.Unauthorized("authentication required"))
			return
		}
		key, _ := c.Get(idempotencyKeyKey)
		hash, _ := c.Get(idempotencyHashKey)
		keyStr, _ := key.(string)
		hashStr, _ := hash.(string)
		replay, err := store.Reserve(c.Request.Context(), operation, keyStr, id.ID, hashStr)
		if err != nil {
			httpx.WriteError(c, err)
			return
		}
		if replay != nil {
			c.Data(replay.StatusCode, "application/json; charset=utf-8", replay.Body)
			c.Abort()
			return
		}
		writer := &captureWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()

		status := c.Writer.Status()
		if status >= http.StatusBadRequest {
			_ = store.Release(c.Request.Context(), operation, keyStr)
			return
		}
		if writer.body.Len() == 0 {
			return
		}
		_ = store.Complete(c.Request.Context(), operation, keyStr, status, writer.body.Bytes())
	}
}

type captureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *captureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
