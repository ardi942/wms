package stockreturn

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/cache"
	"wms/backend/internal/httpx"
	"wms/backend/internal/inventory"
	"wms/backend/internal/messaging"
	"wms/backend/internal/user"
)

type Service struct {
	db    *gorm.DB
	repo  *Repository
	inv   *inventory.Repository
	cache *cache.Cache
}

func NewService(db *gorm.DB, repo *Repository, inv *inventory.Repository) *Service {
	return &Service{db: db, repo: repo, inv: inv}
}

func (s *Service) SetCache(c *cache.Cache) { s.cache = c }

func withActions(dto *DTO, actor *httpx.Identity) *DTO {
	dto.AllowedActions = allowedActions(dto.Status, actor.Role, actor.DepartmentID, dto.Department.ID)
	return dto
}

func (s *Service) invalidate(ctx context.Context) {
	if s.cache != nil {
		s.cache.InvalidateStock(ctx)
	}
}

type CreateItemInput struct {
	ProductID    uuid.UUID `json:"product_id" binding:"required"`
	RequestedQty int       `json:"requested_qty" binding:"required,gt=0"`
}

type CreateInput struct {
	Reason string            `json:"reason" binding:"required,min=5,max=1000"`
	Items  []CreateItemInput `json:"items" binding:"required,min=1,max=50,dive"`
}

type ConfirmLine struct {
	ItemID      uuid.UUID `json:"item_id" binding:"required"`
	AcceptedQty int       `json:"accepted_qty" binding:"gte=0"`
}

type ConfirmInput struct {
	Items []ConfirmLine `json:"items" binding:"required,min=1,max=50,dive"`
}

type RejectInput struct {
	Reason string `json:"reason" binding:"required,min=5,max=1000"`
}

