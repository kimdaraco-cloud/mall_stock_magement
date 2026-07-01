// @ai-modified 2026-07-02 add stock in/out/adjust handlers and movement history
package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mallstock/internal/middleware"
	"mallstock/internal/models"
	"mallstock/internal/repository"
	"mallstock/internal/service"
)

const movementsPerPage = 50

// StockHandler serves stock-in / stock-out / adjustment and movement history.
type StockHandler struct {
	*Base
	Stock    *service.StockService
	Products *service.ProductService
	Stores   *service.StoreService
}

// StockForm renders the stock-in or stock-out form. mode is "in" or "out".
func (h *StockHandler) StockForm(mode string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		d := h.NewData(r)
		if mode == models.MovementIn {
			d.Title = "Stock in — " + p.Name
		} else {
			d.Title = "Stock out — " + p.Name
		}
		d.Data["Product"] = p
		d.Data["Mode"] = mode
		h.Render(w, r, http.StatusOK, "products/stock_form.html", d)
	}
}

func (h *StockHandler) parseInput(r *http.Request, productID int64) service.MovementInput {
	qty, _ := strconv.Atoi(r.PostForm.Get("quantity"))
	in := service.MovementInput{
		ProductID: productID,
		Quantity:  qty,
		Reference: r.PostForm.Get("reference"),
		Notes:     r.PostForm.Get("notes"),
	}
	if u := middleware.CurrentUser(r.Context()); u != nil {
		in.UserID = u.ID
	}
	return in
}

// rerenderStockForm shows the form again with an error message.
func (h *StockHandler) rerenderStockForm(w http.ResponseWriter, r *http.Request, productID int64, mode, msg string) {
	p, err := h.Products.GetByID(r.Context(), productID)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	if mode == models.MovementIn {
		d.Title = "Stock in — " + p.Name
	} else {
		d.Title = "Stock out — " + p.Name
	}
	d.Data["Product"] = p
	d.Data["Mode"] = mode
	d.Errors["quantity"] = msg
	d.Form["quantity"] = r.PostForm.Get("quantity")
	d.Form["reference"] = r.PostForm.Get("reference")
	d.Form["notes"] = r.PostForm.Get("notes")
	h.Render(w, r, http.StatusUnprocessableEntity, "products/stock_form.html", d)
}

func (h *StockHandler) StockIn(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := h.parseInput(r, id)
	m, err := h.Stock.ReceiveStock(r.Context(), in)
	if err != nil {
		h.handleStockErr(w, r, id, models.MovementIn, err)
		return
	}
	h.Flash(r, "success", fmt.Sprintf("Received %d — stock is now %d.", m.Quantity, m.QuantityAfter))
	http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
}

func (h *StockHandler) StockOut(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := h.parseInput(r, id)
	m, err := h.Stock.IssueStock(r.Context(), in)
	if err != nil {
		h.handleStockErr(w, r, id, models.MovementOut, err)
		return
	}
	h.Flash(r, "success", fmt.Sprintf("Issued %d — stock is now %d.", m.Quantity, m.QuantityAfter))
	http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
}

// Adjust handles the inline correction form on the product detail page.
func (h *StockHandler) Adjust(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := h.parseInput(r, id)
	if in.Reference == "" {
		in.Reference = "count correction"
	}
	m, err := h.Stock.AdjustStock(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			http.NotFound(w, r)
		case errors.Is(err, service.ErrInvalidQuantity):
			h.Flash(r, "error", "Adjustment count must be zero or more.")
			http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
		default:
			h.ServerError(w, r, err)
		}
		return
	}
	h.Flash(r, "success", fmt.Sprintf("Count corrected to %d.", m.QuantityAfter))
	http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
}

func (h *StockHandler) handleStockErr(w http.ResponseWriter, r *http.Request, id int64, mode string, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		http.NotFound(w, r)
	case errors.Is(err, service.ErrInvalidQuantity):
		h.rerenderStockForm(w, r, id, mode, "Enter a quantity greater than zero.")
	case errors.Is(err, service.ErrInsufficientStock):
		h.rerenderStockForm(w, r, id, mode, "Not enough stock: "+err.Error())
	default:
		h.ServerError(w, r, err)
	}
}

func parseMovementFilter(r *http.Request) repository.MovementFilter {
	q := r.URL.Query()
	f := repository.MovementFilter{PerPage: movementsPerPage}
	if v, err := strconv.ParseInt(q.Get("product"), 10, 64); err == nil && v > 0 {
		f.ProductID = v
	}
	if v, err := strconv.ParseInt(q.Get("store"), 10, 64); err == nil && v > 0 {
		f.StoreID = v
	}
	if t := q.Get("type"); models.ValidMovementType(t) {
		f.Type = t
	}
	if v, err := time.Parse("2006-01-02", q.Get("from")); err == nil {
		f.From = v
	}
	if v, err := time.Parse("2006-01-02", q.Get("to")); err == nil {
		f.To = v.AddDate(0, 0, 1) // inclusive end date
	}
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		f.Page = v
	} else {
		f.Page = 1
	}
	return f
}

// History renders the filterable movement list.
func (h *StockHandler) History(w http.ResponseWriter, r *http.Request) {
	f := parseMovementFilter(r)
	movements, total, err := h.Stock.History(r.Context(), f)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	stores, err := h.Stores.List(r.Context())
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
	d.Title = "Stock movements"
	d.Data["Movements"] = movements
	d.Data["Stores"] = stores
	d.Data["Pagination"] = pg
	d.Form["type"] = r.URL.Query().Get("type")
	d.Form["store"] = r.URL.Query().Get("store")
	d.Form["from"] = r.URL.Query().Get("from")
	d.Form["to"] = r.URL.Query().Get("to")
	d.Form["product"] = r.URL.Query().Get("product")
	h.Render(w, r, http.StatusOK, "movements/list.html", d)
}
