// @ai-modified 2026-07-02 add transactional stock service (in/out/adjust)
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mallstock/internal/models"
	"mallstock/internal/repository"
)

// Stock operation errors.
var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidQuantity   = errors.New("quantity must be a positive number")
)

// StockService owns every change to products.quantity. All operations insert
// a stock_movements row and update the cached quantity in ONE transaction —
// the stock invariant (CLAUDE.md).
type StockService struct {
	Pool      *pgxpool.Pool
	Movements *repository.MovementRepo // read-side (history pages)
}

// MovementInput is the shared payload for stock operations.
type MovementInput struct {
	ProductID int64
	Quantity  int // for adjust: the corrected absolute count
	Reference string
	Notes     string
	UserID    int64
}

// inTx runs fn inside a transaction, committing on nil error.
func (s *StockService) inTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// apply locks the product row, computes the new balance via compute, then
// inserts the movement and updates the cached quantity atomically.
func (s *StockService) apply(ctx context.Context, in MovementInput, movementType string,
	compute func(current int) (newQty, movementQty int, err error)) (*models.StockMovement, error) {

	var m *models.StockMovement
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		products := &repository.ProductRepo{DB: tx}
		movements := &repository.MovementRepo{DB: tx}

		current, err := products.GetQuantityForUpdate(ctx, in.ProductID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		newQty, movementQty, err := compute(current)
		if err != nil {
			return err
		}

		var userID *int64
		if in.UserID > 0 {
			userID = &in.UserID
		}
		m = &models.StockMovement{
			ProductID:     in.ProductID,
			MovementType:  movementType,
			Quantity:      movementQty,
			QuantityDelta: newQty - current, // signed; makes adjustments self-describing
			QuantityAfter: newQty,
			Reference:     strings.TrimSpace(in.Reference),
			Notes:         strings.TrimSpace(in.Notes),
			UserID:        userID,
		}
		if err := movements.Create(ctx, m); err != nil {
			return err
		}
		return products.SetQuantity(ctx, in.ProductID, newQty)
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ReceiveStock records a stock-in.
func (s *StockService) ReceiveStock(ctx context.Context, in MovementInput) (*models.StockMovement, error) {
	if in.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	return s.apply(ctx, in, models.MovementIn, func(current int) (int, int, error) {
		return current + in.Quantity, in.Quantity, nil
	})
}

// IssueStock records a stock-out, rejecting anything that would go negative.
func (s *StockService) IssueStock(ctx context.Context, in MovementInput) (*models.StockMovement, error) {
	if in.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	return s.apply(ctx, in, models.MovementOut, func(current int) (int, int, error) {
		if in.Quantity > current {
			return 0, 0, fmt.Errorf("%w: have %d, requested %d", ErrInsufficientStock, current, in.Quantity)
		}
		return current - in.Quantity, in.Quantity, nil
	})
}

// AdjustStock sets the count to a corrected value; the movement records the
// absolute delta and the new balance (quantity_after).
func (s *StockService) AdjustStock(ctx context.Context, in MovementInput) (*models.StockMovement, error) {
	if in.Quantity < 0 {
		return nil, ErrInvalidQuantity
	}
	return s.apply(ctx, in, models.MovementAdjustment, func(current int) (int, int, error) {
		delta := in.Quantity - current
		if delta < 0 {
			delta = -delta
		}
		return in.Quantity, delta, nil
	})
}

// RecentForProduct returns the newest movements for a product detail page.
func (s *StockService) RecentForProduct(ctx context.Context, productID int64, limit int) ([]models.StockMovement, error) {
	return s.Movements.ListRecentByProduct(ctx, productID, limit)
}

// History returns a filtered, paginated movement list.
func (s *StockService) History(ctx context.Context, f repository.MovementFilter) ([]models.StockMovement, int, error) {
	return s.Movements.List(ctx, f)
}
