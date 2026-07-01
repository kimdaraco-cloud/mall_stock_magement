// @ai-modified 2026-07-02 add report pages and CSV exports
package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"mallstock/internal/service"
)

// ReportsHandler serves the report pages and their CSV exports.
type ReportsHandler struct {
	*Base
	Reports *service.ReportService
	Stores  *service.StoreService
}

func (h *ReportsHandler) Index(w http.ResponseWriter, r *http.Request) {
	d := h.NewData(r)
	d.Title = "Reports"
	h.Render(w, r, http.StatusOK, "reports/index.html", d)
}

func (h *ReportsHandler) LowStock(w http.ResponseWriter, r *http.Request) {
	storeID, _ := strconv.ParseInt(r.URL.Query().Get("store"), 10, 64)
	items, err := h.Reports.LowStock(r.Context(), storeID)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	stores, err := h.Stores.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Low stock report"
	d.Data["Items"] = items
	d.Data["Stores"] = stores
	d.Form["store"] = r.URL.Query().Get("store")
	h.Render(w, r, http.StatusOK, "reports/low_stock.html", d)
}

func (h *ReportsHandler) Valuation(w http.ResponseWriter, r *http.Request) {
	byStore, err := h.Reports.ValuationByStore(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	byCategory, err := h.Reports.ValuationByCategory(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Stock valuation"
	d.Data["ByStore"] = byStore
	d.Data["ByCategory"] = byCategory
	h.Render(w, r, http.StatusOK, "reports/valuation.html", d)
}

func (h *ReportsHandler) Movements(w http.ResponseWriter, r *http.Request) {
	f := parseMovementFilter(r)
	f.PerPage = 200
	movements, total, err := h.Reports.MovementReport(r.Context(), f)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	stores, err := h.Stores.List(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	d := h.NewData(r)
	d.Title = "Movement report"
	d.Data["Movements"] = movements
	d.Data["Total"] = total
	d.Data["Stores"] = stores
	d.Form["type"] = r.URL.Query().Get("type")
	d.Form["store"] = r.URL.Query().Get("store")
	d.Form["from"] = r.URL.Query().Get("from")
	d.Form["to"] = r.URL.Query().Get("to")
	h.Render(w, r, http.StatusOK, "reports/movements.html", d)
}

// ExportCSV streams the named report as a CSV download, honouring the same
// query filters as the HTML page.
func (h *ReportsHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", name))
	cw := csv.NewWriter(w)
	defer cw.Flush()

	switch name {
	case "low-stock":
		storeID, _ := strconv.ParseInt(r.URL.Query().Get("store"), 10, 64)
		items, err := h.Reports.LowStock(r.Context(), storeID)
		if err != nil {
			h.ServerError(w, r, err)
			return
		}
		_ = cw.Write([]string{"SKU", "Product", "Store", "Category", "Quantity", "Reorder level", "Unit", "Suggested reorder qty"})
		for _, it := range items {
			_ = cw.Write([]string{it.SKU, it.Name, it.StoreName, it.CategoryName,
				strconv.Itoa(it.Quantity), strconv.Itoa(it.ReorderLevel), it.Unit, strconv.Itoa(it.SuggestedQty)})
		}

	case "valuation":
		byStore, err := h.Reports.ValuationByStore(r.Context())
		if err != nil {
			h.ServerError(w, r, err)
			return
		}
		byCategory, err := h.Reports.ValuationByCategory(r.Context())
		if err != nil {
			h.ServerError(w, r, err)
			return
		}
		_ = cw.Write([]string{"Group by", "Group", "Products", "Units", "Stock value"})
		for _, v := range byStore {
			_ = cw.Write([]string{"store", v.Group, strconv.Itoa(v.Products), strconv.Itoa(v.Units), v.Value})
		}
		for _, v := range byCategory {
			_ = cw.Write([]string{"category", v.Group, strconv.Itoa(v.Products), strconv.Itoa(v.Units), v.Value})
		}

	case "movements":
		f := parseMovementFilter(r)
		f.Page = 1
		f.PerPage = 10000 // export cap; noted in DECISIONS.md
		movements, _, err := h.Reports.MovementReport(r.Context(), f)
		if err != nil {
			h.ServerError(w, r, err)
			return
		}
		_ = cw.Write([]string{"When", "Product", "SKU", "Store", "Type", "Quantity", "Balance after", "Reference", "Notes", "By"})
		for _, m := range movements {
			_ = cw.Write([]string{m.CreatedAt.Format("2006-01-02 15:04:05"), m.ProductName, m.ProductSKU,
				m.StoreName, m.MovementType, strconv.Itoa(m.Quantity), strconv.Itoa(m.QuantityAfter),
				m.Reference, m.Notes, m.UserName})
		}

	default:
		http.NotFound(w, r)
	}
}
