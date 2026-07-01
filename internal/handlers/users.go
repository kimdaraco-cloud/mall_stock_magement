// @ai-modified 2026-07-02 add admin user management handlers
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"mallstock/internal/middleware"
	"mallstock/internal/models"
	"mallstock/internal/repository"
	"mallstock/internal/service"
	"mallstock/internal/templates"
)

// UsersHandler serves the admin user management pages.
type UsersHandler struct {
	*Base
	Users  *service.UserService
	Stores *repository.StoreRepo
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Users"
	d.Data["Users"] = users
	h.Render(w, r, http.StatusOK, "users/list.html", d)
}

func (h *UsersHandler) formData(r *http.Request, u *models.User) (*templates.Data, error) {
	stores, err := h.Stores.ListActive(r.Context())
	if err != nil {
		return nil, err
	}
	d := h.NewData(r)
	d.Data["Stores"] = stores
	d.Data["Roles"] = []string{models.RoleAdmin, models.RoleManager, models.RoleStaff}
	if u != nil {
		d.Data["EditUser"] = u
	}
	return d, nil
}

func (h *UsersHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	d, err := h.formData(r, nil)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d.Title = "New user"
	d.Form["role"] = models.RoleStaff
	d.Form["is_active"] = "on"
	h.Render(w, r, http.StatusOK, "users/form.html", d)
}

func parseUserInput(r *http.Request) service.UserInput {
	in := service.UserInput{
		Email:    r.PostForm.Get("email"),
		FullName: r.PostForm.Get("full_name"),
		Password: r.PostForm.Get("password"),
		Role:     r.PostForm.Get("role"),
		IsActive: r.PostForm.Get("is_active") == "on",
	}
	if sid, err := strconv.ParseInt(r.PostForm.Get("store_id"), 10, 64); err == nil && sid > 0 {
		in.StoreID = &sid
	}
	return in
}

// rerenderForm shows the form again with validation errors and sticky values.
func (h *UsersHandler) rerenderForm(w http.ResponseWriter, r *http.Request, u *models.User, in service.UserInput, verr *service.ValidationError, title string) {
	d, err := h.formData(r, u)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d.Title = title
	d.Errors = verr.Fields
	d.Form["email"] = in.Email
	d.Form["full_name"] = in.FullName
	d.Form["role"] = in.Role
	if in.StoreID != nil {
		d.Form["store_id"] = strconv.FormatInt(*in.StoreID, 10)
	}
	if in.IsActive {
		d.Form["is_active"] = "on"
	}
	h.Render(w, r, http.StatusUnprocessableEntity, "users/form.html", d)
}

func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseUserInput(r)
	if _, err := h.Users.Create(r.Context(), in); err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			h.rerenderForm(w, r, nil, in, verr, "New user")
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "User created.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *UsersHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u, err := h.Users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	d, err := h.formData(r, u)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d.Title = "Edit user"
	d.Form["email"] = u.Email
	d.Form["full_name"] = u.FullName
	d.Form["role"] = u.Role
	if u.StoreID != nil {
		d.Form["store_id"] = strconv.FormatInt(*u.StoreID, 10)
	}
	if u.IsActive {
		d.Form["is_active"] = "on"
	}
	h.Render(w, r, http.StatusOK, "users/form.html", d)
}

func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseUserInput(r)
	u, err := h.Users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	if _, err := h.Users.Update(r.Context(), id, in); err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			h.rerenderForm(w, r, u, in, verr, "Edit user")
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "User updated.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *UsersHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if cu := middleware.CurrentUser(r.Context()); cu != nil && cu.ID == id {
		h.Flash(r, "error", "You cannot deactivate your own account.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	if err := h.Users.SetActive(r.Context(), id, false); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "User deactivated.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *UsersHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.Users.SetActive(r.Context(), id, true); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "User activated.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
