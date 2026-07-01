// @ai-modified 2026-07-02 add user repository (SQL only)
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mallstock/internal/models"
)

// UserRepo persists users.
type UserRepo struct {
	DB DB
}

const userCols = `u.id, u.email, u.password_hash, u.full_name, u.role, u.store_id,
	u.is_active, u.created_at, u.updated_at, COALESCE(s.name, '')`

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role,
		&u.StoreID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.StoreName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u, err := scanUser(r.DB.QueryRow(ctx,
		`SELECT `+userCols+` FROM users u LEFT JOIN stores s ON s.id = u.store_id
		 WHERE u.email = $1`, email))
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u, err := scanUser(r.DB.QueryRow(ctx,
		`SELECT `+userCols+` FROM users u LEFT JOIN stores s ON s.id = u.store_id
		 WHERE u.id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT `+userCols+` FROM users u LEFT JOIN stores s ON s.id = u.store_id
		 ORDER BY u.created_at, u.id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("list users scan: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *UserRepo) Create(ctx context.Context, u *models.User) error {
	err := r.DB.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, full_name, role, store_id, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		u.Email, u.PasswordHash, u.FullName, u.Role, u.StoreID, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) Update(ctx context.Context, u *models.User) error {
	tag, err := r.DB.Exec(ctx,
		`UPDATE users SET email = $1, password_hash = $2, full_name = $3, role = $4,
		 store_id = $5, is_active = $6, updated_at = now() WHERE id = $7`,
		u.Email, u.PasswordHash, u.FullName, u.Role, u.StoreID, u.IsActive, u.ID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update user: %w", ErrNotFound)
	}
	return nil
}

// EmailTaken reports whether email is used by a user other than excludeID.
func (r *UserRepo) EmailTaken(ctx context.Context, email string, excludeID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND id <> $2)`,
		email, excludeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("email taken: %w", err)
	}
	return exists, nil
}
