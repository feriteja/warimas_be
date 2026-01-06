-- +migrate Up
BEGIN;

 ALTER TABLE checkout_sessions
ADD COLUMN user_id_int INTEGER;

 
 ALTER TABLE checkout_sessions
DROP COLUMN user_id;

 ALTER TABLE checkout_sessions
RENAME COLUMN user_id_int TO user_id;

 ALTER TABLE checkout_sessions
ADD CONSTRAINT fk_checkout_sessions_user
FOREIGN KEY (user_id)
REFERENCES users(id)
ON DELETE SET NULL;

-- (Optional but recommended)
CREATE INDEX idx_checkout_sessions_user_id
ON checkout_sessions(user_id);

COMMIT;

-- +migrate Down
BEGIN;

 ALTER TABLE checkout_sessions
ADD COLUMN user_id_uuid UUID;

 
 ALTER TABLE checkout_sessions
DROP CONSTRAINT IF EXISTS fk_checkout_sessions_user;

DROP INDEX IF EXISTS idx_checkout_sessions_user_id;

 ALTER TABLE checkout_sessions
DROP COLUMN user_id;

 ALTER TABLE checkout_sessions
RENAME COLUMN user_id_uuid TO user_id;

COMMIT;
