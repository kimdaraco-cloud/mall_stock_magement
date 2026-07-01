// @ai-modified 2026-07-02 embed templates and static assets into the binary
package web

import "embed"

// Templates holds all HTML templates (layouts, pages, partials).
//
//go:embed templates
var Templates embed.FS

// Static holds all static assets served under /static/.
//
//go:embed static
var Static embed.FS
