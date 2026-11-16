package httpx

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page     int
	PageSize int
}

type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
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
	var fromN, toN int64
	for _, pair := range []struct {
		name  string
		value string
		out   *int64
	}{{"date_from", from, &fromN}, {"date_to", to, &toN}} {
		if pair.value == "" {
			continue
		}
		if len(pair.value) != 10 || strings.Count(pair.value, "-") != 2 {
			return "", "", Validation("validation failed", map[string]string{"date": "expected YYYY-MM-DD"})
		}
		n, e := parseDate(pair.value)
		if e != nil {
			return "", "", Validation("validation failed", map[string]string{"date": "expected YYYY-MM-DD"})
		}
		*pair.out = n
	}
	if from != "" && to != "" && fromN > toN {
		return "", "", Validation("validation failed", map[string]string{"date": "date_from must be before or equal to date_to"})
	}
	return from, to, nil
}

func parseDate(v string) (int64, error) {
	parsed, err := time.Parse("2006-01-02", v)
	if err != nil {
		return 0, err
	}
	if parsed.Format("2006-01-02") != v {
		return 0, strconv.ErrSyntax
	}
	if parsed.Year() < 1900 {
		return 0, strconv.ErrSyntax
	}
	return int64(parsed.Year()*10000 + int(parsed.Month())*100 + parsed.Day()), nil
}
