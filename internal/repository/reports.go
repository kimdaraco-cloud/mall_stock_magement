// @ai-modified 2026-07-02 add report repository (dashboard stats, low stock, valuation)
package repository

import (
	"context"
	"fmt"

	"mallstock/internal/models"
)

// ReportRepo runs read-only aggregate queries for dashboard and reports.
type ReportRepo struct {
	DB DB
}

// DashboardStats returns the four dashboard card values in one round trip.
func (r *ReportRepo) DashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	var s models.DashboardStats
	err := r.DB.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM products WHERE is_active),
			(SELECT COALESCE(sum(quantity * cost_price), 0)::text FROM products WHERE is_active),
			(SELECT count(*) FROM products WHERE is_active AND quantity <= reorder_level),
			(SELECT count(*) FROM stores WHERE is_active)`,
	).Scan(&s.TotalProducts, &s.StockValue, &s.LowStockCount, &s.ActiveStores)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: %w", err)
	}
	return &s, nil
}

// LowStock lists active products at or below their reorder level, most
// urgent first. storeID 0 means all stores. The suggested reorder quantity
// tops the product back up to twice its reorder level (min 1).
func (r *ReportRepo) LowStock(ctx context.Context, storeID int64) ([]models.LowStockItem, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT p.id, p.sku, p.name, s.name, COALESCE(c.name,''), p.quantity, p.reorder_level, p.unit,
		       GREATEST(p.reorder_level * 2 - p.quantity, 1)
		FROM products p
		JOIN stores s ON s.id = p.store_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE p.is_active AND p.quantity <= p.reorder_level
		  AND ($1 = 0 OR p.store_id = $1)
		ORDER BY (p.reorder_level - p.quantity) DESC, p.name`, storeID)
	if err != nil {
		return nil, fmt.Errorf("low stock report: %w", err)
	}
	defer rows.Close()

	var items []models.LowStockItem
	for rows.Next() {
		var it models.LowStockItem
		if err := rows.Scan(&it.ProductID, &it.SKU, &it.Name, &it.StoreName, &it.CategoryName,
			&it.Quantity, &it.ReorderLevel, &it.Unit, &it.SuggestedQty); err != nil {
			return nil, fmt.Errorf("low stock scan: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ValuationByStore aggregates active-product stock value per store.
func (r *ReportRepo) ValuationByStore(ctx context.Context) ([]models.ValuationRow, error) {
	return r.valuation(ctx, `
		SELECT s.name, count(p.id), COALESCE(sum(p.quantity), 0),
		       COALESCE(sum(p.quantity * p.cost_price), 0)::text
		FROM products p
		JOIN stores s ON s.id = p.store_id
		WHERE p.is_active
		GROUP BY s.name ORDER BY s.name`)
}

// ValuationByCategory aggregates active-product stock value per category.
func (r *ReportRepo) ValuationByCategory(ctx context.Context) ([]models.ValuationRow, error) {
	return r.valuation(ctx, `
		SELECT COALESCE(c.name, 'Uncategorised'), count(p.id), COALESCE(sum(p.quantity), 0),
		       COALESCE(sum(p.quantity * p.cost_price), 0)::text
		FROM products p
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE p.is_active
		GROUP BY c.name ORDER BY COALESCE(c.name, 'Uncategorised')`)
}

func (r *ReportRepo) valuation(ctx context.Context, query string) ([]models.ValuationRow, error) {
	rows, err := r.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("valuation report: %w", err)
	}
	defer rows.Close()

	var out []models.ValuationRow
	for rows.Next() {
		var v models.ValuationRow
		if err := rows.Scan(&v.Group, &v.Products, &v.Units, &v.Value); err != nil {
			return nil, fmt.Errorf("valuation scan: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
