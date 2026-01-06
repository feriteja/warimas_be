-- +migrate Up
BEGIN;

-- =========================
-- ORDERS TABLE
-- =========================

-- 1. Add checkout_session_id for idempotency & traceability
ALTER TABLE orders
ADD COLUMN checkout_session_id uuid;

-- 2. Make user_id nullable (guest checkout support)
ALTER TABLE orders
ALTER COLUMN user_id DROP NOT NULL;

-- 3. Rename total -> total_price (clearer meaning)
ALTER TABLE orders
RENAME COLUMN total TO total_price;

-- 4. Add payment reference fields (minimal & safe)
ALTER TABLE orders
ADD COLUMN payment_provider varchar(50),
ADD COLUMN payment_reference varchar(100),
ADD COLUMN payment_status varchar(20);

-- 5. Unique constraint to prevent duplicate orders
ALTER TABLE orders
ADD CONSTRAINT uniq_orders_checkout_session
UNIQUE (checkout_session_id);

-- =========================
-- ORDER ITEMS TABLE
-- =========================

-- 6. Snapshot fields (VERY IMPORTANT)
ALTER TABLE order_items
ADD COLUMN variant_name varchar(255),
ADD COLUMN product_name varchar(255);

-- 7. Rename unit_price -> unit_price (keep if already named correctly)
-- (skip if already correct)

-- 8. Add subtotal (price * quantity)
ALTER TABLE order_items
ADD COLUMN subtotal numeric(14,2);

-- =========================
-- INDEXES
-- =========================

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

COMMIT;

-- +migrate Down
BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_order_items_order_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_user_id;

-- Order items rollback
ALTER TABLE order_items
DROP COLUMN IF EXISTS subtotal,
DROP COLUMN IF EXISTS product_name,
DROP COLUMN IF EXISTS variant_name;

-- Orders rollback
ALTER TABLE orders
DROP CONSTRAINT IF EXISTS uniq_orders_checkout_session;

ALTER TABLE orders
DROP COLUMN IF EXISTS payment_status,
DROP COLUMN IF EXISTS payment_reference,
DROP COLUMN IF EXISTS payment_provider,
DROP COLUMN IF EXISTS checkout_session_id;

ALTER TABLE orders
RENAME COLUMN total_price TO total;

ALTER TABLE orders
ALTER COLUMN user_id SET NOT NULL;

COMMIT;
