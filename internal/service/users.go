// @ai-modified 2026-07-02 add user management service
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

// UserService owns user management rules.
type UserService struct {
	Users *repository.UserRepo
}

// UserInput is the form payload for creating/updating a user.
type UserInput struct {
	Email    string
	FullName string
	Password string // blank on update = keep current password
	Role     string
	StoreID  *int64
	IsActive bool
}

func (s *UserService) validate(ctx context.Context, in UserInput, existingID int64, requirePassword bool) error {
	v := NewValidation()
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))

	if in.Email == "" {
		v.Fields["email"] = "Email is required."
	} else if _, err := mail.ParseAddress(in.Email); err != nil {
		v.Fields["email"] = "Enter a valid email address."
	} else {
		taken, err := s.Users.EmailTaken(ctx, in.Email, existingID)
		if err != nil {
			return fmt.Errorf("validate user: %w", err)
		}
		if taken {
			v.Fields["email"] = "This email is already in use."
		}
	}
	if strings.TrimSpace(in.FullName) == "" {
		v.Fields["full_name"] = "Full name is required."
	}
	if requirePassword && len(in.Password) < 8 {
		v.Fields["password"] = "Password must be at least 8 characters."
	}
	if !requirePassword && in.Password != "" && len(in.Password) < 8 {
		v.Fields["password"] = "Password must be at least 8 characters."
	}
	if !models.ValidRole(in.Role) {
		v.Fields["role"] = "Choose a valid role."
	}
	if v.Any() {
		return v
	}
	return nil
}

// Create adds a new user with a hashed password.
func (s *UserService) Create(ctx context.Context, in UserInput) (*models.User, error) {
	if err := s.validate(ctx, in, 0, true); err != nil {
		return nil, err
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	u := &models.User{
		Email:        strings.TrimSpace(strings.ToLower(in.Email)),
		PasswordHash: hash,
		FullName:     strings.TrimSpace(in.FullName),
		Role:         in.Role,
		StoreID:      in.StoreID,
		IsActive:     in.IsActive,
	}
	if err := s.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Update modifies an existing user; password only changes when provided.
func (s *UserService) Update(ctx context.Context, id int64, in UserInput) (*models.User, error) {
	u, err := s.Users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.validate(ctx, in, id, false); err != nil {
		return nil, err
	}
	u.Email = strings.TrimSpace(strings.ToLower(in.Email))
	u.FullName = strings.TrimSpace(in.FullName)
	u.Role = in.Role
	u.StoreID = in.StoreID
	u.IsActive = in.IsActive
	if in.Password != "" {
		hash, err := HashPassword(in.Password)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
		u.PasswordHash = hash
	}
	if err := s.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// SetActive toggles a user's active flag (deactivate instead of delete).
func (s *UserService) SetActive(ctx context.Context, id int64, active bool) error {
	u, err := s.Users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	u.IsActive = active
	return s.Users.Update(ctx, u)
}

func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	return s.Users.List(ctx)
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u, err := s.Users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}
