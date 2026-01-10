-- +migrate Up
BEGIN;

-- 0. Enable pgcrypto if not exists (SAFE)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Add external_id column
ALTER TABLE checkout_sessions
ADD COLUMN external_id VARCHAR(32);

-- 2. Backfill existing rows
UPDATE checkout_sessions
SET external_id = 'CK_' || substring(
    encode(digest(id::text, 'sha1'), 'hex')
    FROM 1 FOR 16
)
WHERE external_id IS NULL;

-- 3. Make it NOT NULL
ALTER TABLE checkout_sessions
ALTER COLUMN external_id SET NOT NULL;

-- 4. Add UNIQUE constraint (creates unique index)
ALTER TABLE checkout_sessions
ADD CONSTRAINT checkout_sessions_external_id_unique UNIQUE (external_id);

-- 5. Optional explicit index (usually not required)
CREATE INDEX checkout_sessions_external_id_idx
ON checkout_sessions (external_id);

COMMIT;


-- +migrate Down

BEGIN;

DROP INDEX IF EXISTS checkout_sessions_external_id_idx;

ALTER TABLE checkout_sessions
DROP CONSTRAINT IF EXISTS checkout_sessions_external_id_unique;

ALTER TABLE checkout_sessions
DROP COLUMN IF EXISTS external_id;

COMMIT;
