// @ai-modified 2026-07-02 add shared handler helpers (render data, flash, errors)
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"

	"mallstock/internal/middleware"
	"mallstock/internal/templates"
)

// Base carries dependencies shared by all resource handlers.
type Base struct {
	Tmpl    *templates.Cache
	Log     *slog.Logger
	Session *scs.SessionManager
}

// NewData builds the common template payload for a request: CSRF token,
// current user, current path, and any pending flash message.
func (b *Base) NewData(r *http.Request) *templates.Data {
	d := templates.NewData()
	d.CSRFToken = nosurf.Token(r)
	d.CurrentPath = r.URL.Path
	if u := middleware.CurrentUser(r.Context()); u != nil {
		d.User = u
		d.IsAuthenticated = true
	}
	if msg := b.Session.PopString(r.Context(), "flash"); msg != "" {
		kind := b.Session.PopString(r.Context(), "flashKind")
		if kind == "" {
			kind = "info"
		}
		d.Flash = &templates.Flash{Kind: kind, Message: msg}
	}
	return d
}

// Flash queues a one-shot message for the next rendered page.
func (b *Base) Flash(r *http.Request, kind, message string) {
	b.Session.Put(r.Context(), "flash", message)
	b.Session.Put(r.Context(), "flashKind", kind)
}

// Render writes a full page and logs+500s on template failure.
func (b *Base) Render(w http.ResponseWriter, r *http.Request, status int, page string, d *templates.Data) {
	if err := b.Tmpl.Render(w, status, page, d); err != nil {
		b.ServerError(w, r, err)
	}
}

// ServerError logs the error and sends a generic 500.
func (b *Base) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	b.Log.Error("server error", "method", r.Method, "path", r.URL.Path, "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
