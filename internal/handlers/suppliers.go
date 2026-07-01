// @ai-modified 2026-07-02 add supplier CRUD handlers
package handlers

import (
	"errors"
	"net/http"

	"mallstock/internal/service"
	"mallstock/internal/templates"
)

// SuppliersHandler serves supplier CRUD pages.
type SuppliersHandler struct {
	*Base
	Suppliers *service.SupplierService
}

func (h *SuppliersHandler) List(w http.ResponseWriter, r *http.Request) {
	sups, err := h.Suppliers.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Suppliers"
	d.Data["Suppliers"] = sups
	h.Render(w, r, http.StatusOK, "suppliers/list.html", d)
}

func parseSupplierInput(r *http.Request) service.SupplierInput {
	return service.SupplierInput{
		Name:          r.PostForm.Get("name"),
		ContactPerson: r.PostForm.Get("contact_person"),
		Email:         r.PostForm.Get("email"),
		Phone:         r.PostForm.Get("phone"),
		Address:       r.PostForm.Get("address"),
	}
}

func supplierFormFill(d *templates.Data, in service.SupplierInput) {
	d.Form["name"] = in.Name
	d.Form["contact_person"] = in.ContactPerson
	d.Form["email"] = in.Email
	d.Form["phone"] = in.Phone
	d.Form["address"] = in.Address
}

func (h *SuppliersHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	d := h.NewData(r)
	d.Title = "New supplier"
	h.Render(w, r, http.StatusOK, "suppliers/form.html", d)
}

func (h *SuppliersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseSupplierInput(r)
	if _, err := h.Suppliers.Create(r.Context(), in); err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			d := h.NewData(r)
			d.Title = "New supplier"
			d.Errors = verr.Fields
			supplierFormFill(d, in)
			h.Render(w, r, http.StatusUnprocessableEntity, "suppliers/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Supplier created.")
	http.Redirect(w, r, "/suppliers", http.StatusSeeOther)
}

func (h *SuppliersHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s, err := h.Suppliers.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Edit supplier"
	d.Data["Supplier"] = s
	supplierFormFill(d, service.SupplierInput{
		Name: s.Name, ContactPerson: s.ContactPerson, Email: s.Email,
		Phone: s.Phone, Address: s.Address,
	})
	h.Render(w, r, http.StatusOK, "suppliers/form.html", d)
}

func (h *SuppliersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseSupplierInput(r)
	if _, err := h.Suppliers.Update(r.Context(), id, in); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			s, gerr := h.Suppliers.GetByID(r.Context(), id)
			if gerr != nil {
				h.ServerError(w, r, gerr)
				return
			}
			d := h.NewData(r)
			d.Title = "Edit supplier"
			d.Data["Supplier"] = s
			d.Errors = verr.Fields
			supplierFormFill(d, in)
			h.Render(w, r, http.StatusUnprocessableEntity, "suppliers/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Supplier updated.")
	http.Redirect(w, r, "/suppliers", http.StatusSeeOther)
}

func (h *SuppliersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.Suppliers.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Supplier deleted.")
	http.Redirect(w, r, "/suppliers", http.StatusSeeOther)
}
