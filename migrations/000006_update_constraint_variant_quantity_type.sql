-- +migrate Up

-- Normalize existing data
UPDATE variants
SET quantity_type = LOWER(quantity_type);

-- Remove old constraint
ALTER TABLE variants
DROP CONSTRAINT IF EXISTS variants_quantity_type_check;

-- Set new default
ALTER TABLE variants
ALTER COLUMN quantity_type SET DEFAULT 'unit';

-- Add new constraint
ALTER TABLE variants
ADD CONSTRAINT variants_quantity_type_check
CHECK (quantity_type IN ('unit', 'kg', 'liter', 'sack'));

-- +migrate Down

-- Remove new constraint
ALTER TABLE variants
DROP CONSTRAINT IF EXISTS variants_quantity_type_check;

-- Restore default
ALTER TABLE variants
ALTER COLUMN quantity_type SET DEFAULT 'UNIT';

-- Restore old constraint
ALTER TABLE variants
ADD CONSTRAINT variants_quantity_type_check
CHECK (quantity_type = 'UNIT');
