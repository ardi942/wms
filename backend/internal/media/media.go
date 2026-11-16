package media

import (
	"context"
	"errors"
	"io"
	"net/http"

	"wms/backend/internal/httpx"
)

var (
	ErrNotFound     = errors.New("media: object not found")
	ErrInvalidKey   = errors.New("media: invalid storage key")
	ErrTooLarge     = errors.New("media: image is too large")
	ErrInvalidImage = errors.New("media: invalid image")
)

type Store interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

func AsAppError(err error) *httpx.AppError {
	var maxErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxErr), errors.Is(err, ErrTooLarge):
		return httpx.Err(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
	case errors.Is(err, ErrInvalidImage):
		return httpx.FieldError("image", "must be a valid JPEG or PNG image within the allowed size and dimensions")
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("image not found")
	default:
		return httpx.AsAppError(err)
	}
}
