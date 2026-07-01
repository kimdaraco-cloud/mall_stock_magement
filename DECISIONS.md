<!-- @ai-modified 2026-07-02 create decisions log for build choices -->
# Decisions Log

Ambiguous calls made during the build, with rationale. Newest at the bottom.

## Phase 0

- **Go module name: `mallstock`** — no VCS remote exists for this repo, so a plain
  module name is simpler than inventing a fake `github.com/...` path.
- **PostgreSQL via Docker** — no local Postgres install; started container
  `mall-stock-pg` (postgres:16-alpine) on `localhost:5432`, db `mall_stock`,
  user/password `postgres/postgres`. Matches `.env.example` defaults.
- **Migrations run via `go run ./cmd/migrate`** — golang-migrate is used as a
  *library* behind a tiny CLI in `cmd/migrate`, so nobody needs to install the
  `migrate` binary. `make migrate-up/down/create` wrap it.
- **Tailwind via CDN for now** — the standalone Tailwind CLI isn't installed;
  plan.md §2 explicitly allows the CDN build for prototyping. A `make tailwind`
  target can be added later when the CLI is adopted.
- **HTMX vendored** — `web/static/js/htmx.min.js` downloaded (v1.9.12) so the app
  works offline and is self-contained.
- **Templates/static embedded with `embed.FS`** — per plan.md §13, so the binary
  is self-contained; a dev flag could re-enable disk loading later if wanted.
