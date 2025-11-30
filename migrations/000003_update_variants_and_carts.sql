-- +migrate Up
BEGIN;

-- 1. ALTER TABLE variants: add subcategory_id column
ALTER TABLE variants
ADD COLUMN subcategory_id UUID;

-- Add FK to categories
ALTER TABLE variants
ADD CONSTRAINT variants_subcategory_id_fkey
FOREIGN KEY (subcategory_id)
REFERENCES subcategories(id)
ON DELETE SET NULL;

----------------------------------------------------

-- 2. ALTER TABLE carts: remove product_id and add variant_id
ALTER TABLE carts
DROP COLUMN IF EXISTS product_id;

ALTER TABLE carts
ADD COLUMN variant_id UUID;

-- Add foreign key to variants table
ALTER TABLE carts
ADD CONSTRAINT carts_variant_id_fkey
FOREIGN KEY (variant_id)
REFERENCES variants(id)
ON DELETE CASCADE;

COMMIT;

-- +migrate Down
BEGIN;

-- Remove FK from carts
ALTER TABLE carts
DROP CONSTRAINT IF EXISTS carts_variant_id_fkey;

-- Remove column variant_id
ALTER TABLE carts
DROP COLUMN IF EXISTS variant_id;

-- Re-add product_id (no type guessing, adjust if needed)
ALTER TABLE carts
ADD COLUMN product_id UUID;

-- (No FK because we don’t know original table design)

----------------------------------------------------

-- Remove FK from variants
ALTER TABLE variants
DROP CONSTRAINT IF EXISTS variants_subcategory_id_fkey;

-- Remove column subcategory_id
ALTER TABLE variants
DROP COLUMN IF EXISTS subcategory_id;

COMMIT;
