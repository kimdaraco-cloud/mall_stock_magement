// @ai-modified 2026-07-02 add store/category/supplier services with validation
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"mallstock/internal/models"
	"mallstock/internal/repository"
)

// StoreService owns store rules.
type StoreService struct {
	Stores *repository.StoreRepo
}

// StoreInput is the form payload for a store.
type StoreInput struct {
	Name         string
	UnitNumber   string
	Floor        string
	Category     string
	ContactName  string
	ContactPhone string
	ContactEmail string
	IsActive     bool
}

func (in *StoreInput) validate() error {
	v := NewValidation()
	if strings.TrimSpace(in.Name) == "" {
		v.Fields["name"] = "Name is required."
	}
	if in.ContactEmail != "" {
		if _, err := mail.ParseAddress(in.ContactEmail); err != nil {
			v.Fields["contact_email"] = "Enter a valid email address."
		}
	}
	if v.Any() {
		return v
	}
	return nil
}

func (in *StoreInput) apply(s *models.Store) {
	s.Name = strings.TrimSpace(in.Name)
	s.UnitNumber = strings.TrimSpace(in.UnitNumber)
	s.Floor = strings.TrimSpace(in.Floor)
	s.Category = strings.TrimSpace(in.Category)
	s.ContactName = strings.TrimSpace(in.ContactName)
	s.ContactPhone = strings.TrimSpace(in.ContactPhone)
	s.ContactEmail = strings.TrimSpace(in.ContactEmail)
	s.IsActive = in.IsActive
}

func (s *StoreService) List(ctx context.Context) ([]models.Store, error) {
	return s.Stores.List(ctx)
}

func (s *StoreService) ListActive(ctx context.Context) ([]models.Store, error) {
	return s.Stores.ListActive(ctx)
}

func (s *StoreService) GetByID(ctx context.Context, id int64) (*models.Store, error) {
	st, err := s.Stores.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return st, nil
}

func (s *StoreService) Create(ctx context.Context, in StoreInput) (*models.Store, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	st := &models.Store{}
	in.apply(st)
	if err := s.Stores.Create(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StoreService) Update(ctx context.Context, id int64, in StoreInput) (*models.Store, error) {
	st, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	in.apply(st)
	if err := s.Stores.Update(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// SetActive toggles a store; stores are never hard-deleted (plan.md §6).
func (s *StoreService) SetActive(ctx context.Context, id int64, active bool) error {
	st, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	st.IsActive = active
	return s.Stores.Update(ctx, st)
}

// CategoryService owns category rules.
type CategoryService struct {
	Categories *repository.CategoryRepo
}

// CategoryInput is the form payload for a category.
type CategoryInput struct {
	Name        string
	Description string
}

func (s *CategoryService) List(ctx context.Context) ([]models.Category, error) {
	return s.Categories.List(ctx)
}

func (s *CategoryService) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	c, err := s.Categories.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Create(ctx context.Context, in CategoryInput) (*models.Category, error) {
	if strings.TrimSpace(in.Name) == "" {
		v := NewValidation()
		v.Fields["name"] = "Name is required."
		return nil, v
	}
	c := &models.Category{Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description)}
	if err := s.Categories.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Update(ctx context.Context, id int64, in CategoryInput) (*models.Category, error) {
	c, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		v := NewValidation()
		v.Fields["name"] = "Name is required."
		return nil, v
	}
	c.Name = strings.TrimSpace(in.Name)
	c.Description = strings.TrimSpace(in.Description)
	if err := s.Categories.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	if err := s.Categories.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

// SupplierService owns supplier rules.
type SupplierService struct {
	Suppliers *repository.SupplierRepo
}

// SupplierInput is the form payload for a supplier.
type SupplierInput struct {
	Name          string
	ContactPerson string
	Email         string
	Phone         string
	Address       string
}

func (in *SupplierInput) validate() error {
	v := NewValidation()
	if strings.TrimSpace(in.Name) == "" {
		v.Fields["name"] = "Name is required."
	}
	if in.Email != "" {
		if _, err := mail.ParseAddress(in.Email); err != nil {
			v.Fields["email"] = "Enter a valid email address."
		}
	}
	if v.Any() {
		return v
	}
	return nil
}

func (in *SupplierInput) apply(s *models.Supplier) {
	s.Name = strings.TrimSpace(in.Name)
	s.ContactPerson = strings.TrimSpace(in.ContactPerson)
	s.Email = strings.TrimSpace(in.Email)
	s.Phone = strings.TrimSpace(in.Phone)
	s.Address = strings.TrimSpace(in.Address)
}

func (s *SupplierService) List(ctx context.Context) ([]models.Supplier, error) {
	return s.Suppliers.List(ctx)
}

func (s *SupplierService) GetByID(ctx context.Context, id int64) (*models.Supplier, error) {
	sup, err := s.Suppliers.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sup, nil
}

func (s *SupplierService) Create(ctx context.Context, in SupplierInput) (*models.Supplier, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	sup := &models.Supplier{}
	in.apply(sup)
	if err := s.Suppliers.Create(ctx, sup); err != nil {
		return nil, err
	}
	return sup, nil
}

func (s *SupplierService) Update(ctx context.Context, id int64, in SupplierInput) (*models.Supplier, error) {
	sup, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	in.apply(sup)
	if err := s.Suppliers.Update(ctx, sup); err != nil {
		return nil, err
	}
	return sup, nil
}

func (s *SupplierService) Delete(ctx context.Context, id int64) error {
	if err := s.Suppliers.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete supplier: %w", err)
	}
	return nil
}
