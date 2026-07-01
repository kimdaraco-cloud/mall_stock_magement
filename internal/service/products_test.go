// @ai-modified 2026-07-02 add product service validation tests
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"mallstock/internal/repository"
)

func TestProductValidation(t *testing.T) {
	pool := testPool(t)
	svc := &ProductService{Products: &repository.ProductRepo{DB: pool}}
	ctx := context.Background()

	// A store to attach products to, and one existing product for SKU clashes.
	var storeID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO stores (name) VALUES ('validation-test-store') RETURNING id`).Scan(&storeID); err != nil {
		t.Fatalf("create store: %v", err)
	}
	existingSKU := fmt.Sprintf("DUP-%d", time.Now().UnixNano())
	var existingID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO products (sku, name, store_id) VALUES ($1, 'existing', $2) RETURNING id`,
		existingSKU, storeID).Scan(&existingID); err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM products WHERE store_id = $1`, storeID)
		_, _ = pool.Exec(ctx, `DELETE FROM stores WHERE id = $1`, storeID)
	})

	valid := func() ProductInput {
		return ProductInput{
			SKU:  fmt.Sprintf("OK-%d", time.Now().UnixNano()),
			Name: "Valid product", StoreID: storeID,
			CostPrice: "1.50", SellingPrice: "3.99", Unit: "pcs", IsActive: true,
		}
	}

	tests := []struct {
		name      string
		mutate    func(in *ProductInput)
		wantField string // "" means expect success
	}{
		{"valid product passes", func(in *ProductInput) {}, ""},
		{"empty SKU rejected", func(in *ProductInput) { in.SKU = "" }, "sku"},
		{"duplicate SKU rejected", func(in *ProductInput) { in.SKU = existingSKU }, "sku"},
		{"empty name rejected", func(in *ProductInput) { in.Name = " " }, "name"},
		{"missing store rejected", func(in *ProductInput) { in.StoreID = 0 }, "store_id"},
		{"negative price rejected", func(in *ProductInput) { in.CostPrice = "-1" }, "cost_price"},
		{"three decimals rejected", func(in *ProductInput) { in.SellingPrice = "1.999" }, "selling_price"},
		{"non-numeric price rejected", func(in *ProductInput) { in.CostPrice = "abc" }, "cost_price"},
		{"price with exponent rejected", func(in *ProductInput) { in.CostPrice = "1e3" }, "cost_price"},
		{"negative reorder level rejected", func(in *ProductInput) { in.ReorderLevel = -1 }, "reorder_level"},
		{"blank prices default to zero", func(in *ProductInput) { in.CostPrice = ""; in.SellingPrice = "" }, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := valid()
			tc.mutate(&in)
			p, err := svc.Create(ctx, in)

			if tc.wantField == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if p.Quantity != 0 {
					t.Errorf("new product quantity = %d, want 0", p.Quantity)
				}
				return
			}
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("want ValidationError, got %v", err)
			}
			if _, ok := verr.Fields[tc.wantField]; !ok {
				t.Errorf("want error on field %q, got %v", tc.wantField, verr.Fields)
			}
		})
	}
}
