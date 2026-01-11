-- +migrate Up

ALTER TABLE payments
ALTER COLUMN amount
TYPE BIGINT
USING amount::BIGINT;


-- +migrate Down

ALTER TABLE payments
ALTER COLUMN amount
TYPE NUMERIC;
