-- +migrate Up
CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_user_product UNIQUE (user_id, product_id)
);

-- Automatically update `updated_at` on change
CREATE OR REPLACE FUNCTION update_cart_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_cart_updated_at
BEFORE UPDATE ON carts
FOR EACH ROW
EXECUTE PROCEDURE update_cart_updated_at();


-- +migrate Down
DROP TRIGGER IF EXISTS trigger_update_cart_updated_at ON carts;
DROP FUNCTION IF EXISTS update_cart_updated_at;
DROP TABLE IF EXISTS carts;
