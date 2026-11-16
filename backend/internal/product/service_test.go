package product

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToDTOImageURL(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	key := "products/image.jpg"
	updatedAt := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	wantURL := "/products/" + id.String() + "/image"

	t.Run("with image", func(t *testing.T) {
		dto := toDTO(&row{ID: id, ImageKey: &key, ImageUpdatedAt: &updatedAt})
		if dto.ImageURL == nil || *dto.ImageURL != wantURL {
			t.Fatalf("ImageURL = %v, want %q", dto.ImageURL, wantURL)
		}
		if dto.ImageUpdatedAt == nil || !dto.ImageUpdatedAt.Equal(updatedAt) {
			t.Fatalf("ImageUpdatedAt = %v, want %v", dto.ImageUpdatedAt, updatedAt)
		}
	})

	t.Run("without image", func(t *testing.T) {
		dto := toDTO(&row{ID: id})
		if dto.ImageURL != nil {
			t.Fatalf("ImageURL = %v, want nil", dto.ImageURL)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		empty := ""
		dto := toDTO(&row{ID: id, ImageKey: &empty})
		if dto.ImageURL != nil {
			t.Fatalf("ImageURL = %v, want nil", dto.ImageURL)
		}
	})
}
