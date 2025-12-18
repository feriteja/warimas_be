-- +migrate Up

-- Enable extension for UUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE sellers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id INTEGER NOT NULL,
    name VARCHAR(150) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT fk_sellers_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Enforce one active seller per user
CREATE UNIQUE INDEX uniq_sellers_user_id
    ON sellers(user_id)
    WHERE deleted_at IS NULL;

-- Soft delete index
CREATE INDEX idx_sellers_deleted_at
    ON sellers(deleted_at);

-- Trigger function (namespaced)
CREATE OR REPLACE FUNCTION sellers_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sellers_updated_at
BEFORE UPDATE ON sellers
FOR EACH ROW
EXECUTE FUNCTION sellers_set_updated_at();

-- +migrate Down

DROP TRIGGER IF EXISTS trg_sellers_updated_at ON sellers;
DROP FUNCTION IF EXISTS sellers_set_updated_at();

DROP INDEX IF EXISTS idx_sellers_deleted_at;
DROP INDEX IF EXISTS uniq_sellers_user_id;

DROP TABLE IF EXISTS sellers;
