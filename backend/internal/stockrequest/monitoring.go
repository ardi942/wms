package stockrequest

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/httpx"
)

type MonitoringRequestRow struct {
	ID             uuid.UUID     `json:"id"`
	RequestNumber  string        `json:"request_number"`
	Status         string        `json:"status"`
	Department     DepartmentRef `json:"department"`
	RequestedBy    ActorRef      `json:"requested_by"`
	ItemCount      int           `json:"item_count"`
	TotalRequested int           `json:"total_requested"`
	TotalIssued    int           `json:"total_issued"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func toMonitoringRequestRow(row ListDTO) MonitoringRequestRow {
	return MonitoringRequestRow{
		ID:             row.ID,
		RequestNumber:  row.RequestNumber,
		Status:         row.Status,
		Department:     row.Department,
		RequestedBy:    row.RequestedBy,
		ItemCount:      row.ItemCount,
		TotalRequested: row.TotalRequested,
		TotalIssued:    row.TotalIssued,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (h *Handler) MonitorList(c *gin.Context) {
	p, perr := httpx.ParsePagination(c)
	if perr != nil {
		httpx.WriteError(c, perr)
		return
	}
	var departmentID *uuid.UUID
	if raw := c.Query("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Validation("validation failed", map[string]string{"department_id": "must be a valid UUID"}))
			return
		}
		departmentID = &id
	}
	from, to, derr := httpx.ParseDateRange(c)
	if derr != nil {
		httpx.WriteError(c, derr)
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), Filter{
		Status: c.Query("status"), DepartmentID: departmentID,
		DateFrom: from, DateTo: to, Search: c.Query("search"),
		Page: p.Page, PageSize: p.PageSize,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	rows := make([]MonitoringRequestRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, toMonitoringRequestRow(item))
	}
	c.Header("Cache-Control", "no-store")
	httpx.List(c, rows, httpx.NewMeta(p, total))
}
