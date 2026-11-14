package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Details    any
}

func (e *AppError) Error() string { return e.Message }

func Err(status int, code, message string) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: message}
}

func Validation(message string, details any) *AppError {
	return &AppError{HTTPStatus: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: message, Details: details}
}

func Unauthorized(message string) *AppError {
	return Err(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(message string) *AppError {
	return Err(http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(message string) *AppError {
	return Err(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(code, message string, details any) *AppError {
	return &AppError{HTTPStatus: http.StatusConflict, Code: code, Message: message, Details: details}
}

func Internal(err error) *AppError {
	return &AppError{HTTPStatus: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "internal server error"}
}

func AsAppError(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return Internal(err)
}

func FieldError(field, message string) *AppError {
	return Validation("validation failed", map[string]string{field: message})
}

func Errorf(format string, args ...any) *AppError {
	return Internal(fmt.Errorf(format, args...))
}
