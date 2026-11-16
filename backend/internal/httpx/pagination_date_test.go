package httpx

import "testing"

func TestParseDateInvalidCalendarDates(t *testing.T) {
	for _, value := range []string{"2026-02-31", "2026-13-01", "2026-00-10", "2026-04-31", "1899-12-31", "abc"} {
		if _, err := parseDate(value); err == nil {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
	if _, err := parseDate("2026-02-28"); err != nil {
		t.Fatalf("expected 2026-02-28 valid: %v", err)
	}
	if _, err := parseDate("2024-02-29"); err != nil {
		t.Fatalf("expected leap day valid: %v", err)
	}
}
