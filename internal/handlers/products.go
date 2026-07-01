// @ai-modified 2026-07-02 add product CRUD handlers with search/filter/pagination
package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"mallstock/internal/repository"
	"mallstock/internal/service"
	"mallstock/internal/templates"
)

const productsPerPage = 20

// ProductsHandler serves product CRUD pages.
type ProductsHandler struct {
	*Base
	Products   *service.ProductService
	Stores     *service.StoreService
	Categories *service.CategoryService
	Suppliers  *service.SupplierService
	Stock      *service.StockService
}

// Pagination is the view model for the pager partial.
type Pagination struct {
	Page       int
	TotalPages int
	Total      int
	PrevURL    string
	NextURL    string
}

func parseProductFilter(r *http.Request) repository.ProductFilter {
	q := r.URL.Query()
	f := repository.ProductFilter{
		Query:   q.Get("q"),
		PerPage: productsPerPage,
	}
	if v, err := strconv.ParseInt(q.Get("store"), 10, 64); err == nil && v > 0 {
		f.StoreID = v
	}
	if v, err := strconv.ParseInt(q.Get("category"), 10, 64); err == nil && v > 0 {
		f.CategoryID = v
	}
	f.LowStock = q.Get("low") == "1"
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		f.Page = v
	} else {
		f.Page = 1
	}
	return f
}

func pageURL(r *http.Request, page int) string {
	q := r.URL.Query()
	q.Set("page", strconv.Itoa(page))
	u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
	return u.String()
}

func (h *ProductsHandler) List(w http.ResponseWriter, r *http.Request) {
	f := parseProductFilter(r)
	products, total, err := h.Products.List(r.Context(), f)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	stores, err := h.Stores.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	cats, err := h.Categories.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	totalPages := (total + f.PerPage - 1) / f.PerPage
	if totalPages < 1 {
		totalPages = 1
	}
	pg := Pagination{Page: f.Page, TotalPages: totalPages, Total: total}
	if f.Page > 1 {
		pg.PrevURL = pageURL(r, f.Page-1)
	}
	if f.Page < totalPages {
		pg.NextURL = pageURL(r, f.Page+1)
	}

	d := h.NewData(r)
	d.Title = "Products"
	d.Data["Products"] = products
	d.Data["Stores"] = stores
	d.Data["Categories"] = cats
	d.Data["Pagination"] = pg
	d.Form["q"] = f.Query
	if f.StoreID > 0 {
		d.Form["store"] = strconv.FormatInt(f.StoreID, 10)
	}
	if f.CategoryID > 0 {
		d.Form["category"] = strconv.FormatInt(f.CategoryID, 10)
	}
	if f.LowStock {
		d.Form["low"] = "1"
	}
	h.Render(w, r, http.StatusOK, "products/list.html", d)
}

func (h *ProductsHandler) formData(r *http.Request) (*templates.Data, error) {
	stores, err := h.Stores.ListActive(r.Context())
	if err != nil {
		return nil, err
	}
	cats, err := h.Categories.List(r.Context())
	if err != nil {
		return nil, err
	}
	sups, err := h.Suppliers.List(r.Context())
	if err != nil {
		return nil, err
	}
	d := h.NewData(r)
	d.Data["Stores"] = stores
	d.Data["Categories"] = cats
	d.Data["Suppliers"] = sups
	return d, nil
}

func parseProductInput(r *http.Request) service.ProductInput {
	in := service.ProductInput{
		SKU:          r.PostForm.Get("sku"),
		Barcode:      r.PostForm.Get("barcode"),
		Name:         r.PostForm.Get("name"),
		Description:  r.PostForm.Get("description"),
		CostPrice:    r.PostForm.Get("cost_price"),
		SellingPrice: r.PostForm.Get("selling_price"),
		Unit:         r.PostForm.Get("unit"),
		IsActive:     r.PostForm.Get("is_active") == "on",
	}
	if v, err := strconv.ParseInt(r.PostForm.Get("store_id"), 10, 64); err == nil && v > 0 {
		in.StoreID = v
	}
	if v, err := strconv.ParseInt(r.PostForm.Get("category_id"), 10, 64); err == nil && v > 0 {
		in.CategoryID = &v
	}
	if v, err := strconv.ParseInt(r.PostForm.Get("supplier_id"), 10, 64); err == nil && v > 0 {
		in.SupplierID = &v
	}
	if v, err := strconv.Atoi(r.PostForm.Get("reorder_level")); err == nil {
		in.ReorderLevel = v
	}
	return in
}

