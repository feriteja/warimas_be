-- +migrate Up

ALTER TABLE order_items
ADD COLUMN quantity_type varchar(20) NOT NULL DEFAULT 'UNIT';


-- +migrate Down
ALTER TABLE order_items
DROP COLUMN IF EXISTS quantity_type;
