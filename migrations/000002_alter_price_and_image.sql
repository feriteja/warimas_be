-- +migrate Up
-- ============= PRICE TYPE CHANGES =============
-- Change products.price from INTEGER → NUMERIC(12,2)
ALTER TABLE products
ALTER COLUMN price TYPE NUMERIC(12,2)
USING price::numeric;

-- Standardize variants.price to NUMERIC(12,2)
ALTER TABLE variants
ALTER COLUMN price TYPE NUMERIC(12,2)
USING price::numeric;


-- ============= IMAGE URL CHANGES =============
-- Add product.imageUrl
ALTER TABLE products
ADD COLUMN imageUrl TEXT;

-- Rename variants.image → imageUrl
ALTER TABLE variants
RENAME COLUMN image TO imageUrl;



-- +migrate Down

-- ============= REVERT PRICE TYPE =============
-- Revert products.price NUMERIC → INTEGER
ALTER TABLE products
ALTER COLUMN price TYPE INTEGER
USING ROUND(price)::integer;

-- Revert variants.price NUMERIC → INTEGER
ALTER TABLE variants
ALTER COLUMN price TYPE INTEGER
USING ROUND(price)::integer;


-- ============= REVERT IMAGE URL CHANGES =============
-- Remove product.imageUrl
ALTER TABLE products
DROP COLUMN imageUrl;

-- Rename variants.imageUrl → image
ALTER TABLE variants
RENAME COLUMN imageUrl TO image;
