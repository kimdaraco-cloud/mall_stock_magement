// @ai-modified 2026-07-02 add dashboard/report view models
package models

// DashboardStats backs the dashboard cards. StockValue is a decimal string
// (Σ quantity × cost_price), never a float.
type DashboardStats struct {
	TotalProducts int
	StockValue    string
	LowStockCount int
	ActiveStores  int
}

// LowStockItem is one row of the low-stock report.
type LowStockItem struct {
	ProductID    int64
	SKU          string
	Name         string
	StoreName    string
	CategoryName string
	Quantity     int
	ReorderLevel int
	Unit         string
	SuggestedQty int // suggested reorder quantity
}

// ValuationRow is one group line of the stock valuation report.
type ValuationRow struct {
	Group    string // store or category name
	Products int
	Units    int
	Value    string // decimal string
}
