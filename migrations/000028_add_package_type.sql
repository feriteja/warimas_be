-- +migrate Up
ALTER TABLE packages ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'personal';
ALTER TABLE packages ADD CONSTRAINT chk_packages_type CHECK (type IN ('personal', 'promotion'));
ALTER TABLE packages ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE packages ADD COLUMN deleted_at TIMESTAMPTZ;

-- +migrate Down
ALTER TABLE packages DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE packages DROP COLUMN IF EXISTS is_active;
ALTER TABLE packages DROP CONSTRAINT IF EXISTS chk_packages_type;
ALTER TABLE packages DROP COLUMN IF EXISTS type;