-- +migrate Up
ALTER TABLE checkout_sessions ADD COLUMN payment_method VARCHAR(40);

-- +migrate Down
ALTER TABLE checkout_sessions DROP COLUMN payment_method;