func productFormFill(d *templates.Data, in service.ProductInput) {
	d.Form["sku"] = in.SKU
	d.Form["barcode"] = in.Barcode
	d.Form["name"] = in.Name
	d.Form["description"] = in.Description
	d.Form["cost_price"] = in.CostPrice
	d.Form["selling_price"] = in.SellingPrice
	d.Form["unit"] = in.Unit
	d.Form["reorder_level"] = strconv.Itoa(in.ReorderLevel)
	d.Form["store_id"] = strconv.FormatInt(in.StoreID, 10)
	if in.CategoryID != nil {
		d.Form["category_id"] = strconv.FormatInt(*in.CategoryID, 10)
	}
	if in.SupplierID != nil {
		d.Form["supplier_id"] = strconv.FormatInt(*in.SupplierID, 10)
	}
	if in.IsActive {
		d.Form["is_active"] = "on"
	}
}

func (h *ProductsHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	d, err := h.formData(r)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d.Title = "New product"
	d.Form["is_active"] = "on"
	d.Form["unit"] = "pcs"
	d.Form["reorder_level"] = "0"
	h.Render(w, r, http.StatusOK, "products/form.html", d)
}

func (h *ProductsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseProductInput(r)
	p, err := h.Products.Create(r.Context(), in)
	if err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			d, derr := h.formData(r)
			if derr != nil {
				h.ServerError(w, r, derr)
				return
			}
			d.Title = "New product"
			d.Errors = verr.Fields
			productFormFill(d, in)
			h.Render(w, r, http.StatusUnprocessableEntity, "products/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Product created.")
	http.Redirect(w, r, fmt.Sprintf("/products/%d", p.ID), http.StatusSeeOther)
}

func (h *ProductsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.Products.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	movements, err := h.Stock.RecentForProduct(r.Context(), id, 10)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = p.Name
	d.Data["Product"] = p
	d.Data["Movements"] = movements
	h.Render(w, r, http.StatusOK, "products/detail.html", d)
}

func (h *ProductsHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.Products.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	d, err := h.formData(r)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d.Title = "Edit product"
	d.Data["Product"] = p
	productFormFill(d, service.ProductInput{
		SKU: p.SKU, Barcode: p.Barcode, Name: p.Name, Description: p.Description,
		CategoryID: p.CategoryID, SupplierID: p.SupplierID, StoreID: p.StoreID,
		CostPrice: p.CostPrice, SellingPrice: p.SellingPrice,
		ReorderLevel: p.ReorderLevel, Unit: p.Unit, IsActive: p.IsActive,
	})
	h.Render(w, r, http.StatusOK, "products/form.html", d)
}

func (h *ProductsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := parseProductInput(r)
	if _, err := h.Products.Update(r.Context(), id, in); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			p, gerr := h.Products.GetByID(r.Context(), id)
			if gerr != nil {
				h.ServerError(w, r, gerr)
				return
			}
			d, derr := h.formData(r)
			if derr != nil {
				h.ServerError(w, r, derr)
				return
			}
			d.Title = "Edit product"
			d.Data["Product"] = p
			d.Errors = verr.Fields
			productFormFill(d, in)
			h.Render(w, r, http.StatusUnprocessableEntity, "products/form.html", d)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Product updated.")
	http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
}

func (h *ProductsHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.Products.Deactivate(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.ServerError(w, r, err)
		return
	}
	h.Flash(r, "success", "Product deactivated.")
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}
