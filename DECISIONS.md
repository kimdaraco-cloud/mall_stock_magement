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

## Phase 1

- **Sessions stored in Postgres (`scs/pgxstore`)** — plan.md §11 calls for a
  server-side session store; a `sessions` table migration (000003) backs it, so
  sessions survive restarts. Lifetime 12h.
- **nosurf `SetIsTLSFunc`** — nosurf v1.2.0 assumes TLS by default and then
  requires browser origin headers on every POST. We detect the real scheme via
  `r.TLS` / `X-Forwarded-Proto`. Note: nosurf still requires *some* origin
  evidence (Sec-Fetch-Site / Origin / Referer) on POSTs — all browsers send
  these; bare curl needs `-H "Origin: ..."`.
- **Deactivate, never delete users** — user rows are referenced by
  `stock_movements.user_id` later; the UI toggles `is_active` only. Admins
  cannot deactivate their own account.
- **Login treats a deactivated account as invalid credentials** — avoids
  leaking which emails exist.

## Phase 2

- **Prices are decimal strings end-to-end** — Go carries prices as validated
  strings (`^\d{1,10}(\.\d{1,2})?$`), cast with `::numeric` on write and
  `::text` on read. No float ever touches money; no decimal dependency added.
- **Stores/products soft-delete only; categories/suppliers hard-delete** —
  stores and products are referenced by history, so they toggle `is_active`.
  Categories/suppliers use plain DELETE; products keep working because the FK
  is `ON DELETE SET NULL`.
- **Catalog permissions** — list/detail pages are visible to every logged-in
  role; create/edit/deactivate routes are gated to admin+manager, matching the
  plan.md §8 matrix.
- **New products always start at quantity 0** — receiving stock is Phase 3's
  transactional stock-in; letting product creation set a quantity would bypass
  the stock invariant.

## Phase 3

- **Row lock (`SELECT ... FOR UPDATE`) serialises stock math** — every stock
  operation locks the product row, computes the new balance, inserts the
  movement, and updates the cached quantity in one transaction. A concurrency
  test (20 goroutines vs 10 units) proves overselling is impossible.
- **Adjustment stores the absolute delta** — `quantity` is always positive per
  the schema comment; for adjustments it is `|new − old|`, direction is
  recoverable from `quantity_after` vs the previous movement.
- **Adjust form lives inline on the product detail page** — plan.md §7 defines
  only `POST /products/{id}/adjust` (no GET form page), so a small inline form
  with a confirm dialog fits better than a dedicated page.
- **All roles may record stock in/out/adjust** — per the §8 capability matrix
  (staff included); only catalog *editing* is restricted.
