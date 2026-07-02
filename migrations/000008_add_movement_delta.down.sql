-- @ai-modified 2026-07-02 drop quantity_delta column
ALTER TABLE stock_movements DROP COLUMN IF EXISTS quantity_delta;
