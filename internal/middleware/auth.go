// @ai-modified 2026-07-02 add session-user loading, auth and role middleware
package middleware

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"

	"mallstock/internal/models"
	"mallstock/internal/service"
)

type ctxKey int

const userKey ctxKey = 0

// SessionUserID is the session key holding the logged-in user's ID.
const SessionUserID = "userID"

// CurrentUser returns the logged-in user from the request context, or nil.
func CurrentUser(ctx context.Context) *models.User {
	u, _ := ctx.Value(userKey).(*models.User)
	return u
}

// WithUser returns a context carrying u — used by LoadUser and by tests.
func WithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// Auth bundles the dependencies the auth middleware needs.
type Auth struct {
	Session *scs.SessionManager
	Users   *service.UserService
}

// LoadUser resolves the session's user ID into a *models.User in context.
// Invalid or stale sessions are cleared, not treated as errors.
func (a *Auth) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := a.Session.GetInt64(r.Context(), SessionUserID)
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}
		u, err := a.Users.GetByID(r.Context(), id)
		if err != nil || !u.IsActive {
			a.Session.Remove(r.Context(), SessionUserID)
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), u)))
	})
}

// RequireAuth redirects anonymous requests to /login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentUser(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole gates a route group to the given roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := CurrentUser(r.Context())
			if u == nil || !allowed[u.Role] {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