func (s *Service) Create(ctx context.Context, in CreateInput, actor *httpx.Identity) (*DTO, error) {
	if actor.Role != user.RoleDepartmentUser || actor.DepartmentID == nil {
		return nil, httpx.Forbidden("only department users can create returns")
	}
	reason := strings.TrimSpace(in.Reason)
	if len(reason) < 5 {
		return nil, httpx.FieldError("reason", "must be at least 5 characters")
	}
	if len(in.Items) < 1 || len(in.Items) > 50 {
		return nil, httpx.FieldError("items", "must contain between 1 and 50 lines")
	}
	seen := map[uuid.UUID]bool{}
	ids := make([]uuid.UUID, 0, len(in.Items))
	for _, it := range in.Items {
		if seen[it.ProductID] {
			return nil, httpx.FieldError("items", "duplicate product in return")
		}
		seen[it.ProductID] = true
		ids = append(ids, it.ProductID)
		if it.RequestedQty < 1 || it.RequestedQty > 1_000_000 {
			return nil, httpx.FieldError("items", "requested quantity must be between 1 and 1,000,000")
		}
	}
	type row struct {
		ID       uuid.UUID `gorm:"column:id"`
		IsActive bool      `gorm:"column:is_active"`
	}
	var products []row
	if err := s.db.WithContext(ctx).Raw(`SELECT id, is_active FROM products WHERE id IN ?`, ids).Scan(&products).Error; err != nil {
		return nil, err
	}
	if len(products) != len(ids) {
		return nil, httpx.FieldError("items", "one or more products do not exist")
	}
	for _, p := range products {
		if !p.IsActive {
			return nil, httpx.Conflict("INACTIVE_PRODUCT", "one or more products are inactive", map[string]any{"product_id": p.ID})
		}
	}

	var returnID uuid.UUID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		number, err := inventory.NextDocumentNumber(tx, inventory.SeqReturn(), "RET")
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		ret := &Return{
			ID:           uuid.Must(uuid.NewV7()),
			ReturnNumber: number,
			DepartmentID: *actor.DepartmentID,
			RequestedBy:  actor.ID,
			Status:       StatusPending,
			Reason:       reason,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		returnID = ret.ID
		items := make([]Item, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, Item{
				ID:           uuid.Must(uuid.NewV7()),
				ReturnID:     ret.ID,
				ProductID:    it.ProductID,
				RequestedQty: it.RequestedQty,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
		if err := s.repo.Create(tx, ret, items); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventReturnCreated, "stock_return", ret.ID, map[string]any{
			"return_id": ret.ID, "return_number": ret.ReturnNumber,
			"department_id": ret.DepartmentID, "requester_id": ret.RequestedBy,
			"item_count": len(items),
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, returnID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, actor *httpx.Identity) (*DTO, error) {
	dto, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("stock return not found")
		}
		return nil, err
	}
	if actor.Role == user.RoleDepartmentUser {
		if actor.DepartmentID == nil || dto.Department.ID != *actor.DepartmentID {
			return nil, httpx.NotFound("stock return not found")
		}
	}
	return withActions(dto, actor), nil
}

func (s *Service) List(ctx context.Context, f Filter) ([]ListDTO, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Confirm(ctx context.Context, returnID uuid.UUID, in ConfirmInput, actor *httpx.Identity) (*DTO, error) {
	acceptedByItem := make(map[uuid.UUID]int, len(in.Items))
	for _, l := range in.Items {
		if l.AcceptedQty < 0 {
			return nil, httpx.FieldError("items", "accepted quantity cannot be negative")
		}
		if _, dup := acceptedByItem[l.ItemID]; dup {
			return nil, httpx.FieldError("items", "duplicate return item")
		}
		acceptedByItem[l.ItemID] = l.AcceptedQty
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ret, err := s.repo.GetForUpdate(tx, returnID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("stock return not found")
			}
			return err
		}
		if ret.Status != StatusPending {
			return httpx.Conflict("INVALID_STATE", "return cannot be confirmed in its current state", map[string]any{"status": ret.Status})
		}
		items, err := s.repo.GetItemsForUpdate(tx, returnID)
		if err != nil {
			return err
		}
		if len(acceptedByItem) != len(items) {
			return httpx.FieldError("items", "every return item must be provided exactly once")
		}
		type line struct {
			item     *Item
			accepted int
			product  *inventory.ProductInfo
		}
		lines := make([]line, 0, len(items))
		anyAccepted := false
		for i := range items {
			accepted, ok := acceptedByItem[items[i].ID]
			if !ok {
				return httpx.FieldError("items", "every return item must be provided exactly once")
			}
			if accepted > items[i].RequestedQty {
				return httpx.Conflict("QUANTITY_EXCEEDS_REMAINING", "accepted quantity exceeds requested quantity", map[string]any{
					"item_id": items[i].ID, "requested": items[i].RequestedQty,
				})
			}
			if accepted > 0 {
				anyAccepted = true
			}
			product, err := s.inv.GetProductInfo(tx, items[i].ProductID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return httpx.NotFound("product not found")
				}
				return err
			}
			if !product.IsActive {
				return httpx.Conflict("INACTIVE_PRODUCT", "product is inactive", map[string]any{"product_id": product.ID})
			}
			lines = append(lines, line{item: &items[i], accepted: accepted, product: product})
		}
		if !anyAccepted {
			return httpx.Validation("validation failed", map[string]string{"items": "at least one accepted quantity must be greater than zero"})
		}
		sort.Slice(lines, func(i, j int) bool { return lines[i].product.ID.String() < lines[j].product.ID.String() })

		for _, l := range lines {
			if l.accepted == 0 {
				continue
			}
			// Serialize concurrent returns for the same department/product.
			if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))`,
				ret.DepartmentID.String(), l.product.ID.String()).Error; err != nil {
				return err
			}
			netIssued, err := s.inv.NetIssued(tx, ret.DepartmentID, l.product.ID)
			if err != nil {
				return err
			}
			if l.accepted > netIssued {
				return httpx.Conflict("RETURN_EXCEEDS_ISSUED", "return exceeds the quantity issued to this department", map[string]any{
					"product_id": l.product.ID, "net_issued": netIssued, "requested": l.accepted,
				})
			}
		}

		type movementLine struct {
			ProductID uuid.UUID `json:"product_id"`
			Quantity  int       `json:"quantity"`
		}
		movementLines := make([]movementLine, 0, len(lines))
		for _, l := range lines {
			if l.accepted == 0 {
				continue
			}
			movement, _, err := s.inv.ApplyMovement(tx, inventory.MovementInput{
				ProductID:       l.product.ID,
				MovementType:    inventory.TypeIn,
				QuantityDelta:   l.accepted,
				ReferenceType:   inventory.RefReturn,
				ReferenceID:     ret.ID,
				ReferenceNumber: ret.ReturnNumber,
				DepartmentID:    &ret.DepartmentID,
				ActorID:         actor.ID,
				Reason:          &ret.Reason,
				MinStock:        l.product.MinStock,
			})
			if err != nil {
				return err
			}
			_ = movement
			if err := s.repo.UpdateItemAccepted(tx, l.item.ID, l.accepted); err != nil {
				return err
			}
			movementLines = append(movementLines, movementLine{ProductID: l.product.ID, Quantity: l.accepted})
		}
		now := time.Now().UTC()
		if err := s.repo.UpdateHeader(tx, ret.ID, map[string]any{
			"status": StatusConfirmed, "processed_by": actor.ID, "processed_at": now,
		}); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventReturnConfirmed, "stock_return", ret.ID, map[string]any{
			"return_id": ret.ID, "return_number": ret.ReturnNumber,
			"requester_id": ret.RequestedBy, "lines": movementLines,
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, returnID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}

func (s *Service) Reject(ctx context.Context, returnID uuid.UUID, reason string, actor *httpx.Identity) (*DTO, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 5 {
		return nil, httpx.FieldError("reason", "must be at least 5 characters")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ret, err := s.repo.GetForUpdate(tx, returnID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("stock return not found")
			}
			return err
		}
		if ret.Status != StatusPending {
			return httpx.Conflict("INVALID_STATE", "return cannot be rejected in its current state", map[string]any{"status": ret.Status})
		}
		now := time.Now().UTC()
		if err := s.repo.UpdateHeader(tx, ret.ID, map[string]any{
			"status": StatusRejected, "rejection_reason": reason,
			"processed_by": actor.ID, "processed_at": now,
		}); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventReturnRejected, "stock_return", ret.ID, map[string]any{
			"return_id": ret.ID, "return_number": ret.ReturnNumber,
			"requester_id": ret.RequestedBy, "reason": reason,
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, returnID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}

func (s *Service) Cancel(ctx context.Context, returnID uuid.UUID, actor *httpx.Identity) (*DTO, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ret, err := s.repo.GetForUpdate(tx, returnID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("stock return not found")
			}
			return err
		}
		if actor.DepartmentID == nil || ret.DepartmentID != *actor.DepartmentID {
			return httpx.NotFound("stock return not found")
		}
		if ret.Status != StatusPending {
			return httpx.Conflict("INVALID_STATE", "only pending returns can be cancelled", map[string]any{"status": ret.Status})
		}
		if err := s.repo.UpdateHeader(tx, ret.ID, map[string]any{"status": StatusCanceled}); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventReturnCanceled, "stock_return", ret.ID, map[string]any{
			"return_id": ret.ID, "return_number": ret.ReturnNumber, "requester_id": ret.RequestedBy,
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, returnID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}
