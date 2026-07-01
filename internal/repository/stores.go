// @ai-modified 2026-07-02 add store repository (SQL only)
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mallstock/internal/models"
)

// StoreRepo persists stores.
type StoreRepo struct {
	DB DB
}

const storeCols = `id, name, COALESCE(unit_number,''), COALESCE(floor,''), COALESCE(category,''),
	COALESCE(contact_name,''), COALESCE(contact_phone,''), COALESCE(contact_email,''),
	is_active, created_at, updated_at`

func scanStore(row pgx.Row) (*models.Store, error) {
	var s models.Store
	err := row.Scan(&s.ID, &s.Name, &s.UnitNumber, &s.Floor, &s.Category,
		&s.ContactName, &s.ContactPhone, &s.ContactEmail,
		&s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *StoreRepo) List(ctx context.Context) ([]models.Store, error) {
	rows, err := r.DB.Query(ctx, `SELECT `+storeCols+` FROM stores ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list stores: %w", err)
	}
	defer rows.Close()

	var stores []models.Store
	for rows.Next() {
		s, err := scanStore(rows)
		if err != nil {
			return nil, fmt.Errorf("list stores scan: %w", err)
		}
		stores = append(stores, *s)
	}
	return stores, rows.Err()
}

func (r *StoreRepo) ListActive(ctx context.Context) ([]models.Store, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT `+storeCols+` FROM stores WHERE is_active ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list active stores: %w", err)
	}
	defer rows.Close()

	var stores []models.Store
	for rows.Next() {
		s, err := scanStore(rows)
		if err != nil {
			return nil, fmt.Errorf("list active stores scan: %w", err)
		}
		stores = append(stores, *s)
	}
	return stores, rows.Err()
}

func (r *StoreRepo) GetByID(ctx context.Context, id int64) (*models.Store, error) {
	s, err := scanStore(r.DB.QueryRow(ctx,
		`SELECT `+storeCols+` FROM stores WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get store by id: %w", err)
	}
	return s, nil
}

func (r *StoreRepo) Create(ctx context.Context, s *models.Store) error {
	err := r.DB.QueryRow(ctx,
		`INSERT INTO stores (name, unit_number, floor, category, contact_name, contact_phone, contact_email, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`,
		s.Name, s.UnitNumber, s.Floor, s.Category, s.ContactName, s.ContactPhone, s.ContactEmail, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}
	return nil
}

func (r *StoreRepo) Update(ctx context.Context, s *models.Store) error {
	tag, err := r.DB.Exec(ctx,
		`UPDATE stores SET name = $1, unit_number = $2, floor = $3, category = $4,
		 contact_name = $5, contact_phone = $6, contact_email = $7, is_active = $8,
		 updated_at = now() WHERE id = $9`,
		s.Name, s.UnitNumber, s.Floor, s.Category, s.ContactName, s.ContactPhone,
		s.ContactEmail, s.IsActive, s.ID)
	if err != nil {
		return fmt.Errorf("update store: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update store: %w", ErrNotFound)
	}
	return nil
}

// HasProducts reports whether any product references the store.
func (r *StoreRepo) HasProducts(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM products WHERE store_id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store has products: %w", err)
	}
	return exists, nil
}
