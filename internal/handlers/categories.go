// @ai-modified 2026-07-02 add category CRUD handlers
package handlers

import (
	"errors"
	"net/http"

	"mallstock/internal/service"
)

// CategoriesHandler serves category CRUD pages.
type CategoriesHandler struct {
	*Base
	Categories *service.CategoryService
}

func (h *CategoriesHandler) List(w http.ResponseWriter, r *http.Request) {
	cats, err := h.Categories.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Categories"
	d.Data["Categories"] = cats
	h.Render(w, r, http.StatusOK, "categories/list.html", d)
}

func (h *CategoriesHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	d := h.NewData(r)
	d.Title = "New category"
	h.Render(w, r, http.StatusOK, "categories/form.html", d)
}

func (h *CategoriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := service.CategoryInput{
		Name:        r.PostForm.Get("name"),
		Description: r.PostForm.Get("description"),
	}
	if _, err := h.Categories.Create(r.Context(), in); err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			d := h.NewData(r)
			d.Title = "New category"
			d.Errors = verr.Fields
			d.Form["name"] = in.Name
			d.Form["description"] = in.Description
			h.Render(w, r, http.StatusUnprocessableEntity, "categories/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Category created.")
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *CategoriesHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	c, err := h.Categories.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Edit category"
	d.Data["Category"] = c
	d.Form["name"] = c.Name
	d.Form["description"] = c.Description
	h.Render(w, r, http.StatusOK, "categories/form.html", d)
}

func (h *CategoriesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := service.CategoryInput{
		Name:        r.PostForm.Get("name"),
		Description: r.PostForm.Get("description"),
	}
	if _, err := h.Categories.Update(r.Context(), id, in); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			c, gerr := h.Categories.GetByID(r.Context(), id)
			if gerr != nil {
				h.ServerError(w, r, gerr)
				return
			}
			d := h.NewData(r)
			d.Title = "Edit category"
			d.Data["Category"] = c
			d.Errors = verr.Fields
			d.Form["name"] = in.Name
			d.Form["description"] = in.Description
			h.Render(w, r, http.StatusUnprocessableEntity, "categories/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Category updated.")
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *CategoriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.Categories.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Category deleted. Products in it are now uncategorised.")
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}
