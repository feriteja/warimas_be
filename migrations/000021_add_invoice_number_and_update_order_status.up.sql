-- +migrate Up
BEGIN;

-- 1. Add nullable invoice_number
ALTER TABLE orders
ADD COLUMN invoice_number TEXT;

-- 2. Enforce uniqueness (NULL allowed)
ALTER TABLE orders
ADD CONSTRAINT orders_invoice_number_unique UNIQUE (invoice_number);

-- 3. Drop DEFAULT on status (IMPORTANT)
ALTER TABLE orders
ALTER COLUMN status DROP DEFAULT;

-- 4. Create new enum
CREATE TYPE order_status_new AS ENUM (
  'PENDING_PAYMENT',
  'PAID',
  'ACCEPTED',
  'SHIPPED',
  'COMPLETED',
  'CANCELLED',
  'FAILED'
);

-- 5. Migrate status column
ALTER TABLE orders
ALTER COLUMN status TYPE order_status_new
USING (
  CASE status::text
    WHEN 'PENDING' THEN 'PENDING_PAYMENT'
    ELSE status::text
  END
)::order_status_new;

-- 6. Restore DEFAULT (adjust if needed)
ALTER TABLE orders
ALTER COLUMN status SET DEFAULT 'PENDING_PAYMENT';

-- 7. Replace enum
DROP TYPE order_status;
ALTER TYPE order_status_new RENAME TO order_status;

-- 8. Prevent invoice overwrite
CREATE OR REPLACE FUNCTION prevent_invoice_update()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.invoice_number IS NOT NULL
     AND NEW.invoice_number <> OLD.invoice_number THEN
    RAISE EXCEPTION 'invoice_number cannot be modified once set';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_invoice_update
BEFORE UPDATE OF invoice_number ON orders
FOR EACH ROW
EXECUTE FUNCTION prevent_invoice_update();

COMMIT;


-- +migrate Down
BEGIN;

-- Drop protection trigger
DROP TRIGGER IF EXISTS trg_prevent_invoice_update ON orders;
DROP FUNCTION IF EXISTS prevent_invoice_update;

-- Drop unique constraint
ALTER TABLE orders
DROP CONSTRAINT IF EXISTS orders_invoice_number_unique;

-- Drop column
ALTER TABLE orders
DROP COLUMN invoice_number;

-- Drop DEFAULT before enum rollback
ALTER TABLE orders
ALTER COLUMN status DROP DEFAULT;

-- Recreate old enum
CREATE TYPE order_status_old AS ENUM (
  'PENDING',
  'PAID',
  'SHIPPED',
  'COMPLETED',
  'CANCELLED',
  'FAILED'
);

-- Revert status
ALTER TABLE orders
ALTER COLUMN status TYPE order_status_old
USING (
  CASE status::text
    WHEN 'PENDING_PAYMENT' THEN 'PENDING'
    ELSE status::text
  END
)::order_status_old;

-- Restore old DEFAULT
ALTER TABLE orders
ALTER COLUMN status SET DEFAULT 'PENDING';

DROP TYPE order_status;
ALTER TYPE order_status_old RENAME TO order_status;

COMMIT;
