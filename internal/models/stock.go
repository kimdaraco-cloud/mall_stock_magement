// @ai-modified 2026-07-02 add StockMovement model and movement type constants
package models

import "time"

// Movement types. quantity is always positive; the type gives direction.
const (
	MovementIn         = "in"
	MovementOut        = "out"
	MovementAdjustment = "adjustment"
)

// ValidMovementType reports whether t is a known movement type.
func ValidMovementType(t string) bool {
	return t == MovementIn || t == MovementOut || t == MovementAdjustment
}

// StockMovement is an immutable audit record of a stock change.
type StockMovement struct {
	ID            int64
	ProductID     int64
	MovementType  string
	Quantity      int // always positive
	QuantityAfter int // running balance after this movement
	Reference     string
	Notes         string
	UserID        *int64
	CreatedAt     time.Time

	// Joined display fields, not columns.
	ProductName string
	ProductSKU  string
	StoreName   string
	UserName    string
}
