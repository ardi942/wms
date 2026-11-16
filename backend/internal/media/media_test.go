package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
)

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{G: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestValidateAcceptsJPEGAndPNG(t *testing.T) {
	svc := NewService(nil, 1<<20)
	for _, tc := range []struct {
		name string
		data []byte
		mime string
		ext  string
	}{
		{name: "jpeg", data: jpegBytes(t), mime: "image/jpeg", ext: "jpg"},
		{name: "png", data: pngBytes(t), mime: "image/png", ext: "png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img, err := svc.Validate(bytes.NewReader(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if img.MIME != tc.mime || img.Extension != tc.ext {
				t.Fatalf("got %s/%s", img.MIME, img.Extension)
			}
		})
	}
}

func gifBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func grayBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestValidateRejectsInvalidImages(t *testing.T) {
	svc := NewService(nil, 1<<20)
	if _, err := svc.Validate(bytes.NewReader([]byte("not an image"))); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected invalid image, got %v", err)
	}
	if _, err := svc.Validate(bytes.NewReader(nil)); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected invalid empty image, got %v", err)
	}
	small := NewService(nil, 100)
	if _, err := small.Validate(bytes.NewReader(jpegBytes(t))); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected too large, got %v", err)
	}
}

func TestValidateRejectsUnsupportedFormats(t *testing.T) {
	svc := NewService(nil, 1<<20)
	webp := []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
	for name, data := range map[string][]byte{
		"svg":  []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`),
		"gif":  gifBytes(t),
		"webp": webp,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Validate(bytes.NewReader(data)); !errors.Is(err, ErrInvalidImage) {
				t.Fatalf("expected invalid image, got %v", err)
			}
		})
	}
}

func TestValidateRejectsCorruptAndMismatchedData(t *testing.T) {
	svc := NewService(nil, 1<<20)
	corruptPNG := append([]byte("\x89PNG\r\n\x1a\n"), []byte("not really a png")...)
	if _, err := svc.Validate(bytes.NewReader(corruptPNG)); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected invalid corrupt png, got %v", err)
	}
	// PNG signature with JPEG payload: detection and decode disagree.
	mismatch := append([]byte("\x89PNG\r\n\x1a\n"), jpegBytes(t)...)
	if _, err := svc.Validate(bytes.NewReader(mismatch)); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected invalid mismatched image, got %v", err)
	}
}

func TestValidateRejectsExcessiveDimensions(t *testing.T) {
	svc := NewService(nil, 4<<20)
	if _, err := svc.Validate(bytes.NewReader(grayBytes(t, 4097, 1))); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected invalid width, got %v", err)
	}
	if _, err := svc.Validate(bytes.NewReader(grayBytes(t, 4096, 4001))); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected invalid pixel count, got %v", err)
	}
}

func TestNewKey(t *testing.T) {
	for _, prefix := range []string{"products", "users"} {
		for _, ext := range []string{"jpg", "png"} {
			key, err := NewKey(prefix, ext)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(key, prefix+"/") || !strings.HasSuffix(key, "."+ext) {
				t.Fatalf("unexpected key %q", key)
			}
		}
	}
	if _, err := NewKey("other", "jpg"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected invalid key, got %v", err)
	}
	if _, err := NewKey("products", "gif"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected invalid extension, got %v", err)
	}
}

func TestLocalStoreCRUDAndTraversal(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "products/image.jpg"
	if err := store.Put(ctx, key, bytes.NewReader(jpegBytes(t))); err != nil {
		t.Fatal(err)
	}
	file, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected stored data")
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	for _, key := range []string{"../escape.jpg", "/etc/passwd", `products\..\escape.jpg`, "."} {
		if err := store.Put(ctx, key, bytes.NewReader([]byte("x"))); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("expected invalid key %q, got %v", key, err)
		}
	}
	if err := store.Delete(ctx, "products/missing.jpg"); err != nil {
		t.Fatalf("delete missing object should be a no-op, got %v", err)
	}
}
