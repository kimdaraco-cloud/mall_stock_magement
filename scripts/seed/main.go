// @ai-modified 2026-07-02 seed realistic sample data (users, catalog, movements)
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"mallstock/internal/config"
	"mallstock/internal/database"
	"mallstock/internal/models"
	"mallstock/internal/repository"
	"mallstock/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	adminID, created, err := seedUser(ctx, pool, "admin@mall.local", "admin123", "Administrator", models.RoleAdmin)
	if err != nil {
		return err
	}
	if created {
		fmt.Println("seed: created admin@mall.local / admin123 — change it immediately")
	}
	if _, _, err := seedUser(ctx, pool, "manager@mall.local", "manager123", "Mia Manager", models.RoleManager); err != nil {
		return err
	}
	if _, _, err := seedUser(ctx, pool, "staff@mall.local", "staff1234", "Sam Staff", models.RoleStaff); err != nil {
		return err
	}

	// Sample data is only loaded into an empty catalog, so re-running seed
	// never duplicates or disturbs real data.
	var productCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM products`).Scan(&productCount); err != nil {
		return fmt.Errorf("count products: %w", err)
	}
	if productCount > 0 {
		fmt.Println("seed: products already exist — skipping sample data")
		return nil
	}

	fmt.Println("seed: loading sample data...")

	stores := []models.Store{
		{Name: "Tech World", UnitNumber: "A-101", Floor: "1", Category: "Electronics", ContactName: "Tina Ong", ContactPhone: "555-0101", IsActive: true},
		{Name: "Fashion Hub", UnitNumber: "B-204", Floor: "2", Category: "Apparel", ContactName: "Farid Khan", ContactPhone: "555-0102", IsActive: true},
		{Name: "Fresh Mart", UnitNumber: "G-010", Floor: "G", Category: "Grocery", ContactName: "Grace Lim", ContactPhone: "555-0103", IsActive: true},
	}
	storeRepo := &repository.StoreRepo{DB: pool}
	for i := range stores {
		if err := storeRepo.Create(ctx, &stores[i]); err != nil {
			return err
		}
	}

	categories := []models.Category{
		{Name: "Electronics", Description: "Gadgets, cables and devices"},
		{Name: "Apparel", Description: "Clothing and accessories"},
		{Name: "Grocery", Description: "Food and daily essentials"},
	}
	catRepo := &repository.CategoryRepo{DB: pool}
	for i := range categories {
		if err := catRepo.Create(ctx, &categories[i]); err != nil {
			return err
		}
	}

	suppliers := []models.Supplier{
		{Name: "Acme Wholesale", ContactPerson: "Joe Acme", Email: "sales@acme.test", Phone: "555-0201"},
		{Name: "Global Textiles", ContactPerson: "Nina Weave", Email: "orders@gtex.test", Phone: "555-0202"},
		{Name: "FarmDirect Co", ContactPerson: "Omar Field", Email: "supply@farmdirect.test", Phone: "555-0203"},
	}
	supRepo := &repository.SupplierRepo{DB: pool}
	for i := range suppliers {
		if err := supRepo.Create(ctx, &suppliers[i]); err != nil {
			return err
		}
	}

	type seedProduct struct {
		p       models.Product
		initial int // received via a real stock-in movement
		sold    int // issued via a real stock-out movement
	}
	sp := func(sku, name string, store, cat, sup int, cost, sell string, reorder int, unit string, initial, sold int) seedProduct {
		return seedProduct{
			p: models.Product{
				SKU: sku, Name: name,
				StoreID: stores[store].ID, CategoryID: &categories[cat].ID, SupplierID: &suppliers[sup].ID,
				CostPrice: cost, SellingPrice: sell, ReorderLevel: reorder, Unit: unit, IsActive: true,
			},
			initial: initial, sold: sold,
		}
	}
	seedProducts := []seedProduct{
		sp("TW-001", "USB-C Cable 1m", 0, 0, 0, "2.50", "7.99", 20, "pcs", 100, 35),
		sp("TW-002", "Wireless Mouse", 0, 0, 0, "8.00", "19.99", 10, "pcs", 50, 42),
		sp("TW-003", "27\" Monitor", 0, 0, 0, "120.00", "199.00", 5, "pcs", 12, 4),
		sp("FH-001", "Cotton T-Shirt (M)", 1, 1, 1, "4.50", "14.90", 15, "pcs", 80, 30),
		sp("FH-002", "Denim Jeans (32)", 1, 1, 1, "12.00", "39.90", 10, "pcs", 40, 36),
		sp("FM-001", "Mineral Water 1.5L", 2, 2, 2, "0.40", "1.20", 50, "bottle", 300, 260),
		sp("FM-002", "Jasmine Rice 5kg", 2, 2, 2, "6.00", "9.50", 20, "bag", 60, 15),
		sp("FM-003", "Olive Oil 500ml", 2, 2, 2, "3.80", "7.50", 10, "bottle", 25, 22),
	}

	prodRepo := &repository.ProductRepo{DB: pool}
	stock := &service.StockService{Pool: pool, Movements: &repository.MovementRepo{DB: pool}}
	for i := range seedProducts {
		s := &seedProducts[i]
		if err := prodRepo.Create(ctx, &s.p); err != nil {
			return err
		}
		// Movements go through the stock service so the invariant holds.
		if s.initial > 0 {
			if _, err := stock.ReceiveStock(ctx, service.MovementInput{
				ProductID: s.p.ID, Quantity: s.initial, Reference: "PO-SEED-1",
				Notes: "opening stock", UserID: adminID,
			}); err != nil {
				return err
			}
		}
		if s.sold > 0 {
			if _, err := stock.IssueStock(ctx, service.MovementInput{
				ProductID: s.p.ID, Quantity: s.sold, Reference: "SALES-SEED",
				Notes: "sample sales", UserID: adminID,
			}); err != nil {
				return err
			}
		}
	}

	fmt.Printf("seed: %d stores, %d categories, %d suppliers, %d products with movement history\n",
		len(stores), len(categories), len(suppliers), len(seedProducts))
	fmt.Println("seed: logins — admin@mall.local/admin123, manager@mall.local/manager123, staff@mall.local/staff1234")
	return nil
}

// seedUser inserts a user if the email is free; returns its id.
func seedUser(ctx context.Context, pool *pgxpool.Pool, email, password, name, role string) (int64, bool, error) {
	hash, err := service.HashPassword(password)
	if err != nil {
		return 0, false, err
	}
	var id int64
	err = pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, full_name, role, is_active)
		 VALUES ($1, $2, $3, $4, TRUE)
		 ON CONFLICT (email) DO NOTHING
		 RETURNING id`, email, hash, name, role).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	// Row already existed; fetch its id.
	if err2 := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id); err2 != nil {
		return 0, false, fmt.Errorf("seed user %s: %w", email, err2)
	}
	return id, false, nil
}
