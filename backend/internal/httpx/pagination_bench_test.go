package httpx

import "testing"

func BenchmarkParseDate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = parseDate("2026-11-16")
	}
}
