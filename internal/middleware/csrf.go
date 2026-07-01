// @ai-modified 2026-07-02 add nosurf CSRF middleware
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/justinas/nosurf"
)

// CSRF protects all state-changing requests with a nosurf token cookie.
func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := nosurf.New(next)
		h.SetBaseCookie(http.Cookie{
			HttpOnly: true,
			Path:     "/",
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
		// nosurf assumes TLS by default and then demands browser origin
		// headers; detect the actual scheme (direct TLS or trusted proxy).
		h.SetIsTLSFunc(func(r *http.Request) bool {
			return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		})
		h.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Warn("csrf rejected", "path", r.URL.Path, "reason", nosurf.Reason(r))
			http.Error(w, "Forbidden - CSRF token invalid", http.StatusForbidden)
		}))
		return h
	}
}
