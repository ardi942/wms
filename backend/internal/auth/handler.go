package auth

import (
	"github.com/gin-gonic/gin"
	"wms/backend/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) Me(c *gin.Context) {
	id := httpx.MustIdentity(c)
	u, err := h.svc.CurrentUser(c.Request.Context(), id.ID.String())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"user": u})
}
