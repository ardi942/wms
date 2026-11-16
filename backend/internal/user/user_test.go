package user

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToDTOImageURL(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	key := "users/image.png"
	updatedAt := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	wantURL := "/users/" + id.String() + "/image"

	t.Run("with image", func(t *testing.T) {
		dto := ToDTO(&User{ID: id, ImageKey: &key, ImageUpdatedAt: &updatedAt})
		if dto.ImageURL == nil || *dto.ImageURL != wantURL {
			t.Fatalf("ImageURL = %v, want %q", dto.ImageURL, wantURL)
		}
		if dto.ImageUpdatedAt == nil || !dto.ImageUpdatedAt.Equal(updatedAt) {
			t.Fatalf("ImageUpdatedAt = %v, want %v", dto.ImageUpdatedAt, updatedAt)
		}
	})

	t.Run("without image", func(t *testing.T) {
		dto := ToDTO(&User{ID: id})
		if dto.ImageURL != nil {
			t.Fatalf("ImageURL = %v, want nil", dto.ImageURL)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		empty := ""
		dto := ToDTO(&User{ID: id, ImageKey: &empty})
		if dto.ImageURL != nil {
			t.Fatalf("ImageURL = %v, want nil", dto.ImageURL)
		}
	})
}
