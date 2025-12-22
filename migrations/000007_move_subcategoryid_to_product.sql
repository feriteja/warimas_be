

-- +migrate Up

BEGIN;

-- 1️⃣ Add subcategory_id column to products
ALTER TABLE products
ADD COLUMN subcategory_id uuid;

-- 2️⃣ Copy existing subcategory data from variants → products
UPDATE products p
SET subcategory_id = v.subcategory_id
FROM (
    SELECT DISTINCT ON (product_id)
           product_id,
           subcategory_id
    FROM variants
    WHERE subcategory_id IS NOT NULL
    ORDER BY product_id, subcategory_id
) v
WHERE p.id = v.product_id;

-- 3️⃣ Add foreign key to products
ALTER TABLE products
ADD CONSTRAINT products_subcategory_id_fkey
FOREIGN KEY (subcategory_id)
REFERENCES subcategories(id)
ON DELETE SET NULL;

-- 4️⃣ Drop foreign key and column from variants
ALTER TABLE variants
DROP CONSTRAINT IF EXISTS variants_subcategory_id_fkey;

ALTER TABLE variants
DROP COLUMN IF EXISTS subcategory_id;

COMMIT;

-- +migrate Down
 BEGIN;

-- 1️⃣ Re-add subcategory_id to variants
ALTER TABLE variants
ADD COLUMN subcategory_id uuid;

-- 2️⃣ Copy data back from products → variants
UPDATE variants v
SET subcategory_id = p.subcategory_id
FROM products p
WHERE v.product_id = p.id;

-- 3️⃣ Recreate foreign key on variants
ALTER TABLE variants
ADD CONSTRAINT variants_subcategory_id_fkey
FOREIGN KEY (subcategory_id)
REFERENCES subcategories(id)
ON DELETE SET NULL;

-- 4️⃣ Remove subcategory_id from products
ALTER TABLE products
DROP CONSTRAINT IF EXISTS products_subcategory_id_fkey;

ALTER TABLE products
DROP COLUMN IF EXISTS subcategory_id;

COMMIT;
