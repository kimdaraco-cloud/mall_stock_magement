// @ai-modified 2026-07-02 slim to healthz only; dashboard replaced the home page
package handlers

import (
	"context"
	"net/http"
	"time"
)

// Pinger is the subset of the DB pool the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
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
