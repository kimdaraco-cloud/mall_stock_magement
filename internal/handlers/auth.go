// @ai-modified 2026-07-02 add login/logout handlers
package handlers

import (
	"errors"
	"net/http"

	"mallstock/internal/middleware"
	"mallstock/internal/service"
)

// AuthHandler serves login and logout.
type AuthHandler struct {
	*Base
	Auth *service.AuthService
}

func (h *AuthHandler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if middleware.CurrentUser(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	d := h.NewData(r)
	d.Title = "Log in"
	h.Render(w, r, http.StatusOK, "login.html", d)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")

	u, err := h.Auth.Authenticate(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			d := h.NewData(r)
			d.Title = "Log in"
			d.Form["email"] = email
			d.Errors["login"] = "Invalid email or password."
			h.Render(w, r, http.StatusUnprocessableEntity, "login.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}

	// Rotate the session token on privilege change (session fixation defence).
	if err := h.Session.RenewToken(r.Context()); err != nil {
		h.ServerError(w, r, err)
		return
	}
	h.Session.Put(r.Context(), middleware.SessionUserID, u.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.Session.Destroy(r.Context()); err != nil {
		h.ServerError(w, r, err)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
