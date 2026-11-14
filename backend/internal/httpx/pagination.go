package httpx

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page     int
	PageSize int
}

type Meta struct {
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func ParsePagination(c *gin.Context) (Pagination, *AppError) {
	p := Pagination{Page: 1, PageSize: 20}
	if raw := c.Query("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return p, Validation("validation failed", map[string]string{"page": "must be a positive integer"})
		}
		p.Page = n
	}
	if raw := c.Query("page_size"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return p, Validation("validation failed", map[string]string{"page_size": "must be between 1 and 100"})
		}
		p.PageSize = n
	}
	return p, nil
}

func (p Pagination) Offset() int { return (p.Page - 1) * p.PageSize }

func NewMeta(p Pagination, total int64) Meta {
	pages := 0
	if total > 0 {
		pages = int((total + int64(p.PageSize) - 1) / int64(p.PageSize))
	}
	return Meta{Page: p.Page, PageSize: p.PageSize, Total: total, TotalPages: pages}
}

func ParseDateRange(c *gin.Context) (from, to string, err *AppError) {
	from = c.Query("date_from")
	to = c.Query("date_to")
	for _, v := range []string{from, to} {
		if v == "" {
			continue
		}
		if len(v) != 10 || strings.Count(v, "-") != 2 {
			return "", "", Validation("validation failed", map[string]string{"date": "expected YYYY-MM-DD"})
		}
		if _, e := parseDate(v); e != nil {
			return "", "", Validation("validation failed", map[string]string{"date": "expected YYYY-MM-DD"})
		}
	}
	return from, to, nil
}

func parseDate(v string) (int64, error) {
	var y, m, d int
	if _, err := fmt.Sscanf(v, "%d-%d-%d", &y, &m, &d); err != nil {
		return 0, err
	}
	if m < 1 || m > 12 || d < 1 || d > 31 || y < 1900 {
		return 0, fmt.Errorf("invalid date")
	}
	return int64(y*10000 + m*100 + d), nil
}
