package report

import "testing"

func FuzzCSVSanitize(f *testing.F) {
	for _, s := range []string{"=1+1", "-5", "normal", "@cmd", "+cmd", "\tfoo"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = sanitizeCSVField(s)
	})
}
