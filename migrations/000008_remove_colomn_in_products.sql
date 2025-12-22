-- +migrate Up
-- Remove price and stock from products, add description to variants
BEGIN;

-- Remove columns from products
ALTER TABLE products
DROP COLUMN IF EXISTS price;

ALTER TABLE products
DROP COLUMN IF EXISTS stock;

-- Add description column to variants
ALTER TABLE variants
ADD COLUMN description text;

COMMIT;


-- +migrate Down
-- Revert changes: add price and stock back to products, remove description from variants
BEGIN;

-- Add columns back to products
ALTER TABLE products
ADD COLUMN price numeric(10,2) DEFAULT 0 NOT NULL;

ALTER TABLE products
ADD COLUMN stock integer DEFAULT 0 NOT NULL;

-- Remove description column from variants
ALTER TABLE variants
DROP COLUMN IF EXISTS description;

COMMIT;
