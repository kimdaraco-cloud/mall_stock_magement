// @ai-modified 2026-07-02 add auth/role middleware tests
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mallstock/internal/models"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.User
		wantStatus int
		wantLoc    string
	}{
		{"anonymous is redirected to login", nil, http.StatusSeeOther, "/login"},
		{"logged-in user passes", &models.User{ID: 1, Role: models.RoleStaff}, http.StatusOK, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products", nil)
			if tc.user != nil {
				req = req.WithContext(WithUser(req.Context(), tc.user))
			}
			rec := httptest.NewRecorder()
			RequireAuth(okHandler()).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantLoc != "" && rec.Header().Get("Location") != tc.wantLoc {
				t.Errorf("location = %q, want %q", rec.Header().Get("Location"), tc.wantLoc)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.User
		roles      []string
		wantStatus int
	}{
		{"admin allowed on admin route", &models.User{Role: models.RoleAdmin}, []string{models.RoleAdmin}, http.StatusOK},
		{"staff forbidden on admin route", &models.User{Role: models.RoleStaff}, []string{models.RoleAdmin}, http.StatusForbidden},
		{"manager allowed on manage route", &models.User{Role: models.RoleManager}, []string{models.RoleAdmin, models.RoleManager}, http.StatusOK},
		{"staff forbidden on manage route", &models.User{Role: models.RoleStaff}, []string{models.RoleAdmin, models.RoleManager}, http.StatusForbidden},
		{"no user is forbidden", nil, []string{models.RoleAdmin}, http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			if tc.user != nil {
				req = req.WithContext(WithUser(req.Context(), tc.user))
			}
			rec := httptest.NewRecorder()
			RequireRole(tc.roles...)(okHandler()).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
