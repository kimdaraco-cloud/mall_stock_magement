# Mall Stock Management System — Project Plan

A web application to manage inventory (stock) for a shopping mall, built with **Go** (backend + server-rendered HTML) and **HTML/HTMX** (frontend). This document is the single source of truth for scope, architecture, data model, and the build roadmap. Work through it phase by phase.

---

## 1. Overview

### 1.1 Purpose
A centralized system for mall staff to track products and stock across multiple shops/units in the mall: what's in stock, what's running low, what moved in and out, and what it's all worth.

### 1.2 Goals
- Manage the shops/units in the mall and the products in each.
- Track live stock levels with a full, auditable history of every movement.
- Alert on low stock so nothing runs out unexpectedly.
- Give managers dashboards and reports (stock levels, valuation, movement history).
- Simple role-based access (admin / manager / staff).
- Fast to run locally, easy to deploy as a single binary.

### 1.3 Non-Goals (v1 scope boundary)
- No point-of-sale / cash register / receipt printing.
- No accounting / invoicing / tax engine.
- No customer-facing storefront or e-commerce.
- No mobile native app (the web UI is responsive instead).
- No real-time multi-user websockets (page + HTMX refresh is enough).

Keep these out of v1. They are listed in [§14 Future Enhancements](#14-future-enhancements).

---

## 2. Tech Stack

| Layer | Choice | Why |
|---|---|---|
| Language | **Go 1.22+** | Single binary, fast, great stdlib, easy for an agent to work in. |
| Router | **chi** (`github.com/go-chi/chi/v5`) | Lightweight, idiomatic, clean middleware + URL params. |
| Database | **PostgreSQL 16** | Solid relational store. (SQLite is fine for local dev — see note below.) |
| DB driver | **pgx v5** (`github.com/jackc/pgx/v5` + `pgxpool`) | Fast, well-maintained, no ORM magic. |
| Migrations | **golang-migrate** | Plain `.sql` up/down files, versioned. |
| Data access | Hand-written SQL in a **repository layer** | Explicit, debuggable, no hidden queries. |
| Templating | **`html/template`** (stdlib) | Server-rendered HTML, auto-escaped, zero deps. |
| Interactivity | **HTMX** | Dynamic partial updates without a JS framework. |
| Styling | **Tailwind CSS** (standalone CLI, no Node) | Utility-first; a CDN build is fine for prototyping. |
| Sessions | **scs** (`github.com/alexedwards/scs/v2`) | Simple, secure server-side sessions. |
| Passwords | **bcrypt** (`golang.org/x/crypto/bcrypt`) | Standard password hashing. |
| CSRF | **nosurf** (`github.com/justinas/nosurf`) | CSRF protection for form POSTs. |
| Logging | **`log/slog`** (stdlib) | Structured logging. |
| Live reload (dev) | **air** (`github.com/air-verse/air`) | Auto-rebuild on save. |
| Lint | **golangci-lint** | Catches bugs and style issues. |

> **Dependencies are introduced gradually by phase** — you do not need to install all of these on day one. Phase 0 needs Go, chi, pgx, and golang-migrate; everything else comes in when its phase starts.

> **SQLite alternative:** If you want zero database setup for local dev, swap the driver to `modernc.org/sqlite` (pure Go, no CGO) and adjust the SQL types (`BIGSERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`, `TIMESTAMPTZ` → `TIMESTAMP`). Keep the repository interface identical so the rest of the app doesn't change.

---

## 3. Architecture

Layered architecture. Dependencies point **downward only**. Never let a lower layer import an upper one.

```
┌─────────────────────────────────────────────┐
│  HTTP Handlers  (internal/handlers)          │  parse request, call service, render template
├─────────────────────────────────────────────┤
│  Services       (internal/service)           │  business rules, validation, transactions
├─────────────────────────────────────────────┤
│  Repositories   (internal/repository)        │  SQL only — no business logic
├─────────────────────────────────────────────┤
│  Database       (PostgreSQL via pgxpool)     │
└─────────────────────────────────────────────┘
      ↑ Models (internal/models) are shared plain structs, no logic
```

**Request flow example — "receive stock":**
`POST /products/42/stock-in`
→ `StockHandler.StockIn` (reads form, CSRF ok, user from session)
→ `StockService.ReceiveStock(productID, qty, ref, userID)` (in a DB transaction: insert movement + update product quantity)
→ `MovementRepo.Create` + `ProductRepo.UpdateQuantity`
→ handler re-renders the product row / redirects with a flash message.

**Rules**
- Handlers never write SQL. Repositories never contain business logic.
- All multi-write operations (e.g. movement + quantity update) run inside **one transaction** in the service layer.
- Repositories accept a `context.Context` and (where needed) a transaction handle.

---

## 4. Domain Model

| Entity | Description |
|---|---|
| **User** | A person who logs in. Has a role and (optionally) belongs to one store. |
| **Store** | A shop / unit in the mall (name, unit number, floor). Products belong to a store. |
| **Category** | Product grouping (e.g. Electronics, Apparel, Grocery). |
| **Supplier** | Where products are purchased from. |
| **Product** | A stock-keeping item: SKU, prices, current quantity, reorder level. Belongs to a store. |
| **StockMovement** | An immutable audit record of every stock change (in / out / adjustment). |
| **PurchaseOrder** *(later)* | An order to a supplier to replenish stock. |

**Key relationships**
- Store `1—*` Product
- Category `1—*` Product · Supplier `1—*` Product
- Product `1—*` StockMovement
- Store `0..1—*` User

---

## 5. Database Schema

Migrations live in `/migrations` as timestamped `.up.sql` / `.down.sql` pairs. Below is the target schema (PostgreSQL).

```sql
-- 0001_stores
CREATE TABLE stores (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    unit_number   VARCHAR(50),
    floor         VARCHAR(50),
    category      VARCHAR(100),
    contact_name  VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_email VARCHAR(255),
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 0002_users
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name     VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'staff',  -- admin | manager | staff
    store_id      BIGINT REFERENCES stores(id) ON DELETE SET NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 0003_categories
CREATE TABLE categories (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 0004_suppliers
CREATE TABLE suppliers (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    contact_person VARCHAR(255),
    email          VARCHAR(255),
    phone          VARCHAR(50),
    address        TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 0005_products
CREATE TABLE products (
    id            BIGSERIAL PRIMARY KEY,
    sku           VARCHAR(100) UNIQUE NOT NULL,
    barcode       VARCHAR(100),
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    category_id   BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    supplier_id   BIGINT REFERENCES suppliers(id) ON DELETE SET NULL,
    store_id      BIGINT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    cost_price    NUMERIC(12,2) NOT NULL DEFAULT 0,
    selling_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    quantity      INTEGER     NOT NULL DEFAULT 0,
    reorder_level INTEGER     NOT NULL DEFAULT 0,
    unit          VARCHAR(50) NOT NULL DEFAULT 'pcs',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 0006_stock_movements
CREATE TABLE stock_movements (
    id             BIGSERIAL PRIMARY KEY,
    product_id     BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    movement_type  VARCHAR(20) NOT NULL,   -- in | out | adjustment
    quantity       INTEGER     NOT NULL,   -- always positive; type gives direction
    quantity_after INTEGER     NOT NULL,   -- running balance after this movement
    reference      VARCHAR(255),           -- PO no., invoice, or reason
    notes          TEXT,
    user_id        BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_products_store        ON products(store_id);
CREATE INDEX idx_products_category     ON products(category_id);
CREATE INDEX idx_products_sku          ON products(sku);
CREATE INDEX idx_products_low_stock    ON products(store_id) WHERE quantity <= reorder_level;
CREATE INDEX idx_movements_product     ON stock_movements(product_id, created_at DESC);
```

**Later (Phase 6) — purchase orders:**
```sql
CREATE TABLE purchase_orders (
    id            BIGSERIAL PRIMARY KEY,
    po_number     VARCHAR(50) UNIQUE NOT NULL,
    supplier_id   BIGINT REFERENCES suppliers(id),
    store_id      BIGINT REFERENCES stores(id),
    status        VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft|ordered|received|cancelled
    order_date    DATE,
    expected_date DATE,
    notes         TEXT,
    created_by    BIGINT REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE purchase_order_items (
    id         BIGSERIAL PRIMARY KEY,
    po_id      BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id),
    quantity   INTEGER NOT NULL,
    unit_cost  NUMERIC(12,2) NOT NULL
);
```

**Data integrity rule:** `products.quantity` is a cached running total. It must **only ever** change inside the same transaction that inserts a `stock_movements` row. Never update it directly anywhere else. `quantity_after` on each movement lets you audit and rebuild the total if needed.

---

## 6. Features by Module

### Auth & Users
- Email + password login, secure session cookie, logout.
- Roles: **admin** (everything, incl. user management), **manager** (all stock ops + reports), **staff** (view + record stock in/out).
- Admin can create/deactivate users and assign a store.

### Stores
- CRUD for mall shops/units. Deactivate instead of hard-delete when they have products.

### Categories & Suppliers
- CRUD. Used as dropdowns when creating products.

### Products
- CRUD with SKU (unique), barcode, prices, unit, reorder level, category, supplier, store.
- List view with **search** (name / SKU / barcode), **filter** (store, category, low-stock only), and **pagination**.
- Product detail page shows current stock + recent movements.

### Stock Operations
- **Stock In** (receiving): add quantity, with reference (PO/invoice) and notes.
- **Stock Out** (issue/sale/damage): remove quantity; block if it would go negative.
- **Adjustment**: set the count to a corrected value (records the delta).
- Every operation writes an immutable `stock_movements` row **and** updates the product quantity in one transaction.
- **Movement history** page, filterable by product / store / type / date range.

### Dashboard
- Cards: total products, total stock value (Σ quantity × cost_price), number of low-stock items, number of stores.
- Low-stock table (quantity ≤ reorder level).
- Recent movements feed.

### Reports
- **Low stock report** (with suggested reorder quantities).
- **Stock valuation** (by store / category).
- **Movement report** for a date range.
- **CSV export** for each report.

---

## 7. Routes / Pages

Server-rendered pages + HTMX partials. Mutating routes are `POST` (forms don't do PUT/DELETE natively).

| Method | Path | Purpose |
|---|---|---|
| GET | `/login` | Login form |
| POST | `/login` | Authenticate |
| POST | `/logout` | End session |
| GET | `/` | Dashboard |
| GET | `/products` | List (search/filter/paginate) |
| GET | `/products/new` | New product form |
| POST | `/products` | Create product |
| GET | `/products/{id}` | Product detail + movements |
| GET | `/products/{id}/edit` | Edit form |
| POST | `/products/{id}` | Update product |
| POST | `/products/{id}/delete` | Deactivate product |
| GET | `/products/{id}/stock-in` | Stock-in form |
| POST | `/products/{id}/stock-in` | Receive stock |
| GET | `/products/{id}/stock-out` | Stock-out form |
| POST | `/products/{id}/stock-out` | Issue stock |
| POST | `/products/{id}/adjust` | Adjust count |
| GET | `/movements` | Movement history |
| GET/POST | `/stores`, `/stores/new`, `/stores/{id}`, ... | Store CRUD |
| GET/POST | `/categories`, ... | Category CRUD |
| GET/POST | `/suppliers`, ... | Supplier CRUD |
| GET/POST | `/users`, ... | User management (admin) |
| GET | `/reports/low-stock` | Low stock report |
| GET | `/reports/valuation` | Valuation report |
| GET | `/reports/movements` | Movement report |
| GET | `/reports/{name}/export.csv` | CSV export |
| GET | `/healthz` | Health check |
| GET | `/static/*` | Static assets |

---

## 8. Roles & Permissions

| Capability | Admin | Manager | Staff |
|---|:---:|:---:|:---:|
| View dashboard & products | ✅ | ✅ | ✅ |
| Record stock in / out / adjust | ✅ | ✅ | ✅ |
| Create / edit products | ✅ | ✅ | ❌ |
| Manage stores / categories / suppliers | ✅ | ✅ | ❌ |
| View reports & export | ✅ | ✅ | ✅ (view) |
| Manage users | ✅ | ❌ | ❌ |

Enforce with middleware that reads the role from the session and gates route groups. Also scope data by `store_id` for non-admins if you want store isolation (decide in Phase 1).

---

## 9. Roadmap (Phases & Checklists)

Build in order. Each phase should end with a working, testable app.

### Phase 0 — Project Setup
- [ ] `go mod init`, create directory structure ([§10](#10-project-structure)).
- [ ] Add `chi`, `pgx`, `golang-migrate`.
- [ ] Config loading from env (`.env` for local via `godotenv`).
- [ ] Postgres connection pool + `/healthz`.
- [ ] Migration tooling wired into the Makefile.
- [ ] Base layout template + Tailwind + static file serving.
- [ ] Server boots and serves a "Hello, mall" page.

### Phase 1 — Auth & Users
- [ ] `users` + `stores` migrations.
- [ ] bcrypt password hashing; seed one admin user.
- [ ] Sessions (scs) + login/logout.
- [ ] Auth middleware (require login) + role middleware.
- [ ] CSRF protection on all forms.
- [ ] User management pages (admin).

### Phase 2 — Core Catalog
- [ ] `categories`, `suppliers`, `products` migrations.
- [ ] CRUD for stores, categories, suppliers.
- [ ] CRUD for products (with dropdowns).
- [ ] Product list with search + filter + pagination.

### Phase 3 — Stock Operations (the core)
- [ ] `stock_movements` migration.
- [ ] Stock-in, stock-out (with negative-stock guard), adjustment — all transactional.
- [ ] Product detail shows current qty + recent movements.
- [ ] Movement history page with filters.

### Phase 4 — Dashboard & Reports
- [ ] Dashboard cards + low-stock table + recent movements.
- [ ] Low-stock, valuation, and movement reports.
- [ ] CSV export.

### Phase 5 — Polish
- [ ] Flash messages, form validation errors inline.
- [ ] Empty states, loading indicators (HTMX).
- [ ] Barcode field search; keyboard-friendly forms.
- [ ] Seed script with realistic sample data.
- [ ] Tests for services (stock math) + key handlers.

### Phase 6 — Advanced (optional)
- [ ] Purchase orders + receiving against a PO.
- [ ] Stock transfer between stores.
- [ ] Low-stock email/notification.
- [ ] Audit log of user actions.

---

## 10. Project Structure

```
mall-stock/
├── cmd/
│   └── server/
│       └── main.go            # entrypoint: wire config, db, router, start server
├── internal/
│   ├── config/                # env loading into a Config struct
│   ├── database/              # pgxpool setup, migration runner helpers
│   ├── models/                # plain structs: User, Store, Product, StockMovement...
│   ├── repository/            # SQL per entity (ProductRepo, MovementRepo, ...)
│   ├── service/               # business logic (StockService, ProductService, AuthService)
│   ├── handlers/              # HTTP handlers, grouped by resource
│   ├── middleware/            # auth, role, CSRF, request logging, recover
│   └── templates/             # template parsing + render helpers
├── migrations/                # 0001_*.up.sql / 0001_*.down.sql ...
├── web/
│   ├── templates/
│   │   ├── layouts/           # base.html
│   │   ├── pages/             # dashboard.html, products/list.html, ...
│   │   └── partials/          # product_row.html, flash.html (HTMX fragments)
│   └── static/
│       ├── css/               # tailwind output
│       ├── js/                # htmx.min.js, small app.js
│       └── img/
├── scripts/
│   └── seed.go                # sample data
├── .env.example
├── .gitignore
├── go.mod
├── Makefile
├── CLAUDE.md                  # instructions for Claude Code
├── plan.md                    # this file
└── README.md
```

---

## 11. Security

- **Passwords:** bcrypt (cost ≥ 12). Never store or log plaintext.
- **Sessions:** `HttpOnly`, `Secure` (in prod), `SameSite=Lax` cookies; server-side session store.
- **CSRF:** token on every state-changing form (nosurf).
- **SQL injection:** parameterized queries only — never build SQL with string concatenation of user input.
- **XSS:** rely on `html/template` auto-escaping; never use `template.HTML` on user data.
- **AuthZ:** check role on every protected route, not just hide the button.
- **Secrets:** only via env vars; `.env` is git-ignored; commit `.env.example` with blanks.
- **Input validation:** validate and bound all numbers (quantities ≥ 0, prices ≥ 0) in the service layer.

---

## 12. Testing

- **Unit tests** for services — especially stock math (in/out/adjust, negative guard, `quantity_after`).
- **Repository tests** against a real Postgres (use a test database or a disposable container).
- **Handler tests** with `net/http/httptest` for auth, redirects, and validation errors.
- Table-driven tests (idiomatic Go). Aim for meaningful coverage of the stock engine before breadth.
- `make test` must stay green before each commit.

---

## 13. Configuration & Environment

`.env.example`:
```env
# Server
APP_ENV=development
PORT=8080
SESSION_SECRET=change-me-to-a-long-random-string

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/mall_stock?sslmode=disable

# Optional
LOG_LEVEL=info
```

**Deployment (brief):** build a static binary (`CGO_ENABLED=0 go build`), ship it in a small container or systemd service behind a reverse proxy (Caddy/Nginx for TLS). Run `migrate up` on deploy. Embed templates/static with `embed.FS` so the binary is self-contained.

---

## 14. Future Enhancements
- Barcode scanning via device camera (HTML5) for fast stock-in/out.
- Multi-warehouse / stock transfer workflows with approvals.
- Supplier price history & auto-reorder suggestions.
- Email/SMS low-stock alerts.
- Full user-action audit trail.
- REST/JSON API for integrations or a future mobile app.
- Batch import/export of products via CSV.
- Charts on the dashboard (stock trends over time).

---

## 15. Getting Started (once code exists)

```bash
cp .env.example .env          # fill in values
createdb mall_stock           # or use docker
make migrate-up               # apply schema
make seed                     # load sample data (creates admin user)
make dev                      # run with live reload → http://localhost:8080
```

Default admin (from seed): `admin@mall.local` / `admin123` — **change immediately.**

---

**Next step:** open this repo in Claude Code and start at **Phase 0**. `CLAUDE.md` tells the agent how to work in this codebase.
