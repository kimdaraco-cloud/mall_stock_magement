// @ai-modified 2026-07-02 add Category, Supplier and Product models
package models

import "time"

// Category groups products.
type Category struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
}

// Supplier is where products are purchased from.
type Supplier struct {
	ID            int64
	Name          string
	ContactPerson string
	Email         string
	Phone         string
	Address       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Product is a stock-keeping item. Prices are decimal strings (e.g. "12.50")
// mapped to NUMERIC in Postgres — never floats.
type Product struct {
	ID           int64
	SKU          string
	Barcode      string
	Name         string
	Description  string
	CategoryID   *int64
	SupplierID   *int64
	StoreID      int64
	CostPrice    string
	SellingPrice string
	Quantity     int
	ReorderLevel int
	Unit         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Joined display fields, not columns.
	CategoryName string
	SupplierName string
	StoreName    string
}

// LowStock reports whether the product is at or below its reorder level.
func (p *Product) LowStock() bool { return p.Quantity <= p.ReorderLevel }
