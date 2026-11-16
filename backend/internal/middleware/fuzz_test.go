package middleware

import "testing"

func FuzzValidHeaderValue(f *testing.F) {
	for _, s := range []string{"abc-123", "bad value", "<x>", "Idempotency-Key-01"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = validHeaderValue(s, 128)
	})
}
