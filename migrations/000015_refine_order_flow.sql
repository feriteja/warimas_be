
-- +migrate Up
BEGIN;

-- =========================
-- CHECKOUT SESSIONS
-- =========================

-- Rename total_price → total_amount (only if exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'checkout_sessions'
          AND column_name = 'total_price'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'checkout_sessions'
          AND column_name = 'total_amount'
    ) THEN
        ALTER TABLE checkout_sessions
            RENAME COLUMN total_price TO total_amount;
    END IF;
END $$;

-- Money columns to BIGINT
ALTER TABLE checkout_sessions
    ALTER COLUMN subtotal TYPE BIGINT USING subtotal::BIGINT,
    ALTER COLUMN tax TYPE BIGINT USING tax::BIGINT,
    ALTER COLUMN shipping_fee TYPE BIGINT USING shipping_fee::BIGINT,
    ALTER COLUMN discount TYPE BIGINT USING discount::BIGINT,
    ALTER COLUMN total_amount TYPE BIGINT USING total_amount::BIGINT;

-- Currency normalization
ALTER TABLE checkout_sessions
    ALTER COLUMN currency TYPE varchar(3);

-- Timezone correctness
ALTER TABLE checkout_sessions
    ALTER COLUMN created_at TYPE timestamptz,
    ALTER COLUMN updated_at TYPE timestamptz,
    ALTER COLUMN expires_at TYPE timestamptz;

-- Remove payment coupling
ALTER TABLE checkout_sessions
    DROP COLUMN IF EXISTS payment_ref;

-- =========================
-- ORDERS
-- =========================

-- Rename total_price → total_amount
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'orders'
          AND column_name = 'total_price'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'orders'
          AND column_name = 'total_amount'
    ) THEN
        ALTER TABLE orders
            RENAME COLUMN total_price TO total_amount;
    END IF;
END $$;

-- Money type
ALTER TABLE orders
    ALTER COLUMN total_amount TYPE BIGINT USING total_amount::BIGINT;

-- Currency
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS currency varchar(3);

UPDATE orders
SET currency = 'IDR'
WHERE currency IS NULL;

ALTER TABLE orders
    ALTER COLUMN currency SET NOT NULL;

-- External ID
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS external_id varchar(50);

UPDATE orders
SET external_id = 'ORD-' || id::text
WHERE external_id IS NULL;

ALTER TABLE orders
    ALTER COLUMN external_id SET NOT NULL;

-- Unique index
CREATE UNIQUE INDEX IF NOT EXISTS ux_orders_external_id
    ON orders (external_id);

-- Enforce session reference
ALTER TABLE orders
    ALTER COLUMN checkout_session_id SET NOT NULL;

-- Remove payment coupling
ALTER TABLE orders
    DROP COLUMN IF EXISTS payment_provider,
    DROP COLUMN IF EXISTS payment_reference,
    DROP COLUMN IF EXISTS payment_status;

-- Timezone correctness
ALTER TABLE orders
    ALTER COLUMN created_at TYPE timestamptz,
    ALTER COLUMN updated_at TYPE timestamptz;

-- =========================
-- PAYMENTS
-- =========================

-- Rename external_id → external_reference
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'payments'
          AND column_name = 'external_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'payments'
          AND column_name = 'external_reference'
    ) THEN
        ALTER TABLE payments
            RENAME COLUMN external_id TO external_reference;
    END IF;
END $$;

-- Order linkage
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS order_id UUID;

-- FK (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_payments_order'
    ) THEN
        ALTER TABLE payments
            ADD CONSTRAINT fk_payments_order
                FOREIGN KEY (order_id) REFERENCES orders(id);
    END IF;
END $$;

ALTER TABLE payments
    ALTER COLUMN order_id SET NOT NULL;

-- Provider & currency
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS provider varchar(50),
    ADD COLUMN IF NOT EXISTS currency varchar(3);

UPDATE payments
SET provider = 'xendit',
    currency = 'IDR'
WHERE provider IS NULL OR currency IS NULL;

ALTER TABLE payments
    ALTER COLUMN provider SET NOT NULL,
    ALTER COLUMN currency SET NOT NULL;

-- Status
ALTER TABLE payments
    ALTER COLUMN status TYPE varchar(30);

-- Timezone correctness
ALTER TABLE payments
    ALTER COLUMN created_at TYPE timestamptz,
    ALTER COLUMN updated_at TYPE timestamptz;

-- Idempotency
CREATE UNIQUE INDEX IF NOT EXISTS ux_payments_gateway_ref
    ON payments (provider, external_reference);

COMMIT;



-- +migrate Down
BEGIN;

-- =========================
-- PAYMENTS
-- =========================

DROP INDEX IF EXISTS ux_payments_gateway_ref;

ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS fk_payments_order;

ALTER TABLE payments
    DROP COLUMN IF EXISTS order_id,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS currency;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'payments'
          AND column_name = 'external_reference'
    ) THEN
        ALTER TABLE payments
            RENAME COLUMN external_reference TO external_id;
    END IF;
END $$;

ALTER TABLE payments
    ALTER COLUMN created_at TYPE timestamp,
    ALTER COLUMN updated_at TYPE timestamp;

-- =========================
-- ORDERS
-- =========================

DROP INDEX IF EXISTS ux_orders_external_id;

ALTER TABLE orders
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS currency;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payment_provider varchar(50),
    ADD COLUMN IF NOT EXISTS payment_reference varchar(100),
    ADD COLUMN IF NOT EXISTS payment_status varchar(30);

ALTER TABLE orders
    ALTER COLUMN checkout_session_id DROP NOT NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'orders'
          AND column_name = 'total_amount'
    ) THEN
        ALTER TABLE orders
            RENAME COLUMN total_amount TO total_price;
    END IF;
END $$;

ALTER TABLE orders
    ALTER COLUMN total_price TYPE numeric(14,2)
    USING total_price::numeric;

ALTER TABLE orders
    ALTER COLUMN created_at TYPE timestamp,
    ALTER COLUMN updated_at TYPE timestamp;

-- =========================
-- CHECKOUT SESSIONS
-- =========================

ALTER TABLE checkout_sessions
    ADD COLUMN IF NOT EXISTS payment_ref varchar(100);

ALTER TABLE checkout_sessions
    ALTER COLUMN created_at TYPE timestamp,
    ALTER COLUMN updated_at TYPE timestamp,
    ALTER COLUMN expires_at TYPE timestamp;

ALTER TABLE checkout_sessions
    ALTER COLUMN currency TYPE char(3);

ALTER TABLE checkout_sessions
    ALTER COLUMN subtotal TYPE numeric(14,2) USING subtotal::numeric,
    ALTER COLUMN tax TYPE numeric(14,2) USING tax::numeric,
    ALTER COLUMN shipping_fee TYPE numeric(14,2) USING shipping_fee::numeric,
    ALTER COLUMN discount TYPE numeric(14,2) USING discount::numeric,
    ALTER COLUMN total_amount TYPE numeric(14,2) USING total_amount::numeric;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'checkout_sessions'
          AND column_name = 'total_amount'
    ) THEN
        ALTER TABLE checkout_sessions
            RENAME COLUMN total_amount TO total_price;
    END IF;
END $$;

COMMIT;
