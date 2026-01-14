-- +migrate Up
BEGIN;

-- 1. Add image_url to order_items
ALTER TABLE order_items
ADD COLUMN image_url TEXT;

-- 2. Add deleted_at to orders (soft delete)
ALTER TABLE orders
ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- 3. Partial index for non-deleted orders
CREATE INDEX idx_orders_not_deleted
ON orders (created_at)
WHERE deleted_at IS NULL;

COMMIT;


-- +migrate Down
BEGIN;

-- Drop partial index
DROP INDEX IF EXISTS idx_orders_not_deleted;

-- Remove columns
ALTER TABLE order_items
DROP COLUMN image_url;

ALTER TABLE orders
DROP COLUMN deleted_at;

COMMIT;
