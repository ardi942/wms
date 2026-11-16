package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	"github.com/google/uuid"
)

const (
	maxDimension = 4096
	maxPixels    = 16_000_000
)

type Image struct {
	Data      []byte
	MIME      string
	Extension string
}

type Service struct {
	store    Store
	maxBytes int64
}

func NewService(store Store, maxBytes int64) *Service {
	return &Service{store: store, maxBytes: maxBytes}
}

func (s *Service) Validate(r io.Reader) (*Image, error) {
	if r == nil {
		return nil, ErrInvalidImage
	}
	if s.maxBytes <= 0 {
		return nil, fmt.Errorf("media: image size limit is not configured")
	}
	data, err := io.ReadAll(io.LimitReader(r, s.maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.maxBytes {
		return nil, ErrTooLarge
	}
	if len(data) == 0 {
		return nil, ErrInvalidImage
	}

	mime, format, ext, ok := imageType(data)
	if !ok {
		return nil, ErrInvalidImage
	}
	cfg, decodedFormat, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if decodedFormat != format {
		return nil, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrInvalidImage
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension {
		return nil, ErrInvalidImage
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, ErrInvalidImage
	}
	return &Image{Data: data, MIME: mime, Extension: ext}, nil
}

func imageType(data []byte) (mime, format, ext string, ok bool) {
	switch http.DetectContentType(data) {
	case "image/jpeg":
		return "image/jpeg", "jpeg", "jpg", true
	case "image/png":
		return "image/png", "png", "png", true
	default:
		return "", "", "", false
	}
}

func NewKey(prefix, ext string) (string, error) {
	if prefix != "products" && prefix != "users" {
		return "", ErrInvalidKey
	}
	if ext != "jpg" && ext != "png" {
		return "", ErrInvalidKey
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return prefix + "/" + id.String() + "." + ext, nil
}

func (s *Service) Put(ctx context.Context, key string, img *Image) error {
	return s.store.Put(ctx, key, bytes.NewReader(img.Data))
}

func (s *Service) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	return s.store.Open(ctx, key)
}

func (s *Service) Delete(ctx context.Context, key string) error {
	return s.store.Delete(ctx, key)
}
