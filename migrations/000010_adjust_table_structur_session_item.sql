-- +migrate Up
-- =========================================
-- ADJUST CHECKOUT SESSION ITEMS STRUCTURE
-- =========================================

-- 1. Add new columns
ALTER TABLE checkout_session_items
  ADD COLUMN variant_id UUID,
  ADD COLUMN variant_name TEXT,
  ADD COLUMN imageurl TEXT,
  ADD COLUMN quantity_type TEXT;

-- 2. Backfill variant_id from product_id (legacy compatibility)
UPDATE checkout_session_items
  SET variant_id = product_id;

-- 3. Enforce NOT NULL on variant_id
ALTER TABLE checkout_session_items
  ALTER COLUMN variant_id SET NOT NULL;

-- 4. Drop old product_id column
ALTER TABLE checkout_session_items
  DROP COLUMN product_id;


-- +migrate Down
-- =========================================
-- ROLLBACK CHECKOUT SESSION ITEMS STRUCTURE
-- =========================================

-- 1. Re-add product_id column
ALTER TABLE checkout_session_items
  ADD COLUMN product_id UUID;

-- 2. Backfill product_id from variant_id
UPDATE checkout_session_items
  SET product_id = variant_id;

-- 3. Enforce NOT NULL on product_id
ALTER TABLE checkout_session_items
  ALTER COLUMN product_id SET NOT NULL;

-- 4. Drop new columns
ALTER TABLE checkout_session_items
  DROP COLUMN variant_id,
  DROP COLUMN variant_name,
  DROP COLUMN imageurl,
  DROP COLUMN quantity_type;
