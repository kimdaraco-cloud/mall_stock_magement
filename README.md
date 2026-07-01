<!-- @ai-modified 2026-07-02 add project README -->
# Mall Stock Management System

A web app for mall staff to track products and stock across the shops/units of
a shopping mall: live stock levels, a full audit trail of every movement,
low-stock alerts, and valuation/movement reports.

Built with **Go** (chi, pgx, `html/template`) + **HTMX** + **Tailwind**, backed
by **PostgreSQL**. Ships as a single binary with templates and static assets
embedded. See `plan.md` for the full design and `DECISIONS.md` for choices made
along the way.

## Quick start

```bash
cp .env.example .env               # fill in values (see below)
make db-up                         # start the dev Postgres container (Docker)
make migrate-up                    # apply schema
make seed                          # sample data + default logins
make dev                           # run → http://localhost:8080
```

Default logins from the seed (change immediately):

| Role    | Email              | Password   |
|---------|--------------------|------------|
| admin   | admin@mall.local   | admin123   |
| manager | manager@mall.local | manager123 |
| staff   | staff@mall.local   | staff1234  |

`.env` for local dev:

```env
APP_ENV=development
PORT=8080
SESSION_SECRET=<long random string>
DATABASE_URL=postgres://postgres:postgres@localhost:5432/mall_stock?sslmode=disable
LOG_LEVEL=debug
```

## Common commands

```bash
make test          # go test ./... (service tests need the dev database)
make lint          # golangci-lint (falls back to go vet)
make fmt           # gofmt + goimports
make build         # static binary → bin/server
make migrate-create name=add_thing
```

## Architecture

Layered — dependencies point down only:

```
handlers → services → repositories → PostgreSQL
              ↑ models (plain structs)
```

**The stock invariant:** `products.quantity` is a cached total. It only ever
changes inside the same DB transaction that inserts a `stock_movements` row
(row-locked with `SELECT ... FOR UPDATE`). Stock-out rejects going negative;
every movement stores `quantity_after` so history is auditable and the total
rebuildable.

## Features

- Auth with server-side sessions, bcrypt, CSRF protection, role-based access
  (admin / manager / staff)
- CRUD for stores, categories, suppliers, products (search / filter /
  pagination, HTMX live search)
- Stock in / out / adjustment with an immutable movement history
- Dashboard: stock value, low-stock alerts, recent movements
- Low-stock, valuation, and movement reports with CSV export
