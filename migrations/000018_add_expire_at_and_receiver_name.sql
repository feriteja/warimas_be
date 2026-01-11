-- +migrate Up

-- Add expire_at column to payments table
ALTER TABLE payments
ADD COLUMN expire_at TIMESTAMPTZ;

-- Add receiver_name column to addresses table
ALTER TABLE addresses
ADD COLUMN receiver_name VARCHAR(150) DEFAULT 'USER' NOT NULL ;

CREATE INDEX idx_payments_expire_at
ON payments (expire_at);

-- +migrate Down

ALTER TABLE payments
DROP COLUMN IF EXISTS expire_at;

ALTER TABLE addresses
DROP COLUMN IF EXISTS receiver_name;
