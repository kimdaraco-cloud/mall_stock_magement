// @ai-modified 2026-07-02 add stock movement repository (SQL only)
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mallstock/internal/models"
)

// MovementRepo persists stock movements. Create must run inside the same
// transaction that updates products.quantity (stock invariant).
type MovementRepo struct {
	DB DB
}

// MovementFilter narrows List results. Zero values mean "no filter".
type MovementFilter struct {
	ProductID int64
	StoreID   int64
	Type      string
	From      time.Time
	To        time.Time
	Page      int
	PerPage   int
}

const movementCols = `m.id, m.product_id, m.movement_type, m.quantity, m.quantity_after,
	COALESCE(m.reference,''), COALESCE(m.notes,''), m.user_id, m.created_at,
	p.name, p.sku, s.name, COALESCE(u.full_name,'')`

const movementJoins = ` FROM stock_movements m
	JOIN products p ON p.id = m.product_id
	JOIN stores s ON s.id = p.store_id
	LEFT JOIN users u ON u.id = m.user_id`

func (r *MovementRepo) scan(rows interface {
	Scan(dest ...any) error
}) (*models.StockMovement, error) {
	var m models.StockMovement
	err := rows.Scan(&m.ID, &m.ProductID, &m.MovementType, &m.Quantity, &m.QuantityAfter,
		&m.Reference, &m.Notes, &m.UserID, &m.CreatedAt,
		&m.ProductName, &m.ProductSKU, &m.StoreName, &m.UserName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create inserts an immutable movement row.
func (r *MovementRepo) Create(ctx context.Context, m *models.StockMovement) error {
	err := r.DB.QueryRow(ctx,
		`INSERT INTO stock_movements (product_id, movement_type, quantity, quantity_after, reference, notes, user_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		m.ProductID, m.MovementType, m.Quantity, m.QuantityAfter, m.Reference, m.Notes, m.UserID,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("create movement: %w", err)
	}
	return nil
}

// ListRecentByProduct returns the newest movements for one product.
func (r *MovementRepo) ListRecentByProduct(ctx context.Context, productID int64, limit int) ([]models.StockMovement, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT `+movementCols+movementJoins+`
		 WHERE m.product_id = $1 ORDER BY m.created_at DESC, m.id DESC LIMIT $2`,
		productID, limit)
	if err != nil {
		return nil, fmt.Errorf("recent movements: %w", err)
	}
	defer rows.Close()

	var out []models.StockMovement
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("recent movements scan: %w", err)
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func buildMovementWhere(f MovementFilter) (string, []any) {
	var conds []string
	var args []any
	if f.ProductID > 0 {
		args = append(args, f.ProductID)
		conds = append(conds, fmt.Sprintf("m.product_id = $%d", len(args)))
	}
	if f.StoreID > 0 {
		args = append(args, f.StoreID)
		conds = append(conds, fmt.Sprintf("p.store_id = $%d", len(args)))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		conds = append(conds, fmt.Sprintf("m.movement_type = $%d", len(args)))
	}
	if !f.From.IsZero() {
		args = append(args, f.From)
		conds = append(conds, fmt.Sprintf("m.created_at >= $%d", len(args)))
	}
	if !f.To.IsZero() {
		args = append(args, f.To)
		conds = append(conds, fmt.Sprintf("m.created_at < $%d", len(args)))
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// List returns one page of movements plus the total count.
func (r *MovementRepo) List(ctx context.Context, f MovementFilter) ([]models.StockMovement, int, error) {
	where, args := buildMovementWhere(f)

	var total int
	if err := r.DB.QueryRow(ctx, `SELECT count(*)`+movementJoins+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count movements: %w", err)
	}

	if f.PerPage <= 0 {
		f.PerPage = 50
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)
	q := `SELECT ` + movementCols + movementJoins + where +
		fmt.Sprintf(" ORDER BY m.created_at DESC, m.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	var out []models.StockMovement
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("list movements scan: %w", err)
		}
		out = append(out, *m)
	}
	return out, total, rows.Err()
}
