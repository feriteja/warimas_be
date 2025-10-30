-- +migrate Up
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total NUMERIC(10,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Auto-update updated_at for orders
CREATE OR REPLACE FUNCTION update_order_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_update_order_updated_at'
    ) THEN
        CREATE TRIGGER trigger_update_order_updated_at
        BEFORE UPDATE ON orders
        FOR EACH ROW
        EXECUTE PROCEDURE update_order_updated_at();
    END IF;
END $$;

-- Auto-update updated_at for order_items
CREATE OR REPLACE FUNCTION update_order_item_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_update_order_item_updated_at'
    ) THEN
        CREATE TRIGGER trigger_update_order_item_updated_at
        BEFORE UPDATE ON order_items
        FOR EACH ROW
        EXECUTE PROCEDURE update_order_item_updated_at();
    END IF;
END $$;

-- +migrate Down
DROP TRIGGER IF EXISTS trigger_update_order_item_updated_at ON order_items;
DROP FUNCTION IF EXISTS update_order_item_updated_at;

DROP TRIGGER IF EXISTS trigger_update_order_updated_at ON orders;
DROP FUNCTION IF EXISTS update_order_updated_at;

DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
