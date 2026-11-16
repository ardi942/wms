package stockrequest

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/httpx"
	"wms/backend/internal/user"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) {
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
	identity := httpx.MustIdentity(c)
	if identity.Role == user.RoleDepartmentUser {
		if identity.DepartmentID == nil {
			httpx.WriteError(c, httpx.Forbidden("department user has no department"))
			return
		}
		departmentID = identity.DepartmentID
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
	for i := range items {
		items[i].AllowedActions = allowedActions(items[i].Status, identity.Role, identity.DepartmentID, items[i].Department.ID)
	}
	httpx.List(c, items, httpx.NewMeta(p, total))
}

func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, err := h.svc.Create(c.Request.Context(), in, httpx.MustIdentity(c))
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, dto)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("stock request not found"))
		return
	}
	dto, serr := h.svc.Get(c.Request.Context(), id, httpx.MustIdentity(c))
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) Fulfill(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("stock request not found"))
		return
	}
	var in FulfillInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, serr := h.svc.Fulfill(c.Request.Context(), id, in, httpx.MustIdentity(c))
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("stock request not found"))
		return
	}
	var in RejectInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, serr := h.svc.Reject(c.Request.Context(), id, in.Reason, httpx.MustIdentity(c))
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("stock request not found"))
		return
	}
	dto, serr := h.svc.Cancel(c.Request.Context(), id, httpx.MustIdentity(c))
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}
