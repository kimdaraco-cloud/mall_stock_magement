// @ai-modified 2026-07-02 add category repository (SQL only)
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mallstock/internal/models"
)

// CategoryRepo persists categories.
type CategoryRepo struct {
	DB DB
}

func scanCategory(row pgx.Row) (*models.Category, error) {
	var c models.Category
	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepo) List(ctx context.Context) ([]models.Category, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT id, name, COALESCE(description,''), created_at FROM categories ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("list categories scan: %w", err)
		}
		cats = append(cats, *c)
	}
	return cats, rows.Err()
}

func (r *CategoryRepo) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	c, err := scanCategory(r.DB.QueryRow(ctx,
		`SELECT id, name, COALESCE(description,''), created_at FROM categories WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	return c, nil
}

func (r *CategoryRepo) Create(ctx context.Context, c *models.Category) error {
	err := r.DB.QueryRow(ctx,
		`INSERT INTO categories (name, description) VALUES ($1, $2) RETURNING id, created_at`,
		c.Name, c.Description).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *CategoryRepo) Update(ctx context.Context, c *models.Category) error {
	tag, err := r.DB.Exec(ctx,
		`UPDATE categories SET name = $1, description = $2 WHERE id = $3`,
		c.Name, c.Description, c.ID)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update category: %w", ErrNotFound)
	}
	return nil
}

func (r *CategoryRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.DB.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete category: %w", ErrNotFound)
	}
	return nil
}
