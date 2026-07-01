// @ai-modified 2026-07-02 add template cache and render helpers
package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"mallstock/internal/models"
)

// Flash is a one-shot message shown after a redirect.
type Flash struct {
	Kind    string // success | error | info
	Message string
}

// Data is the payload passed to every template render.
type Data struct {
	Title           string
	CSRFToken       string
	IsAuthenticated bool
	User            *models.User
	CurrentPath     string
	Flash           *Flash
	Errors          map[string]string // field -> message (form validation)
	Form            map[string]string // sticky form values on validation failure
	Data            map[string]any    // page-specific payload
}

// NewData returns a Data with maps initialised so templates can index safely.
func NewData() *Data {
	return &Data{
		Errors: map[string]string{},
		Form:   map[string]string{},
		Data:   map[string]any{},
	}
}

// Cache parses and stores one template set per page, all sharing the base
// layout and partials.
type Cache struct {
	pages map[string]*template.Template
}

var funcs = template.FuncMap{
	"now": time.Now,
}

// New builds the cache from a fs.FS rooted at the web directory
// (templates/layouts, templates/pages, templates/partials).
func New(fsys fs.FS) (*Cache, error) {
	pages, err := fs.Glob(fsys, "templates/pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}
	nested, err := fs.Glob(fsys, "templates/pages/*/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob nested pages: %w", err)
	}
	pages = append(pages, nested...)

	c := &Cache{pages: make(map[string]*template.Template, len(pages))}
	for _, page := range pages {
		name := page[len("templates/pages/"):]
		t, err := template.New(name).Funcs(funcs).ParseFS(fsys,
			"templates/layouts/*.html", page)
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", page, err)
		}
		// Partials are optional (none exist yet in Phase 0).
		if partials, _ := fs.Glob(fsys, "templates/partials/*.html"); len(partials) > 0 {
			if t, err = t.ParseFS(fsys, "templates/partials/*.html"); err != nil {
				return nil, fmt.Errorf("parse partials for %s: %w", page, err)
			}
		}
		c.pages[name] = t
	}
	return c, nil
}

// Render writes a full page (base layout + named page) to w.
func (c *Cache) Render(w http.ResponseWriter, status int, page string, d *Data) error {
	t, ok := c.pages[page]
	if !ok {
		return fmt.Errorf("render: template %q not found", page)
	}
	if d == nil {
		d = NewData()
	}
	// Render to a buffer first so a template error never sends a half page.
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", d); err != nil {
		return fmt.Errorf("render %s: %w", page, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

// RenderPartial writes a single named partial (for HTMX responses).
func (c *Cache) RenderPartial(w http.ResponseWriter, status int, page, name string, d *Data) error {
	t, ok := c.pages[page]
	if !ok {
		return fmt.Errorf("render partial: template set %q not found", page)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, d); err != nil {
		return fmt.Errorf("render partial %s: %w", name, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}
