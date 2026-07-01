// @ai-modified 2026-07-02 add store CRUD handlers
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"mallstock/internal/service"
	"mallstock/internal/templates"
)

// StoresHandler serves store CRUD pages.
type StoresHandler struct {
	*Base
	Stores *service.StoreService
}

func urlID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil && id > 0
}

func (h *StoresHandler) List(w http.ResponseWriter, r *http.Request) {
	stores, err := h.Stores.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Stores"
	d.Data["Stores"] = stores
	h.Render(w, r, http.StatusOK, "stores/list.html", d)
}

func parseStoreInput(r *http.Request) service.StoreInput {
	return service.StoreInput{
		Name:         r.PostForm.Get("name"),
		UnitNumber:   r.PostForm.Get("unit_number"),
		Floor:        r.PostForm.Get("floor"),
		Category:     r.PostForm.Get("category"),
		ContactName:  r.PostForm.Get("contact_name"),
		ContactPhone: r.PostForm.Get("contact_phone"),
		ContactEmail: r.PostForm.Get("contact_email"),
		IsActive:     r.PostForm.Get("is_active") == "on",
	}
}

func storeFormFill(d *templates.Data, in service.StoreInput) {
	d.Form["name"] = in.Name
	d.Form["unit_number"] = in.UnitNumber
	d.Form["floor"] = in.Floor
	d.Form["category"] = in.Category
	d.Form["contact_name"] = in.ContactName
	d.Form["contact_phone"] = in.ContactPhone
	d.Form["contact_email"] = in.ContactEmail
	if in.IsActive {
		d.Form["is_active"] = "on"
	}
}

func (h *StoresHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	d := h.NewData(r)
	d.Title = "New store"
	d.Form["is_active"] = "on"
	h.Render(w, r, http.StatusOK, "stores/form.html", d)
}

func (h *StoresHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseStoreInput(r)
	if _, err := h.Stores.Create(r.Context(), in); err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			d := h.NewData(r)
			d.Title = "New store"
			d.Errors = verr.Fields
			storeFormFill(d, in)
			h.Render(w, r, http.StatusUnprocessableEntity, "stores/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Store created.")
	http.Redirect(w, r, "/stores", http.StatusSeeOther)
}

func (h *StoresHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	st, err := h.Stores.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Edit store"
	d.Data["Store"] = st
	storeFormFill(d, service.StoreInput{
		Name: st.Name, UnitNumber: st.UnitNumber, Floor: st.Floor, Category: st.Category,
		ContactName: st.ContactName, ContactPhone: st.ContactPhone, ContactEmail: st.ContactEmail,
		IsActive: st.IsActive,
	})
	h.Render(w, r, http.StatusOK, "stores/form.html", d)
}

func (h *StoresHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseStoreInput(r)
	if _, err := h.Stores.Update(r.Context(), id, in); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			st, gerr := h.Stores.GetByID(r.Context(), id)
			if gerr != nil {
				h.ServerError(w, r, gerr)
				return
			}
			d := h.NewData(r)
			d.Title = "Edit store"
			d.Data["Store"] = st
			d.Errors = verr.Fields
			storeFormFill(d, in)
			h.Render(w, r, http.StatusUnprocessableEntity, "stores/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Store updated.")
	http.Redirect(w, r, "/stores", http.StatusSeeOther)
}

func (h *StoresHandler) SetActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := urlID(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if err := h.Stores.SetActive(r.Context(), id, active); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			h.ServerError(w, r, err)
			return
		}
		if active {
			h.Flash(r, "success", "Store activated.")
		} else {
			h.Flash(r, "success", "Store deactivated.")
		}
		http.Redirect(w, r, "/stores", http.StatusSeeOther)
	}
}
