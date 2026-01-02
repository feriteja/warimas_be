-- +migrate Up
-- =========================================
-- ADDRESSES
-- =========================================
CREATE TABLE addresses (
  id UUID PRIMARY KEY,
  user_id UUID NULL,

  name VARCHAR(100) NOT NULL,
  phone VARCHAR(30) NOT NULL,

  address_line1 TEXT NOT NULL,
  address_line2 TEXT,

  city VARCHAR(100) NOT NULL,
  province VARCHAR(100) NOT NULL,
  postal_code VARCHAR(20) NOT NULL,
  country CHAR(2) NOT NULL DEFAULT 'ID',

  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_addresses_user_id
  ON addresses(user_id);

-- Only one active default address per user
CREATE UNIQUE INDEX uniq_default_address_per_user
  ON addresses(user_id)
  WHERE is_default = true AND is_active = true;


-- =========================================
-- PREVENT MUTATING ADDRESS CONTENT
-- (but allow is_active / is_default)
-- =========================================
CREATE OR REPLACE FUNCTION prevent_address_content_update()
RETURNS TRIGGER AS $$
BEGIN
  IF (
    NEW.name          <> OLD.name OR
    NEW.phone         <> OLD.phone OR
    NEW.address_line1 <> OLD.address_line1 OR
    COALESCE(NEW.address_line2, '') <> COALESCE(OLD.address_line2, '') OR
    NEW.city          <> OLD.city OR
    NEW.province      <> OLD.province OR
    NEW.postal_code   <> OLD.postal_code OR
    NEW.country       <> OLD.country
  ) THEN
    RAISE EXCEPTION
      'Address content is immutable; create a new address instead';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER address_content_no_update
BEFORE UPDATE ON addresses
FOR EACH ROW
EXECUTE FUNCTION prevent_address_content_update();


-- =========================================
-- CHECKOUT SESSIONS
-- =========================================
CREATE TABLE checkout_sessions (
  id UUID PRIMARY KEY,
  user_id UUID NULL,

  status VARCHAR(20) NOT NULL DEFAULT 'PENDING',

  subtotal BIGINT NOT NULL,
  tax BIGINT NOT NULL DEFAULT 0,
  shipping_fee BIGINT NOT NULL DEFAULT 0,
  discount BIGINT NOT NULL DEFAULT 0,
  total_price BIGINT NOT NULL,

  currency CHAR(3) NOT NULL DEFAULT 'IDR',

  address_id UUID NULL,
  expires_at TIMESTAMP NOT NULL,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

  CONSTRAINT fk_checkout_sessions_address
    FOREIGN KEY (address_id)
    REFERENCES addresses(id)
    ON DELETE RESTRICT
);

CREATE INDEX idx_checkout_sessions_user_id
  ON checkout_sessions(user_id);

CREATE INDEX idx_checkout_sessions_status
  ON checkout_sessions(status);

CREATE INDEX idx_checkout_sessions_expires_at
  ON checkout_sessions(expires_at);


-- =========================================
-- CHECKOUT SESSION ITEMS
-- =========================================
CREATE TABLE checkout_session_items (
  id UUID PRIMARY KEY,
  checkout_session_id UUID NOT NULL,

  product_id UUID NOT NULL,
  product_name TEXT NOT NULL,
  sku VARCHAR(100),

  unit_price BIGINT NOT NULL,
  quantity INT NOT NULL CHECK (quantity > 0),
  subtotal BIGINT NOT NULL,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),

  CONSTRAINT fk_checkout_session_items_session
    FOREIGN KEY (checkout_session_id)
    REFERENCES checkout_sessions(id)
    ON DELETE CASCADE
);

CREATE INDEX idx_checkout_session_items_session_id
  ON checkout_session_items(checkout_session_id);


-- =========================================
-- STATUS SAFETY
-- =========================================
ALTER TABLE checkout_sessions
  ADD CONSTRAINT chk_checkout_sessions_status
  CHECK (status IN ('PENDING', 'PAID', 'EXPIRED', 'CANCELLED'))
  NOT VALID;


-- +migrate Down
-- =========================================
DROP TRIGGER IF EXISTS address_content_no_update ON addresses;
DROP FUNCTION IF EXISTS prevent_address_content_update;

DROP TABLE IF EXISTS checkout_session_items;
DROP TABLE IF EXISTS checkout_sessions;
DROP TABLE IF EXISTS addresses;
