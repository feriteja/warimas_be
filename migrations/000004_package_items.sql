-- +migrate Up

-- Create table: packages
CREATE TABLE packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    image_url TEXT,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create table: package_items
CREATE TABLE package_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id UUID NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(package_id, variant_id)           -- optional but useful
);

-- ================================================
-- Auto-update updated_at Trigger Function
-- ================================================
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Triggers for packages
-- ================================================
CREATE TRIGGER trg_packages_updated_at
BEFORE UPDATE ON packages
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

-- ================================================
-- Triggers for package_items
-- ================================================
CREATE TRIGGER trg_package_items_updated_at
BEFORE UPDATE ON package_items
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();


CREATE INDEX idx_package_items_package_id ON package_items(package_id);
CREATE INDEX idx_package_items_variant_id ON package_items(variant_id);

-- +migrate Down
DROP TABLE IF EXISTS package_items;
DROP TABLE IF EXISTS packages;


DROP TRIGGER IF EXISTS trg_package_items_updated_at ON package_items;
DROP TRIGGER IF EXISTS trg_packages_updated_at ON packages;
DROP FUNCTION IF EXISTS update_timestamp;
