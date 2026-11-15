package stockreturn

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/httpx"
)

type MonitoringReturnRow struct {
	ID             uuid.UUID     `json:"id"`
	ReturnNumber   string        `json:"return_number"`
	Status         string        `json:"status"`
	Department     DepartmentRef `json:"department"`
	RequestedBy    ActorRef      `json:"requested_by"`
	ItemCount      int           `json:"item_count"`
	TotalRequested int           `json:"total_requested"`
	TotalAccepted  int           `json:"total_accepted"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func toMonitoringReturnRow(row ListDTO) MonitoringReturnRow {
	return MonitoringReturnRow{
		ID:             row.ID,
		ReturnNumber:   row.ReturnNumber,
		Status:         row.Status,
		Department:     row.Department,
		RequestedBy:    row.RequestedBy,
		ItemCount:      row.ItemCount,
		TotalRequested: row.TotalRequested,
		TotalAccepted:  row.TotalAccepted,
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
	rows := make([]MonitoringReturnRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, toMonitoringReturnRow(item))
	}
	c.Header("Cache-Control", "no-store")
	httpx.List(c, rows, httpx.NewMeta(p, total))
}
