package user

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
	var active *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Validation("validation failed", map[string]string{"is_active": "must be true or false"}))
			return
		}
		active = &v
	}
	items, total, err := h.svc.List(c.Request.Context(), ListFilter{
		Search: c.Query("search"), IsActive: active, Page: p.Page, PageSize: p.PageSize,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, httpx.NewMeta(p, total))
}

func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, err := h.svc.Create(c.Request.Context(), in, actor(c))
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, dto)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("user not found"))
		return
	}
	var in UpdateInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	dto, uerr := h.svc.Update(c.Request.Context(), id, in, actor(c))
	if uerr != nil {
		httpx.WriteError(c, uerr)
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) Activate(c *gin.Context) {
	h.setActive(c, true)
}

func (h *Handler) Deactivate(c *gin.Context) {
	h.setActive(c, false)
}

func (h *Handler) setActive(c *gin.Context, active bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("user not found"))
		return
	}
	var in StatusInput
	if err := httpx.BindJSON(c, &in); err != nil {
		httpx.WriteError(c, err)
		return
	}
	var dto *DTO
	var changed bool
	if active {
		dto, changed, err = h.svc.Activate(c.Request.Context(), id, in.Reason, actor(c))
	} else {
		dto, changed, err = h.svc.Deactivate(c.Request.Context(), id, in.Reason, actor(c))
	}
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OKWithMeta(c, dto, gin.H{"changed": changed})
}

func (h *Handler) History(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("user not found"))
		return
	}
	p, perr := httpx.ParsePagination(c)
	if perr != nil {
		httpx.WriteError(c, perr)
		return
	}
	items, total, herr := h.svc.History(c.Request.Context(), id, p.Page, p.PageSize)
	if herr != nil {
		httpx.WriteError(c, herr)
		return
	}
	httpx.List(c, items, httpx.NewMeta(p, total))
}

func actor(c *gin.Context) audit.Actor {
	return audit.ActorFromIdentity(httpx.MustIdentity(c), httpx.RequestID(c))
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("user not found"))
		return
	}
	dto, serr := h.svc.Get(c.Request.Context(), id)
	if serr != nil {
		httpx.WriteError(c, serr)
		return
	}
	httpx.OK(c, dto)
}
