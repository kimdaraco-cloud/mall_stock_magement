// @ai-modified 2026-07-02 add dashboard handler (cards, low stock, recent moves)
package handlers

import (
	"net/http"

	"mallstock/internal/repository"
	"mallstock/internal/service"
)

// DashboardHandler serves the landing dashboard.
type DashboardHandler struct {
	*Base
	Reports *service.ReportService
	Stock   *service.StockService
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Reports.DashboardStats(r.Context())
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	lowStock, err := h.Reports.LowStock(r.Context(), 0)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}
	if len(lowStock) > 10 {
		lowStock = lowStock[:10]
	}
	recent, _, err := h.Stock.History(r.Context(), repository.MovementFilter{Page: 1, PerPage: 10})
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	d := h.NewData(r)
	d.Title = "Dashboard"
	d.Data["Stats"] = stats
	d.Data["LowStock"] = lowStock
	d.Data["Recent"] = recent
	h.Render(w, r, http.StatusOK, "dashboard.html", d)
}
