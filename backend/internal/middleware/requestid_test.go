package middleware

import "testing"

func TestValidHeaderValue(t *testing.T) {
	valid := []string{"abc-123_DEF:456.xyz", "a2345678", "0123456789abcdef", "x"}
	for _, v := range valid {
		if !validHeaderValue(v, 128) {
			t.Fatalf("expected %q valid", v)
		}
	}
	invalid := []string{"", "bad space", "bad\nline", "<script>", "bad/value"}
	for _, v := range invalid {
		if validHeaderValue(v, 128) {
			t.Fatalf("expected %q invalid", v)
		}
	}
	if validHeaderValue("0123456789", 8) {
		t.Fatal("expected over-length value invalid")
	}
}
