package httpx

import "testing"

func FuzzParseDateRange(f *testing.F) {
	for _, s := range []string{"2026-01-01", "2026-02-31", "bad", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseDate(s)
	})
}
