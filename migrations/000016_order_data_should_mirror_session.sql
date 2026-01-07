-- +migrate Up
BEGIN;

-- =====================================================
-- ORDER STATUS ENUM
-- =====================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type WHERE typname = 'order_status'
    ) THEN
        CREATE TYPE order_status AS ENUM (
            'PENDING_PAYMENT',
            'PAID',
            'FULFILLING',
            'COMPLETED',
            'CANCELLED'
        );
    END IF;
END$$;

-- =====================================================
-- VALIDATE EXISTING STATUS DATA
-- =====================================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM orders
        WHERE status NOT IN (
            'PENDING_PAYMENT',
            'PAID',
            'FULFILLING',
            'COMPLETED',
            'CANCELLED'
        )
    ) THEN
        RAISE EXCEPTION 'Invalid order status found in orders table';
    END IF;
END$$;

-- =====================================================
-- CONVERT STATUS TO ENUM (SAFE)
-- =====================================================
ALTER TABLE orders
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE orders
    ALTER COLUMN status
    TYPE order_status
    USING status::order_status;

ALTER TABLE orders
    ALTER COLUMN status
    SET DEFAULT 'PENDING_PAYMENT';

-- =====================================================
-- ADD NEW COLUMNS
-- =====================================================
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS subtotal BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS shipping_fee BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS address_id UUID NOT NULL;

-- =====================================================
-- INDEXES
-- =====================================================
CREATE INDEX IF NOT EXISTS idx_orders_status
    ON orders(status);

CREATE INDEX IF NOT EXISTS idx_orders_address_id
    ON orders(address_id);

-- =====================================================
-- FOREIGN KEY (POSTGRES-SAFE)
-- =====================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_orders_address_id'
          AND table_name = 'orders'
    ) THEN
        ALTER TABLE orders
        ADD CONSTRAINT fk_orders_address_id
        FOREIGN KEY (address_id)
        REFERENCES addresses(id)
        ON DELETE SET NULL;
    END IF;
END$$;

COMMIT;

-- +migrate Down
BEGIN;

-- =====================================================
-- DROP FOREIGN KEY
-- =====================================================
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS fk_orders_address_id;

-- =====================================================
-- DROP INDEXES
-- =====================================================
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_address_id;

-- =====================================================
-- REVERT STATUS COLUMN
-- =====================================================
ALTER TABLE orders
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE orders
    ALTER COLUMN status
    TYPE TEXT
    USING status::text;

ALTER TABLE orders
    ALTER COLUMN status
    SET DEFAULT 'PENDING_PAYMENT';

-- =====================================================
-- DROP ADDED COLUMNS
-- =====================================================
ALTER TABLE orders
    DROP COLUMN IF EXISTS address_id,
    DROP COLUMN IF EXISTS discount,
    DROP COLUMN IF EXISTS shipping_fee,
    DROP COLUMN IF EXISTS tax,
    DROP COLUMN IF EXISTS subtotal;

-- =====================================================
-- DROP ENUM ONLY IF UNUSED
-- =====================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_attribute a
        JOIN pg_type t ON a.atttypid = t.oid
        WHERE t.typname = 'order_status'
    ) THEN
        DROP TYPE order_status;
    END IF;
END$$;

COMMIT;
