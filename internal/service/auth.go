// @ai-modified 2026-07-02 add auth service (login credential check)
package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"mallstock/internal/models"
	"mallstock/internal/repository"
)

// ErrInvalidCredentials is returned for a bad email/password combination or
// a deactivated account — deliberately indistinguishable to the caller.
var ErrInvalidCredentials = errors.New("invalid credentials")

// BcryptCost is the hashing cost for passwords (plan.md §11: ≥ 12).
const BcryptCost = 12

// AuthService authenticates users.
type AuthService struct {
	Users *repository.UserRepo
}

// Authenticate checks email+password and returns the user on success.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	u, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Burn a bcrypt comparison so timing doesn't reveal user existence.
			_ = bcrypt.CompareHashAndPassword(
				[]byte("$2a$12$C6UzMDM.H6dfI/f/IKcEeO1qTuDbaVQVKfwF6oXZKcwe2mDkbqxlG"), []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// HashPassword hashes a plaintext password with the project bcrypt cost.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}
