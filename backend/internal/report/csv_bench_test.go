package report

import "testing"

func BenchmarkSanitizeCSVField(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = sanitizeCSVField("=SUM(A1:A2)")
	}
}
