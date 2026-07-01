// @ai-modified 2026-07-02 add template cache render tests
package templates

import (
	"net/http/httptest"
	"strings"
	"testing"

	"mallstock/web"
)

func TestNewParsesAllPages(t *testing.T) {
	if _, err := New(web.Templates); err != nil {
		t.Fatalf("template cache failed to build: %v", err)
	}
}

func TestRenderLoginPage(t *testing.T) {
	c, err := New(web.Templates)
	if err != nil {
		t.Fatalf("build cache: %v", err)
	}
	d := NewData()
	d.Title = "Log in"
	d.CSRFToken = "test-token"

	rec := httptest.NewRecorder()
	if err := c.Render(rec, 200, "login.html", d); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := rec.Body.String()
	for _, want := range []string{"test-token", "Log in", `action="/login"`} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered login page missing %q", want)
		}
	}
}

func TestRenderUnknownPageErrors(t *testing.T) {
	c, err := New(web.Templates)
	if err != nil {
		t.Fatalf("build cache: %v", err)
	}
	if err := c.Render(httptest.NewRecorder(), 200, "nope.html", NewData()); err == nil {
		t.Error("want error for unknown template, got nil")
	}
}
