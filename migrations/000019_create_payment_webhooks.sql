-- +migrate Up


CREATE TABLE payment_webhooks (
    id BIGSERIAL PRIMARY KEY,

    -- Gateway info
    provider VARCHAR(30) NOT NULL, -- xendit, midtrans, stripe
    event_type VARCHAR(100),
    event_id VARCHAR(150),        -- for idempotency
    external_id VARCHAR(150),     -- reference_id / payment_request_id

    -- Security
    signature_valid BOOLEAN NOT NULL DEFAULT false,

    -- Raw payload
    payload JSONB NOT NULL,

    -- Processing status
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,

    -- Optional error tracking
    process_error TEXT
);

-- Idempotency protection
CREATE UNIQUE INDEX ux_payment_webhooks_event
ON payment_webhooks (provider, event_id);

-- Fast lookup by external id
CREATE INDEX idx_payment_webhooks_external_id
ON payment_webhooks (external_id);

ALTER TABLE payments
ADD COLUMN provider_payment_id VARCHAR(150),
ADD COLUMN paid_at TIMESTAMPTZ,
ADD COLUMN failure_reason TEXT;


CREATE UNIQUE INDEX ux_payment_webhooks_event_not_null
ON payment_webhooks (provider, event_id)
WHERE event_id IS NOT NULL;


-- +migrate Down
 
ALTER TABLE payments
DROP COLUMN IF EXISTS provider_payment_id,
DROP COLUMN IF EXISTS paid_at,
DROP COLUMN IF EXISTS failure_reason;

DROP TABLE IF EXISTS payment_webhooks;
