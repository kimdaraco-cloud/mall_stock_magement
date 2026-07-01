-- @ai-modified 2026-07-02 create stock_movements audit table
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

CREATE INDEX idx_movements_product ON stock_movements(product_id, created_at DESC);
CREATE INDEX idx_movements_created ON stock_movements(created_at DESC);
