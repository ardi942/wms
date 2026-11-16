package user

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
	var active *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Validation("validation failed", map[string]string{"is_active": "must be true or false"}))
			return
		}
		active = &v
	}
	role, rerr := parseRole(c.Query("role"))
	if rerr != nil {
		httpx.WriteError(c, rerr)
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), ListFilter{
		Search: c.Query("search"), Role: role, IsActive: active, Page: p.Page, PageSize: p.PageSize,
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

func (h *Handler) GetImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("user not found"))
		return
	}
	identity := httpx.MustIdentity(c)
	if identity.Role != RoleWarehouseAdmin && identity.ID != id {
		httpx.WriteError(c, httpx.Forbidden("you cannot view this user image"))
		return
	}
	file, serr := h.svc.GetImage(c.Request.Context(), id)
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
		httpx.WriteError(c, httpx.NotFound("user not found"))
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
		httpx.WriteError(c, httpx.NotFound("user not found"))
		return
	}
	if err := h.svc.DeleteImage(c.Request.Context(), id, actor(c)); err != nil {
		httpx.WriteError(c, media.AsAppError(err))
		return
	}
	httpx.NoContent(c)
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
