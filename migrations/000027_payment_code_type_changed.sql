-- +migrate Up
ALTER TABLE payments
ALTER COLUMN payment_code TYPE VARCHAR(2048);

-- +migrate Down
ALTER TABLE payments
ALTER COLUMN payment_code TYPE VARCHAR(100)
USING LEFT(payment_code, 100);