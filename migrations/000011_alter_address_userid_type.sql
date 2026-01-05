-- +migrate Up
BEGIN;

-- 1. Add new integer column for user
ALTER TABLE addresses
ADD COLUMN user_id_int INTEGER;



-- 3. Drop old UUID user_id
ALTER TABLE addresses
DROP COLUMN user_id;

-- 4. Rename int column
ALTER TABLE addresses
RENAME COLUMN user_id_int TO user_id;

-- 5. Add guest_id for guest flow
ALTER TABLE addresses   
ADD COLUMN guest_id UUID;

-- 6. Foreign key for registered users
ALTER TABLE addresses
ADD CONSTRAINT fk_address_user
FOREIGN KEY (user_id)
REFERENCES users(id)
ON DELETE CASCADE;

-- 7. Sanity check: positive user_id
ALTER TABLE addresses
ADD CONSTRAINT address_user_id_positive
CHECK (user_id IS NULL OR user_id > 0);

-- 8. Ensure address belongs to EITHER user OR guest (not both)
ALTER TABLE addresses
ADD CONSTRAINT address_owner_check
CHECK (
    (user_id IS NOT NULL AND guest_id IS NULL)
 OR (user_id IS NULL AND guest_id IS NOT NULL)
 OR (user_id IS NULL AND guest_id IS NULL)
);

COMMIT;



-- +migrate Down
BEGIN;

-- 1. Restore UUID user_id
ALTER TABLE addresses
ADD COLUMN user_id_uuid UUID;


-- 2. Drop constraints
ALTER TABLE addresses
DROP CONSTRAINT IF EXISTS address_owner_check;
ALTER TABLE addresses
DROP CONSTRAINT IF EXISTS address_user_id_positive;
ALTER TABLE addresses
DROP CONSTRAINT IF EXISTS fk_address_user;

-- 3. Drop new columns
ALTER TABLE addresses
DROP COLUMN user_id;
ALTER TABLE addresses
DROP COLUMN guest_id;

-- 4. Restore old column name
ALTER TABLE addresses
RENAME COLUMN user_id_uuid TO user_id;

COMMIT;
