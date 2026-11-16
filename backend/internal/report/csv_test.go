package report

import "testing"

func TestSanitizeCSVField(t *testing.T) {
	cases := map[string]string{
		"=SUM(A1)": "'=SUM(A1)",
		"+SUM(A1)": "'+SUM(A1)",
		"@SUM(A1)": "'@SUM(A1)",
		"-SUM(A1)": "'-SUM(A1)",
		"\tcmd":    "'\tcmd",
		"-5":       "-5",
		"-12.50":   "-12.50",
		" normal":  " normal",
		"Product":  "Product",
		"":         "",
	}
	for in, want := range cases {
		if got := sanitizeCSVField(in); got != want {
			t.Fatalf("sanitizeCSVField(%q)=%q want %q", in, got, want)
		}
	}
}
