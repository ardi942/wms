package product

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/audit"
	"wms/backend/internal/httpx"
	"wms/backend/internal/media"
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

func (h *Handler) GetImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("product not found"))
		return
	}
	file, serr := h.svc.GetImage(c.Request.Context(), id, httpx.MustIdentity(c).Role)
	if serr != nil {
		httpx.WriteError(c, media.AsAppError(serr))
		return
	}
	defer file.Reader.Close()
	c.Header("Content-Type", file.MIME)
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, "", file.UpdatedAt, file.Reader)
}

func (h *Handler) UploadImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("product not found"))
		return
	}
	header, err := c.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || errors.Is(err, http.ErrNotMultipart) {
			httpx.WriteError(c, httpx.FieldError("image", "is required"))
			return
		}
		httpx.WriteError(c, media.AsAppError(err))
		return
	}
	file, err := header.Open()
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	defer file.Close()
	dto, serr := h.svc.SetImage(c.Request.Context(), id, file, actor(c))
	if serr != nil {
		httpx.WriteError(c, media.AsAppError(serr))
		return
	}
	httpx.OK(c, dto)
}

func (h *Handler) DeleteImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("product not found"))
		return
	}
	if err := h.svc.DeleteImage(c.Request.Context(), id, actor(c)); err != nil {
		httpx.WriteError(c, media.AsAppError(err))
		return
	}
	httpx.NoContent(c)
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

func actor(c *gin.Context) audit.Actor {
	return audit.ActorFromIdentity(httpx.MustIdentity(c), httpx.RequestID(c))
}
