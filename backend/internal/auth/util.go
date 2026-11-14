package auth

import (
	"strings"

	"github.com/google/uuid"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func parseUUID(v string) (uuid.UUID, error) {
	return uuid.Parse(v)
}
