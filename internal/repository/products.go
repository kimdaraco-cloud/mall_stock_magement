// @ai-modified 2026-07-02 add product repository with filtered list and pagination
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"mallstock/internal/models"
)

// ProductRepo persists products.
type ProductRepo struct {
	DB DB
}

// ProductFilter narrows List results. Zero values mean "no filter".
type ProductFilter struct {
	Query      string // matches name / SKU / barcode, case-insensitive
	StoreID    int64
	CategoryID int64
	LowStock   bool
	ActiveOnly bool
	Page       int // 1-based
	PerPage    int
}

const productCols = `p.id, p.sku, COALESCE(p.barcode,''), p.name, COALESCE(p.description,''),
	p.category_id, p.supplier_id, p.store_id,
	p.cost_price::text, p.selling_price::text,
	p.quantity, p.reorder_level, p.unit, p.is_active, p.created_at, p.updated_at,
	COALESCE(c.name,''), COALESCE(su.name,''), s.name`

const productJoins = ` FROM products p
	JOIN stores s ON s.id = p.store_id
	LEFT JOIN categories c ON c.id = p.category_id
	LEFT JOIN suppliers su ON su.id = p.supplier_id`

func scanProduct(row pgx.Row) (*models.Product, error) {
	var p models.Product
	err := row.Scan(&p.ID, &p.SKU, &p.Barcode, &p.Name, &p.Description,
		&p.CategoryID, &p.SupplierID, &p.StoreID,
		&p.CostPrice, &p.SellingPrice,
		&p.Quantity, &p.ReorderLevel, &p.Unit, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		&p.CategoryName, &p.SupplierName, &p.StoreName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// buildWhere translates a ProductFilter into a WHERE clause + args.
func buildWhere(f ProductFilter) (string, []any) {
	var conds []string
	var args []any
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := fmt.Sprintf("$%d", len(args))
		conds = append(conds, fmt.Sprintf("(p.name ILIKE %s OR p.sku ILIKE %s OR p.barcode ILIKE %s)", n, n, n))
	}
	if f.StoreID > 0 {
		args = append(args, f.StoreID)
		conds = append(conds, fmt.Sprintf("p.store_id = $%d", len(args)))
	}
	if f.CategoryID > 0 {
		args = append(args, f.CategoryID)
		conds = append(conds, fmt.Sprintf("p.category_id = $%d", len(args)))
	}
	if f.LowStock {
		conds = append(conds, "p.quantity <= p.reorder_level")
	}
	if f.ActiveOnly {
		conds = append(conds, "p.is_active")
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// List returns one page of products plus the total row count for pagination.
func (r *ProductRepo) List(ctx context.Context, f ProductFilter) ([]models.Product, int, error) {
	where, args := buildWhere(f)

	var total int
	if err := r.DB.QueryRow(ctx, `SELECT count(*)`+productJoins+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)
	q := `SELECT ` + productCols + productJoins + where +
		fmt.Sprintf(" ORDER BY p.name, p.id LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("list products scan: %w", err)
		}
		products = append(products, *p)
	}
	return products, total, rows.Err()
}

func (r *ProductRepo) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	p, err := scanProduct(r.DB.QueryRow(ctx,
		`SELECT `+productCols+productJoins+` WHERE p.id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *ProductRepo) Create(ctx context.Context, p *models.Product) error {
	err := r.DB.QueryRow(ctx,
		`INSERT INTO products (sku, barcode, name, description, category_id, supplier_id,
		   store_id, cost_price, selling_price, quantity, reorder_level, unit, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8::numeric, $9::numeric, $10, $11, $12, $13)
		 RETURNING id, created_at, updated_at`,
		p.SKU, p.Barcode, p.Name, p.Description, p.CategoryID, p.SupplierID,
		p.StoreID, p.CostPrice, p.SellingPrice, p.Quantity, p.ReorderLevel, p.Unit, p.IsActive,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

// Update changes catalog fields only — quantity is owned by the stock engine
// (see CLAUDE.md stock invariant) and is deliberately not touched here.
func (r *ProductRepo) Update(ctx context.Context, p *models.Product) error {
	tag, err := r.DB.Exec(ctx,
		`UPDATE products SET sku = $1, barcode = $2, name = $3, description = $4,
		 category_id = $5, supplier_id = $6, store_id = $7,
		 cost_price = $8::numeric, selling_price = $9::numeric,
		 reorder_level = $10, unit = $11, is_active = $12, updated_at = now()
		 WHERE id = $13`,
		p.SKU, p.Barcode, p.Name, p.Description, p.CategoryID, p.SupplierID,
		p.StoreID, p.CostPrice, p.SellingPrice, p.ReorderLevel, p.Unit, p.IsActive, p.ID)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update product: %w", ErrNotFound)
	}
	return nil
}

// SetActive toggles soft-delete state.
func (r *ProductRepo) SetActive(ctx context.Context, id int64, active bool) error {
	tag, err := r.DB.Exec(ctx,
		`UPDATE products SET is_active = $1, updated_at = now() WHERE id = $2`, active, id)
	if err != nil {
		return fmt.Errorf("set product active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set product active: %w", ErrNotFound)
	}
	return nil
}

// SKUTaken reports whether sku is used by a product other than excludeID.
func (r *ProductRepo) SKUTaken(ctx context.Context, sku string, excludeID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM products WHERE sku = $1 AND id <> $2)`,
		sku, excludeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sku taken: %w", err)
	}
	return exists, nil
}
