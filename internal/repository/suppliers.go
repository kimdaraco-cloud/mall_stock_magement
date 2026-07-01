// @ai-modified 2026-07-02 add supplier repository (SQL only)
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mallstock/internal/models"
)

// SupplierRepo persists suppliers.
type SupplierRepo struct {
	DB DB
}

const supplierCols = `id, name, COALESCE(contact_person,''), COALESCE(email,''),
	COALESCE(phone,''), COALESCE(address,''), created_at, updated_at`

func scanSupplier(row pgx.Row) (*models.Supplier, error) {
	var s models.Supplier
	err := row.Scan(&s.ID, &s.Name, &s.ContactPerson, &s.Email, &s.Phone,
		&s.Address, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *SupplierRepo) List(ctx context.Context) ([]models.Supplier, error) {
	rows, err := r.DB.Query(ctx, `SELECT `+supplierCols+` FROM suppliers ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()

	var sups []models.Supplier
	for rows.Next() {
		s, err := scanSupplier(rows)
		if err != nil {
			return nil, fmt.Errorf("list suppliers scan: %w", err)
		}
		sups = append(sups, *s)
	}
	return sups, rows.Err()
}

func (r *SupplierRepo) GetByID(ctx context.Context, id int64) (*models.Supplier, error) {
	s, err := scanSupplier(r.DB.QueryRow(ctx,
		`SELECT `+supplierCols+` FROM suppliers WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get supplier: %w", err)
	}
	return s, nil
}

func (r *SupplierRepo) Create(ctx context.Context, s *models.Supplier) error {
	err := r.DB.QueryRow(ctx,
		`INSERT INTO suppliers (name, contact_person, email, phone, address)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		s.Name, s.ContactPerson, s.Email, s.Phone, s.Address,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create supplier: %w", err)
	}
	return nil
}

func (r *SupplierRepo) Update(ctx context.Context, s *models.Supplier) error {
	tag, err := r.DB.Exec(ctx,
		`UPDATE suppliers SET name = $1, contact_person = $2, email = $3, phone = $4,
		 address = $5, updated_at = now() WHERE id = $6`,
		s.Name, s.ContactPerson, s.Email, s.Phone, s.Address, s.ID)
	if err != nil {
		return fmt.Errorf("update supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update supplier: %w", ErrNotFound)
	}
	return nil
}

func (r *SupplierRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.DB.Exec(ctx, `DELETE FROM suppliers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete supplier: %w", ErrNotFound)
	}
	return nil
}
