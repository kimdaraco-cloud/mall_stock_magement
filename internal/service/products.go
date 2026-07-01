// @ai-modified 2026-07-02 add product service with price/SKU validation
package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"mallstock/internal/models"
	"mallstock/internal/repository"
)

// priceRe accepts non-negative decimals with up to 2 fraction digits,
// matching NUMERIC(12,2).
var priceRe = regexp.MustCompile(`^\d{1,10}(\.\d{1,2})?$`)

// ProductService owns product catalog rules.
type ProductService struct {
	Products *repository.ProductRepo
}

// ProductInput is the form payload for a product. Prices are decimal strings.
type ProductInput struct {
	SKU          string
	Barcode      string
	Name         string
	Description  string
	CategoryID   *int64
	SupplierID   *int64
	StoreID      int64
	CostPrice    string
	SellingPrice string
	ReorderLevel int
	Unit         string
	IsActive     bool
}

func (s *ProductService) validate(ctx context.Context, in *ProductInput, excludeID int64) error {
	v := NewValidation()
	in.SKU = strings.TrimSpace(in.SKU)
	in.Name = strings.TrimSpace(in.Name)
	in.CostPrice = strings.TrimSpace(in.CostPrice)
	in.SellingPrice = strings.TrimSpace(in.SellingPrice)
	in.Unit = strings.TrimSpace(in.Unit)

	if in.SKU == "" {
		v.Fields["sku"] = "SKU is required."
	} else {
		taken, err := s.Products.SKUTaken(ctx, in.SKU, excludeID)
		if err != nil {
			return fmt.Errorf("validate product: %w", err)
		}
		if taken {
			v.Fields["sku"] = "This SKU is already in use."
		}
	}
	if in.Name == "" {
		v.Fields["name"] = "Name is required."
	}
	if in.StoreID <= 0 {
		v.Fields["store_id"] = "Choose a store."
	}
	if in.CostPrice == "" {
		in.CostPrice = "0"
	}
	if in.SellingPrice == "" {
		in.SellingPrice = "0"
	}
	if !priceRe.MatchString(in.CostPrice) {
		v.Fields["cost_price"] = "Enter a valid non-negative price (max 2 decimals)."
	}
	if !priceRe.MatchString(in.SellingPrice) {
		v.Fields["selling_price"] = "Enter a valid non-negative price (max 2 decimals)."
	}
	if in.ReorderLevel < 0 {
		v.Fields["reorder_level"] = "Reorder level cannot be negative."
	}
	if in.Unit == "" {
		in.Unit = "pcs"
	}
	if v.Any() {
		return v
	}
	return nil
}

func (in *ProductInput) apply(p *models.Product) {
	p.SKU = in.SKU
	p.Barcode = strings.TrimSpace(in.Barcode)
	p.Name = in.Name
	p.Description = strings.TrimSpace(in.Description)
	p.CategoryID = in.CategoryID
	p.SupplierID = in.SupplierID
	p.StoreID = in.StoreID
	p.CostPrice = in.CostPrice
	p.SellingPrice = in.SellingPrice
	p.ReorderLevel = in.ReorderLevel
	p.Unit = in.Unit
	p.IsActive = in.IsActive
}

func (s *ProductService) List(ctx context.Context, f repository.ProductFilter) ([]models.Product, int, error) {
	return s.Products.List(ctx, f)
}

func (s *ProductService) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	p, err := s.Products.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

// Create adds a product. New products start at quantity 0 — stock arrives
// only through stock movements (see the stock invariant in CLAUDE.md).
func (s *ProductService) Create(ctx context.Context, in ProductInput) (*models.Product, error) {
	if err := s.validate(ctx, &in, 0); err != nil {
		return nil, err
	}
	p := &models.Product{Quantity: 0}
	in.apply(p)
	if err := s.Products.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) Update(ctx context.Context, id int64, in ProductInput) (*models.Product, error) {
	p, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, &in, id); err != nil {
		return nil, err
	}
	in.apply(p)
	if err := s.Products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Deactivate soft-deletes a product (movement history must stay auditable).
func (s *ProductService) Deactivate(ctx context.Context, id int64) error {
	if err := s.Products.SetActive(ctx, id, false); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
