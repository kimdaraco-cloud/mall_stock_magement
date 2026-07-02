-- @ai-modified 2026-07-02 add signed quantity_delta to stock_movements, backfilled from quantity_after
ALTER TABLE stock_movements ADD COLUMN quantity_delta INTEGER;

-- Backfill without touching existing audited values: direction is derived by
-- comparing each row's quantity_after with the previous movement's (0 for the
-- product's first movement). For in/out rows the type already gives the sign.
UPDATE stock_movements m
SET quantity_delta = CASE m.movement_type
    WHEN 'in'  THEN m.quantity
    WHEN 'out' THEN -m.quantity
    ELSE m.quantity_after - COALESCE(prev.prev_after, 0)
END
FROM (
    SELECT id,
           LAG(quantity_after) OVER (PARTITION BY product_id ORDER BY created_at, id) AS prev_after
    FROM stock_movements
) prev
WHERE prev.id = m.id;

ALTER TABLE stock_movements ALTER COLUMN quantity_delta SET NOT NULL;
