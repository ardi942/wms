package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func BindJSON(c *gin.Context, dst any) *AppError {
	if err := c.ShouldBindJSON(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return Err(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		}
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			details := map[string]string{}
			for _, fe := range verrs {
				details[fieldPath(fe.Namespace())] = messageFor(fe)
			}
			return Validation("validation failed", details)
		}
		return Validation("invalid JSON body", nil)
	}
	return nil
}

func fieldPath(ns string) string {
	parts := strings.Split(ns, ".")
	if len(parts) > 1 {
		parts = parts[1:]
	}
	return strings.ToLower(strings.Join(parts, "."))
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be >= %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be > %s", fe.Param())
	case "oneof":
		return "must be one of: " + fe.Param()
	case "uuid":
		return "must be a valid UUID"
	default:
		return "is invalid"
	}
}
