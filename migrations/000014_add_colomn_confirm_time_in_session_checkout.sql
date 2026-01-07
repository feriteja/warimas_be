-- +migrate Up
BEGIN;

ALTER TABLE checkout_sessions
ADD COLUMN confirmed_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN payment_ref VARCHAR(255);

-- Indexes
CREATE INDEX idx_checkout_sessions_confirmed_at
ON checkout_sessions(confirmed_at);

CREATE INDEX idx_checkout_sessions_payment_ref
ON checkout_sessions(payment_ref);

-- Named partial unique index (IMPORTANT)
CREATE UNIQUE INDEX uq_checkout_sessions_payment_ref
ON checkout_sessions(payment_ref)
WHERE payment_ref IS NOT NULL;

COMMIT;


-- +migrate Down
BEGIN;

-- Drop indexes first
DROP INDEX IF EXISTS uq_checkout_sessions_payment_ref;
DROP INDEX IF EXISTS idx_checkout_sessions_payment_ref;
DROP INDEX IF EXISTS idx_checkout_sessions_confirmed_at;

-- Drop columns
ALTER TABLE checkout_sessions
DROP COLUMN payment_ref,
DROP COLUMN confirmed_at;

COMMIT;
