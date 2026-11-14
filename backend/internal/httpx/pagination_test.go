package httpx

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePaginationDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/inventory", nil)
	p, err := ParsePagination(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Page != 1 || p.PageSize != 20 {
		t.Fatalf("expected defaults 1/20, got %d/%d", p.Page, p.PageSize)
	}
	if p.Offset() != 0 {
		t.Fatalf("expected offset 0, got %d", p.Offset())
	}
}

func TestParsePaginationRejectsBadValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{"?page=0", "?page=abc", "?page_size=0", "?page_size=101"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/inventory"+query, nil)
		if _, err := ParsePagination(c); err == nil {
			t.Fatalf("expected validation error for %s", query)
		}
	}
}

func TestNewMeta(t *testing.T) {
	m := NewMeta(Pagination{Page: 2, PageSize: 20}, 41)
	if m.TotalPages != 3 || m.Total != 41 {
		t.Fatalf("unexpected meta: %+v", m)
	}
}

func TestParseDateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/reports?date_from=2026-01-01&date_to=2026-02-01", nil)
	from, to, err := ParseDateRange(c)
	if err != nil || from != "2026-01-01" || to != "2026-02-01" {
		t.Fatalf("unexpected range %q %q err=%v", from, to, err)
	}
}
