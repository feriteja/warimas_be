-- +migrate Up
CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    external_id VARCHAR(255) UNIQUE NOT NULL, -- Xendit external ID
    invoice_url TEXT NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    payment_method VARCHAR(100),
    channel_code VARCHAR(100) NOT NULL,
    payment_code VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Auto-update updated_at on modification
CREATE OR REPLACE FUNCTION update_payments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trigger_update_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE PROCEDURE update_payments_updated_at();


-- +migrate Down
DROP TRIGGER IF EXISTS trigger_update_payments_updated_at ON payments;
DROP FUNCTION IF EXISTS update_payments_updated_at;
DROP TABLE IF EXISTS payments;
