# CLAUDE.md

Working instructions for Claude Code on this repository. Read `plan.md` for full scope, data model, and the phased roadmap. This file is the "how to work here" guide — keep it accurate as the project evolves.

## Project

Mall Stock Management System: a web app to manage inventory across shops/units in a shopping mall. **Go** backend with **server-rendered HTML** (`html/template` + **HTMX**), **PostgreSQL** database. Single-binary deploy.

## Tech Stack

- Go 1.22+ · chi router · pgx v5 (`pgxpool`) · golang-migrate
- `html/template` + HTMX + Tailwind CSS (standalone CLI)
- scs (sessions) · bcrypt (passwords) · nosurf (CSRF) · `log/slog` (logging)
- Hand-written SQL in a repository layer — **no ORM**.

## Commands

Use the Makefile. If a target doesn't exist yet, create it.

```bash
make dev                       # run with live reload (air) → :8080
make run                       # go run ./cmd/server
make build                     # CGO_ENABLED=0 build to ./bin/server
make test                      # go test ./...
make lint                      # golangci-lint run
make fmt                       # gofmt + goimports
make migrate-up                # apply all migrations
make migrate-down              # roll back the last migration
make migrate-create name=xxx   # scaffold a new up/down migration pair
make seed                      # load sample data (creates admin user)
```

Run `make fmt` and `make test` before finishing any change. Do not commit if tests fail.

## Architecture — layered, dependencies point DOWN only

```
handlers → services → repositories → database
              ↑ models (plain structs, shared, no logic)
```

Non-negotiable rules:
- **Handlers** parse the request, call a service, render a template/redirect. No SQL, no business logic.
- **Services** hold business rules, validation, and own **transactions**. All multi-write operations run in ONE transaction here.
- **Repositories** contain SQL only. No business logic. Every method takes `context.Context`.
- **Models** are dumb structs. No methods with side effects.
- A lower layer must never import an upper layer.

## The stock invariant (most important rule)

`products.quantity` is a cached total. It may **only** change inside the same DB transaction that inserts a `stock_movements` row. Never `UPDATE products SET quantity` anywhere else. Each movement stores `quantity_after` for auditability. Stock-out must reject going below zero. When in doubt about stock logic, re-read §5 and §6 of `plan.md`.

## Conventions

**Go**
- Wrap errors with context: `fmt.Errorf("create product: %w", err)`. Return errors up; handle (log + HTTP response) at the handler edge.
- Use `context.Context` as the first arg on service/repo methods.
- Keep functions small and single-purpose. Prefer clarity over cleverness.
- Structured logs via `slog`; never log secrets, passwords, session tokens, or full DB URLs.
- Table-driven tests, standard `testing` package.

**Database / SQL**
- **Parameterized queries only** (`$1, $2, ...`). Never concatenate user input into SQL.
- Schema changes = a new migration pair in `/migrations` (never edit an applied migration). Always write the `down`.
- Use `NUMERIC` for money (already in schema) — never floats for prices.

**HTTP / templates**
- Mutations are `POST` (HTML forms have no PUT/DELETE).
- Every state-changing form includes the CSRF token.
- Return HTMX partials from `web/templates/partials/` for in-place updates; full pages otherwise.
- Rely on `html/template` auto-escaping. Never wrap user input in `template.HTML`.

**Security**
- Check role/permission in middleware on every protected route — hiding a UI button is not access control.
- Secrets come from env only. `.env` is git-ignored; update `.env.example` (with blank values) when adding a new variable.

## Adding a feature (workflow)

1. Confirm which **phase** in `plan.md` it belongs to; work phases in order.
2. If it needs schema: `make migrate-create name=...`, write up + down SQL.
3. Add/extend the **model** struct.
4. Add **repository** methods (SQL only) + a repo test.
5. Add **service** logic (validation, transaction) + a service test.
6. Add **handler** + route + **template**.
7. `make fmt && make test && make lint`. Then a small, focused commit.

## Git

- Small, focused commits. Imperative subject line, e.g. `feat: add stock-in handler`, `fix: guard negative stock`.
- Never commit `.env`, `bin/`, secrets, or generated Tailwind output that's meant to be built.

## Do / Don't

- ✅ Follow the layering; keep SQL in repos and transactions in services.
- ✅ Preserve the stock invariant; write a test for any stock-changing code.
- ✅ Ask/confirm before deleting data or changing an applied migration.
- ❌ Don't add an ORM or a heavy JS framework — stay with SQL + HTMX.
- ❌ Don't skip the `down` migration or `.env.example` update.
- ❌ Don't introduce a dependency without a clear reason; prefer the stdlib.

## Current status

Phase: **3 complete — Phase 4 (dashboard & reports) in progress**. Update this line as phases complete so context is never lost between sessions.

<!-- @ai-modified 2026-07-02 update current-status line after Phase 0 -->
