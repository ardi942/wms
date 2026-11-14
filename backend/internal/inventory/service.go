package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/cache"
	"wms/backend/internal/httpx"
	"wms/backend/internal/messaging"
	"wms/backend/internal/user"
)

type Service struct {
	db    *gorm.DB
	repo  *Repository
	cache *cache.Cache
}

func NewService(db *gorm.DB, repo *Repository) *Service {
	return &Service{db: db, repo: repo}
}

func (s *Service) SetCache(c *cache.Cache) { s.cache = c }

func (s *Service) Repo() *Repository { return s.repo }

type ReceiptInput struct {
	ProductID       uuid.UUID `json:"product_id" binding:"required"`
	Quantity        int       `json:"quantity" binding:"required,gt=0"`
	Supplier        string    `json:"supplier" binding:"required,min=1,max=200"`
	ReferenceNumber string    `json:"reference_number" binding:"required,min=1,max=100"`
	Notes           *string   `json:"notes" binding:"omitempty,max=1000"`
}

type AdjustmentInput struct {
	ProductID     uuid.UUID `json:"product_id" binding:"required"`
	QuantityDelta int       `json:"quantity_delta" binding:"required"`
	Reason        string    `json:"reason" binding:"required,min=5,max=1000"`
}

func (s *Service) Receive(ctx context.Context, in ReceiptInput, actorID uuid.UUID) (*MovementDTO, error) {
	if in.Quantity > 1_000_000_000 {
		return nil, httpx.FieldError("quantity", "is too large")
	}
	supplier := strings.TrimSpace(in.Supplier)
	reference := strings.TrimSpace(in.ReferenceNumber)
	if supplier == "" {
		return nil, httpx.FieldError("supplier", "is required")
	}
	if reference == "" {
		return nil, httpx.FieldError("reference_number", "is required")
	}
	var movementID uuid.UUID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, err := s.repo.GetProductInfo(tx, in.ProductID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("product not found")
			}
			return err
		}
		if !product.IsActive {
			return httpx.Conflict("INACTIVE_PRODUCT", "product is inactive", nil)
		}
		referenceID := uuid.Must(uuid.NewV7())
		movement, _, err := s.repo.ApplyMovement(tx, MovementInput{
			ProductID:       in.ProductID,
			MovementType:    TypeIn,
			QuantityDelta:   in.Quantity,
			ReferenceType:   RefReceipt,
			ReferenceID:     referenceID,
			ReferenceNumber: reference,
			ActorID:         actorID,
			Supplier:        &supplier,
			Notes:           trimPtr(in.Notes),
			MinStock:        product.MinStock,
		})
		if err != nil {
			return err
		}
		movementID = movement.ID
		return messaging.Enqueue(tx, messaging.EventStockReceived, "stock_movement", movement.ID, map[string]any{
			"movement_id":      movement.ID,
			"product_id":       movement.ProductID,
			"quantity":         movement.QuantityDelta,
			"balance_after":    movement.BalanceAfter,
			"reference_number": movement.ReferenceNumber,
			"actor_id":         actorID,
			"low_stock":        false,
		})
	})
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.InvalidateStock(ctx)
	}
	return s.repo.GetMovementDTO(ctx, movementID)
}

func (s *Service) Adjust(ctx context.Context, in AdjustmentInput, actorID uuid.UUID) (*MovementDTO, error) {
	if in.QuantityDelta == 0 {
		return nil, httpx.FieldError("quantity_delta", "must not be zero")
	}
	reason := strings.TrimSpace(in.Reason)
	if len(reason) < 5 {
		return nil, httpx.FieldError("reason", "must be at least 5 characters")
	}
	var movementID uuid.UUID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, err := s.repo.GetProductInfo(tx, in.ProductID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("product not found")
			}
			return err
		}
		if !product.IsActive {
			return httpx.Conflict("INACTIVE_PRODUCT", "product is inactive", nil)
		}
		number, err := NextDocumentNumber(tx, SeqAdjustment(), "ADJ")
		if err != nil {
			return err
		}
		movement, lowStock, err := s.repo.ApplyMovement(tx, MovementInput{
			ProductID:       in.ProductID,
			MovementType:    TypeAdjustment,
			QuantityDelta:   in.QuantityDelta,
			ReferenceType:   RefAdjustment,
			ReferenceID:     uuid.Must(uuid.NewV7()),
			ReferenceNumber: number,
			ActorID:         actorID,
			Reason:          &reason,
			MinStock:        product.MinStock,
		})
		if err != nil {
			return err
		}
		movementID = movement.ID
		return messaging.Enqueue(tx, messaging.EventStockAdjusted, "stock_movement", movement.ID, map[string]any{
			"movement_id":   movement.ID,
			"product_id":    movement.ProductID,
			"delta":         movement.QuantityDelta,
			"balance_after": movement.BalanceAfter,
			"reason":        reason,
			"actor_id":      actorID,
			"low_stock":     lowStock,
		})
	})
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.InvalidateStock(ctx)
	}
	return s.repo.GetMovementDTO(ctx, movementID)
}

func (s *Service) ListBalances(ctx context.Context, f BalanceFilter) ([]InventoryDTO, int64, error) {
	rows, total, err := s.repo.ListBalances(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	stripCost(rows, f.Role)
	return rows, total, nil
}

func (s *Service) GetBalance(ctx context.Context, productID uuid.UUID, role string) (*InventoryDTO, error) {
	dto, err := s.repo.GetBalance(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("product not found")
		}
		return nil, err
	}
	if role != user.RoleSupervisor {
		dto.UnitCost = nil
		dto.StockValue = 0
	}
	return dto, nil
}

func stripCost(rows []InventoryDTO, role string) {
	if role == user.RoleSupervisor {
		return
	}
	for i := range rows {
		rows[i].UnitCost = nil
		rows[i].StockValue = 0
	}
}

func (s *Service) ListMovements(ctx context.Context, f MovementFilter) ([]MovementDTO, int64, error) {
	return s.repo.ListMovements(ctx, f)
}

func trimPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}
