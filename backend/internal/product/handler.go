package product

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/audit"
	"wms/backend/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) {
	p, perr := httpx.ParsePagination(c)
	if perr != nil {
		httpx.WriteError(c, perr)
		return
	}
	var categoryID *uuid.UUID
	if raw := c.Query("category_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Validation("validation failed", map[string]string{"category_id": "must be a valid UUID"}))
			return
		}
		categoryID = &id
	}
	var active *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Validation("validation failed", map[string]string{"is_active": "must be true or false"}))
			return
		}
		active = &v
	}
	lowStock := c.Query("low_stock") == "true"
	id := httpx.MustIdentity(c)
	items, total, err := h.svc.List(c.Request.Context(), ListFilter{
		Search: c.Query("search"), CategoryID: categoryID, IsActive: active,
		LowStock: lowStock, Sort: c.Query("sort"), Page: p.Page, PageSize: p.PageSize, Role: id.Role,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, httpx.NewMeta(p, total))
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("product not found"))
		return
	}
	dto, serr := h.svc.Get(c.Request.Context(), id, httpx.MustIdentity(c).Role)
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, dto)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("product not found"))
		return
	}
	var in UpdateInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, serr := h.svc.Update(c.Request.Context(), id, in, httpx.MustIdentity(c).Role, audit.ActorFromIdentity(httpx.MustIdentity(c), httpx.RequestID(c)))
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) History(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("product not found"))
		return
	}
	p, perr := httpx.ParsePagination(c)
	if perr != nil {
		httpx.WriteError(c, perr)
		return
	}
	items, total, serr := h.svc.History(c.Request.Context(), id, p.Page, p.PageSize, httpx.MustIdentity(c).Role)
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.List(c, items, httpx.NewMeta(p, total))
}
