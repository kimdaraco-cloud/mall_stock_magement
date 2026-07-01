// @ai-modified 2026-07-02 add home and healthz handlers
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"mallstock/internal/templates"
)

// Pinger is the subset of the DB pool the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Home renders the landing page.
type Home struct {
	Tmpl *templates.Cache
	Log  *slog.Logger
}

func (h *Home) Index(w http.ResponseWriter, r *http.Request) {
	d := templates.NewData()
	d.Title = "Home"
	d.CurrentPath = r.URL.Path
	if err := h.Tmpl.Render(w, http.StatusOK, "home.html", d); err != nil {
		h.Log.Error("render home", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// Healthz reports app + database health.
func Healthz(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"degraded","db":"down"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
