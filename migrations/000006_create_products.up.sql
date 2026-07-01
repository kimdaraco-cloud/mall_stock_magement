-- @ai-modified 2026-07-02 create products table with indexes
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

CREATE INDEX idx_products_store     ON products(store_id);
CREATE INDEX idx_products_category  ON products(category_id);
CREATE INDEX idx_products_sku       ON products(sku);
CREATE INDEX idx_products_low_stock ON products(store_id) WHERE quantity <= reorder_level;
