package audit

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wms/backend/internal/httpx"
)

const roleSupervisor = "SUPERVISOR"

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) {
	p, perr := httpx.ParsePagination(c)
	if perr != nil {
		httpx.WriteError(c, perr)
		return
	}
	filter, ferr := parseListFilter(c)
	if ferr != nil {
		httpx.WriteError(c, ferr)
		return
	}
	filter.Page = p.Page
	filter.PageSize = p.PageSize
	identity := httpx.MustIdentity(c)
	items, total, err := h.svc.List(c.Request.Context(), filter, identity.Role == roleSupervisor)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, httpx.NewMeta(p, total))
}

func parseListFilter(c *gin.Context) (ListFilter, *httpx.AppError) {
	var filter ListFilter
	if raw := strings.TrimSpace(c.Query("entity_type")); raw != "" {
		entityType := strings.ToUpper(raw)
		if !ValidEntityType(entityType) {
			return filter, httpx.Validation("validation failed", map[string]string{"entity_type": "is invalid"})
		}
		filter.EntityType = entityType
	}
	if raw := strings.TrimSpace(c.Query("entity_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return filter, httpx.Validation("validation failed", map[string]string{"entity_id": "must be a valid UUID"})
		}
		filter.EntityID = &id
	}
	if raw := strings.TrimSpace(c.Query("actor_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return filter, httpx.Validation("validation failed", map[string]string{"actor_id": "must be a valid UUID"})
		}
		filter.ActorID = &id
	}
	actions := parseActions(c)
	for _, action := range actions {
		if !ValidAction(action) {
			return filter, httpx.Validation("validation failed", map[string]string{"action": "is invalid"})
		}
	}
	filter.Actions = actions
	filter.ActorSearch = strings.TrimSpace(c.Query("actor"))
	from, ferr := parseTimeBound(c.Query("from"), false)
	if ferr != nil {
		return filter, ferr
	}
	to, terr := parseTimeBound(c.Query("to"), true)
	if terr != nil {
		return filter, terr
	}
	filter.From = from
	filter.To = to
	return filter, nil
}

func parseActions(c *gin.Context) []string {
	raw := append([]string{}, c.QueryArray("action")...)
	if single := c.Query("action"); single != "" && len(raw) == 0 {
		raw = append(raw, single)
	}
	seen := map[string]bool{}
	actions := make([]string, 0, len(raw))
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			action := strings.ToUpper(strings.TrimSpace(part))
			if action == "" || seen[action] {
				continue
			}
			seen[action] = true
			actions = append(actions, action)
		}
	}
	return actions
}

func parseTimeBound(raw string, endOfDay bool) (*time.Time, *httpx.AppError) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if len(value) == 10 {
		parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
		if err != nil {
			return nil, httpx.Validation("validation failed", map[string]string{"date": "expected YYYY-MM-DD or RFC3339"})
		}
		if endOfDay {
			parsed = parsed.Add(24 * time.Hour)
		}
		return &parsed, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, httpx.Validation("validation failed", map[string]string{"date": "expected YYYY-MM-DD or RFC3339"})
	}
	return &parsed, nil
}
