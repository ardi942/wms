package stockrequest

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
	Purpose string            `json:"purpose" binding:"required,min=3,max=1000"`
	Items   []CreateItemInput `json:"items" binding:"required,min=1,max=50,dive"`
}

type FulfillLine struct {
	ItemID         uuid.UUID `json:"item_id" binding:"required"`
	IssuedQuantity int       `json:"issued_quantity" binding:"required,gt=0"`
}

type FulfillInput struct {
	Items []FulfillLine `json:"items" binding:"required,min=1,max=50,dive"`
}

type RejectInput struct {
	Reason string `json:"reason" binding:"required,min=5,max=1000"`
}

func (s *Service) Create(ctx context.Context, in CreateInput, actor *httpx.Identity) (*DTO, error) {
	if actor.Role != user.RoleDepartmentUser || actor.DepartmentID == nil {
		return nil, httpx.Forbidden("only department users can create requests")
	}
	purpose := strings.TrimSpace(in.Purpose)
	if len(purpose) < 3 {
		return nil, httpx.FieldError("purpose", "must be at least 3 characters")
	}
	if len(in.Items) < 1 || len(in.Items) > 50 {
		return nil, httpx.FieldError("items", "must contain between 1 and 50 lines")
	}
	seen := map[uuid.UUID]bool{}
	for _, it := range in.Items {
		if seen[it.ProductID] {
			return nil, httpx.FieldError("items", "duplicate product in request")
		}
		seen[it.ProductID] = true
		if it.RequestedQty < 1 || it.RequestedQty > 1_000_000 {
			return nil, httpx.FieldError("items", "requested quantity must be between 1 and 1,000,000")
		}
	}
	if err := s.validateActiveProducts(ctx, in.Items); err != nil {
		return nil, err
	}

	var requestID uuid.UUID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		number, err := inventory.NextDocumentNumber(tx, inventory.SeqRequest(), "REQ")
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		req := &Request{
			ID:            uuid.Must(uuid.NewV7()),
			RequestNumber: number,
			DepartmentID:  *actor.DepartmentID,
			RequestedBy:   actor.ID,
			Status:        StatusPending,
			Purpose:       purpose,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		requestID = req.ID
		items := make([]Item, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, Item{
				ID:           uuid.Must(uuid.NewV7()),
				RequestID:    req.ID,
				ProductID:    it.ProductID,
				RequestedQty: it.RequestedQty,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
		if err := s.repo.Create(tx, req, items); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventRequestCreated, "stock_request", req.ID, map[string]any{
			"request_id": req.ID, "request_number": req.RequestNumber,
			"department_id": req.DepartmentID, "requester_id": req.RequestedBy,
			"item_count": len(items),
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}

func withActions(dto *DTO, actor *httpx.Identity) *DTO {
	dto.AllowedActions = allowedActions(dto.Status, actor.Role, actor.DepartmentID, dto.Department.ID)
	return dto
}

func (s *Service) validateActiveProducts(ctx context.Context, items []CreateItemInput) error {
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ProductID)
	}
	type row struct {
		ID       uuid.UUID `gorm:"column:id"`
		IsActive bool      `gorm:"column:is_active"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`SELECT id, is_active FROM products WHERE id IN ?`, ids).Scan(&rows).Error; err != nil {
		return err
	}
	if len(rows) != len(ids) {
		return httpx.FieldError("items", "one or more products do not exist")
	}
	for _, r := range rows {
		if !r.IsActive {
			return httpx.Conflict("INACTIVE_PRODUCT", "one or more products are inactive", map[string]any{"product_id": r.ID})
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, actor *httpx.Identity) (*DTO, error) {
	dto, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("stock request not found")
		}
		return nil, err
	}
	if actor.Role == user.RoleDepartmentUser {
		if actor.DepartmentID == nil || dto.Department.ID != *actor.DepartmentID {
			return nil, httpx.NotFound("stock request not found")
		}
	}
	return withActions(dto, actor), nil
}

func (s *Service) List(ctx context.Context, f Filter) ([]ListDTO, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Fulfill(ctx context.Context, requestID uuid.UUID, in FulfillInput, actor *httpx.Identity) (*DTO, error) {
	lineByItem := make(map[uuid.UUID]int, len(in.Items))
	for _, l := range in.Items {
		if l.IssuedQuantity < 1 {
			return nil, httpx.FieldError("items", "issued quantity must be positive")
		}
		if _, dup := lineByItem[l.ItemID]; dup {
			return nil, httpx.FieldError("items", "duplicate request item")
		}
		lineByItem[l.ItemID] = l.IssuedQuantity
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		req, err := s.repo.GetForUpdate(tx, requestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("stock request not found")
			}
			return err
		}
		if req.Status != StatusPending && req.Status != StatusPartiallyFulfilled {
			return httpx.Conflict("INVALID_STATE", "request cannot be fulfilled in its current state", map[string]any{"status": req.Status})
		}
		items, err := s.repo.GetItemsForUpdate(tx, requestID)
		if err != nil {
			return err
		}
		itemMap := make(map[uuid.UUID]*Item, len(items))
		for i := range items {
			itemMap[items[i].ID] = &items[i]
		}
		type pendingLine struct {
			item     *Item
			issued   int
			product  *inventory.ProductInfo
		}
		lines := make([]pendingLine, 0, len(lineByItem))
		for itemID, qty := range lineByItem {
			item, ok := itemMap[itemID]
			if !ok {
				return httpx.FieldError("items", "item does not belong to this request")
			}
			remaining := item.RequestedQty - item.IssuedQty
			if qty > remaining {
				return httpx.Conflict("QUANTITY_EXCEEDS_REMAINING", "issued quantity exceeds remaining request quantity", map[string]any{
					"item_id": itemID, "remaining": remaining,
				})
			}
			product, err := s.inv.GetProductInfo(tx, item.ProductID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return httpx.NotFound("product not found")
				}
				return err
			}
			if !product.IsActive {
				return httpx.Conflict("INACTIVE_PRODUCT", "product is inactive", map[string]any{"product_id": product.ID})
			}
			lines = append(lines, pendingLine{item: item, issued: qty, product: product})
		}
		sort.Slice(lines, func(i, j int) bool { return lines[i].product.ID.String() < lines[j].product.ID.String() })

		type movementLine struct {
			ProductID uuid.UUID `json:"product_id"`
			Quantity  int       `json:"quantity"`
		}
		movementLines := make([]movementLine, 0, len(lines))
		newIssued := make(map[uuid.UUID]int, len(lines))
		for _, l := range lines {
			movement, lowStock, err := s.inv.ApplyMovement(tx, inventory.MovementInput{
				ProductID:       l.product.ID,
				MovementType:    inventory.TypeOut,
				QuantityDelta:   -l.issued,
				ReferenceType:   inventory.RefRequest,
				ReferenceID:     req.ID,
				ReferenceNumber: req.RequestNumber,
				DepartmentID:    &req.DepartmentID,
				ActorID:         actor.ID,
				Notes:           &req.Purpose,
				MinStock:        l.product.MinStock,
			})
			if err != nil {
				return err
			}
			if err := messaging.Enqueue(tx, messaging.EventStockIssued, "stock_movement", movement.ID, map[string]any{
				"movement_id": movement.ID, "request_id": req.ID, "request_number": req.RequestNumber,
				"product_id": movement.ProductID, "quantity": l.issued, "balance_after": movement.BalanceAfter,
				"department_id": req.DepartmentID, "requester_id": req.RequestedBy,
				"actor_id": actor.ID, "low_stock": lowStock,
			}); err != nil {
				return err
			}
			newIssued[l.item.ID] = l.item.IssuedQty + l.issued
			if err := s.repo.UpdateItemIssued(tx, l.item.ID, newIssued[l.item.ID]); err != nil {
				return err
			}
			movementLines = append(movementLines, movementLine{ProductID: l.product.ID, Quantity: l.issued})
		}

		totalRequested, totalIssued := 0, 0
		for i := range items {
			totalRequested += items[i].RequestedQty
			if v, ok := newIssued[items[i].ID]; ok {
				totalIssued += v
			} else {
				totalIssued += items[i].IssuedQty
			}
		}
		status := DeriveStatus(totalRequested, totalIssued)
		now := time.Now().UTC()
		if err := s.repo.UpdateHeader(tx, req.ID, map[string]any{
			"status": status, "processed_by": actor.ID, "processed_at": now,
		}); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventRequestFulfilled, "stock_request", req.ID, map[string]any{
			"request_id": req.ID, "request_number": req.RequestNumber,
			"department_id": req.DepartmentID, "requester_id": req.RequestedBy,
			"status": status, "partial": status == StatusPartiallyFulfilled,
			"lines": movementLines,
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}

func (s *Service) Reject(ctx context.Context, requestID uuid.UUID, reason string, actor *httpx.Identity) (*DTO, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 5 {
		return nil, httpx.FieldError("reason", "must be at least 5 characters")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		req, err := s.repo.GetForUpdate(tx, requestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("stock request not found")
			}
			return err
		}
		if req.Status != StatusPending && req.Status != StatusPartiallyFulfilled {
			return httpx.Conflict("INVALID_STATE", "request cannot be rejected in its current state", map[string]any{"status": req.Status})
		}
		now := time.Now().UTC()
		if err := s.repo.UpdateHeader(tx, req.ID, map[string]any{
			"status": StatusRejected, "rejection_reason": reason,
			"processed_by": actor.ID, "processed_at": now,
		}); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventRequestRejected, "stock_request", req.ID, map[string]any{
			"request_id": req.ID, "request_number": req.RequestNumber,
			"requester_id": req.RequestedBy, "reason": reason,
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}

func (s *Service) Cancel(ctx context.Context, requestID uuid.UUID, actor *httpx.Identity) (*DTO, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		req, err := s.repo.GetForUpdate(tx, requestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("stock request not found")
			}
			return err
		}
		if actor.DepartmentID == nil || req.DepartmentID != *actor.DepartmentID {
			return httpx.NotFound("stock request not found")
		}
		if req.Status != StatusPending {
			return httpx.Conflict("INVALID_STATE", "only pending requests can be cancelled", map[string]any{"status": req.Status})
		}
		if err := s.repo.UpdateHeader(tx, req.ID, map[string]any{"status": StatusCanceled}); err != nil {
			return err
		}
		return messaging.Enqueue(tx, messaging.EventRequestCanceled, "stock_request", req.ID, map[string]any{
			"request_id": req.ID, "request_number": req.RequestNumber, "requester_id": req.RequestedBy,
		})
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	dto, err := s.repo.GetDetail(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return withActions(dto, actor), nil
}
