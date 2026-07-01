// @ai-modified 2026-07-02 add stock service tests (in/out/adjust, negative guard, concurrency)
package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"mallstock/internal/models"
	"mallstock/internal/repository"
)

// testPool connects to the test database, skipping if unavailable.
// TEST_DATABASE_URL wins; the local dev default is the fallback.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/mall_stock?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("test database unreachable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// newTestProduct creates a throwaway store+product and removes them after.
func newTestProduct(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()

	var storeID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO stores (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("test-store-%s", t.Name())).Scan(&storeID)
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}

	var productID int64
	err = pool.QueryRow(ctx,
		`INSERT INTO products (sku, name, store_id) VALUES ($1, $2, $3) RETURNING id`,
		fmt.Sprintf("TEST-%s-%d", t.Name(), time.Now().UnixNano()),
		"test product", storeID).Scan(&productID)
	if err != nil {
		t.Fatalf("create test product: %v", err)
	}

	t.Cleanup(func() {
		// movements cascade with the product
		_, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
		_, _ = pool.Exec(ctx, `DELETE FROM stores WHERE id = $1`, storeID)
	})
	return productID
}

func quantityOf(t *testing.T, pool *pgxpool.Pool, productID int64) int {
	t.Helper()
	var q int
	if err := pool.QueryRow(context.Background(),
		`SELECT quantity FROM products WHERE id = $1`, productID).Scan(&q); err != nil {
		t.Fatalf("read quantity: %v", err)
	}
	return q
}

func TestStockOperations(t *testing.T) {
	pool := testPool(t)
	svc := &StockService{Pool: pool, Movements: &repository.MovementRepo{DB: pool}}
	ctx := context.Background()

	tests := []struct {
		name      string
		run       func(t *testing.T, productID int64)
		wantQty   int
		wantAfter []int // quantity_after per movement, oldest first
	}{
		{
			name: "receive then issue keeps a correct running balance",
			run: func(t *testing.T, id int64) {
				mustOK(t, errOnly(svc.ReceiveStock(ctx, MovementInput{ProductID: id, Quantity: 10, Reference: "PO-1"})))
				mustOK(t, errOnly(svc.ReceiveStock(ctx, MovementInput{ProductID: id, Quantity: 5})))
				mustOK(t, errOnly(svc.IssueStock(ctx, MovementInput{ProductID: id, Quantity: 7, Reference: "sale"})))
			},
			wantQty:   8,
			wantAfter: []int{10, 15, 8},
		},
		{
			name: "issue below zero is rejected and leaves state untouched",
			run: func(t *testing.T, id int64) {
				mustOK(t, errOnly(svc.ReceiveStock(ctx, MovementInput{ProductID: id, Quantity: 3})))
				_, err := svc.IssueStock(ctx, MovementInput{ProductID: id, Quantity: 4})
				if !errors.Is(err, ErrInsufficientStock) {
					t.Fatalf("want ErrInsufficientStock, got %v", err)
				}
			},
			wantQty:   3,
			wantAfter: []int{3},
		},
		{
			name: "adjustment records the delta and sets the count",
			run: func(t *testing.T, id int64) {
				mustOK(t, errOnly(svc.ReceiveStock(ctx, MovementInput{ProductID: id, Quantity: 20})))
				m, err := svc.AdjustStock(ctx, MovementInput{ProductID: id, Quantity: 12, Reference: "recount"})
				if err != nil {
					t.Fatalf("adjust: %v", err)
				}
				if m.Quantity != 8 { // |12-20|
					t.Errorf("adjustment delta = %d, want 8", m.Quantity)
				}
			},
			wantQty:   12,
			wantAfter: []int{20, 12},
		},
		{
			name: "zero or negative quantities are invalid",
			run: func(t *testing.T, id int64) {
				if _, err := svc.ReceiveStock(ctx, MovementInput{ProductID: id, Quantity: 0}); !errors.Is(err, ErrInvalidQuantity) {
					t.Errorf("receive 0: want ErrInvalidQuantity, got %v", err)
				}
				if _, err := svc.IssueStock(ctx, MovementInput{ProductID: id, Quantity: -2}); !errors.Is(err, ErrInvalidQuantity) {
					t.Errorf("issue -2: want ErrInvalidQuantity, got %v", err)
				}
			},
			wantQty:   0,
			wantAfter: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			productID := newTestProduct(t, pool)
			tc.run(t, productID)

			if got := quantityOf(t, pool, productID); got != tc.wantQty {
				t.Errorf("cached quantity = %d, want %d", got, tc.wantQty)
			}

			rows, err := pool.Query(ctx,
				`SELECT quantity_after FROM stock_movements WHERE product_id = $1 ORDER BY id`, productID)
			if err != nil {
				t.Fatalf("read movements: %v", err)
			}
			defer rows.Close()
			var after []int
			for rows.Next() {
				var a int
				if err := rows.Scan(&a); err != nil {
					t.Fatalf("scan: %v", err)
				}
				after = append(after, a)
			}
			if len(after) != len(tc.wantAfter) {
				t.Fatalf("movement count = %d, want %d", len(after), len(tc.wantAfter))
			}
			for i := range after {
				if after[i] != tc.wantAfter[i] {
					t.Errorf("movement %d quantity_after = %d, want %d", i, after[i], tc.wantAfter[i])
				}
			}
		})
	}
}

// TestStockConcurrentIssue hammers one product from many goroutines; the row
// lock must prevent overselling and keep the balance exact.
func TestStockConcurrentIssue(t *testing.T) {
	pool := testPool(t)
	svc := &StockService{Pool: pool, Movements: &repository.MovementRepo{DB: pool}}
	ctx := context.Background()
	productID := newTestProduct(t, pool)

	mustOK(t, errOnly(svc.ReceiveStock(ctx, MovementInput{ProductID: productID, Quantity: 10})))

	const workers = 20 // each tries to take 1; only 10 can succeed
	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.IssueStock(ctx, MovementInput{ProductID: productID, Quantity: 1}); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if succeeded != 10 {
		t.Errorf("successful issues = %d, want exactly 10", succeeded)
	}
	if got := quantityOf(t, pool, productID); got != 0 {
		t.Errorf("final quantity = %d, want 0", got)
	}
}

// errOnly collapses a (movement, error) pair to just the error.
func errOnly(_ *models.StockMovement, err error) error { return err }

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("stock op failed: %v", err)
	}
}
