// @ai-modified 2026-07-02 add User and Store models with role constants
package models

import "time"

// Roles. Keep in sync with the users.role column comment.
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleStaff   = "staff"
)

// ValidRole reports whether r is a known role.
func ValidRole(r string) bool {
	return r == RoleAdmin || r == RoleManager || r == RoleStaff
}

// User is a person who can log in.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	FullName     string
	Role         string
	StoreID      *int64
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time

	StoreName string // joined display field, not a column
}

// IsAdmin reports whether the user has the admin role.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// CanManageCatalog reports whether the user may edit products/stores/etc.
func (u *User) CanManageCatalog() bool {
	return u.Role == RoleAdmin || u.Role == RoleManager
}

// Store is a shop/unit in the mall.
type Store struct {
	ID           int64
	Name         string
	UnitNumber   string
	Floor        string
	Category     string
	ContactName  string
	ContactPhone string
	ContactEmail string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
