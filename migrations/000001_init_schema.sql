-- +migrate Up

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

-- Functions
CREATE FUNCTION public.update_cart_updated_at() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE FUNCTION public.update_order_item_updated_at() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE FUNCTION public.update_order_updated_at() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE FUNCTION public.update_payments_updated_at() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

---------------------------------------------------------
-- SEQUENCES
---------------------------------------------------------

CREATE SEQUENCE public.carts_id_seq START 1;
CREATE SEQUENCE public.order_items_id_seq START 1;
CREATE SEQUENCE public.orders_id_seq START 1;
CREATE SEQUENCE public.payments_id_seq START 1;
CREATE SEQUENCE public.users_id_seq START 1;

---------------------------------------------------------
-- TABLES
---------------------------------------------------------

CREATE TABLE public.users (
    id integer NOT NULL DEFAULT nextval('public.users_id_seq'),
    email text NOT NULL,
    password text NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    role text DEFAULT 'USER' NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_email_key UNIQUE (email)
);

CREATE TABLE public.carts (
    id integer NOT NULL DEFAULT nextval('public.carts_id_seq'),
    user_id integer NOT NULL,
    product_id integer NOT NULL,
    quantity integer DEFAULT 1 NOT NULL CHECK (quantity > 0),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT carts_pkey PRIMARY KEY (id),
    CONSTRAINT unique_user_product UNIQUE (user_id, product_id)
);

CREATE TABLE public.category (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    CONSTRAINT category_pkey PRIMARY KEY (id)
);

CREATE TABLE public.orders (
    id integer NOT NULL DEFAULT nextval('public.orders_id_seq'),
    user_id integer NOT NULL,
    total numeric(10,2) DEFAULT 0 NOT NULL,
    status varchar(20) DEFAULT 'PENDING' NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (id)
);

CREATE TABLE public.order_items (
    id integer NOT NULL DEFAULT nextval('public.order_items_id_seq'),
    order_id integer NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price numeric(10,2) NOT NULL CHECK (unit_price >= 0),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    variant_id uuid,
    CONSTRAINT order_items_pkey PRIMARY KEY (id)
);

CREATE TABLE public.payments (
    id integer NOT NULL DEFAULT nextval('public.payments_id_seq'),
    order_id integer NOT NULL,
    external_id varchar(255) NOT NULL,
    invoice_url text NOT NULL,
    amount numeric(12,2) NOT NULL,
    status varchar(50) DEFAULT 'PENDING' NOT NULL,
    payment_method varchar(100),
    channel_code varchar(100) NOT NULL,
    payment_code varchar(100) NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT payments_pkey PRIMARY KEY (id),
    CONSTRAINT payments_external_id_key UNIQUE (external_id)
);

CREATE TABLE public.products (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category_id uuid NOT NULL,
    seller_id uuid NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    price integer NOT NULL,
    stock integer DEFAULT 0 NOT NULL,
    description text,
    status text DEFAULT 'active' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT products_pkey PRIMARY KEY (id),
    CONSTRAINT products_slug_key UNIQUE (slug)
);

CREATE TABLE public.variants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    name text NOT NULL,
    quantity_type text NOT NULL CHECK (quantity_type = 'UNIT'),
    price numeric(12,2) NOT NULL,
    stock integer DEFAULT 0 NOT NULL,
    image text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT variants_pkey PRIMARY KEY (id)
);

CREATE TABLE public.product_variant_map (
    variant_id uuid NOT NULL,
    product_id integer NOT NULL,
    CONSTRAINT product_variant_map_pkey PRIMARY KEY (variant_id)
);

CREATE TABLE public.stock_movement (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    variant_id uuid NOT NULL,
    type text NOT NULL,
    quantity integer NOT NULL,
    note text,
    created_at timestamp without time zone DEFAULT now(),
    CONSTRAINT stock_movement_pkey PRIMARY KEY (id)
);

CREATE TABLE public.subcategories (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category_id uuid NOT NULL,
    name text NOT NULL,
    CONSTRAINT subcategories_pkey PRIMARY KEY (id)
);

-- Your custom migrations table
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version text NOT NULL,
    applied_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT schema_migrations_pkey PRIMARY KEY (version)
);

---------------------------------------------------------
-- INDEXES
---------------------------------------------------------

CREATE INDEX idx_order_items_variant_id ON public.order_items(variant_id);
CREATE INDEX idx_stock_movement_variant_id ON public.stock_movement(variant_id);

---------------------------------------------------------
-- FOREIGN KEYS
---------------------------------------------------------

ALTER TABLE public.carts
    ADD CONSTRAINT carts_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE public.orders
    ADD CONSTRAINT orders_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE public.order_items
    ADD CONSTRAINT order_items_order_id_fkey FOREIGN KEY (order_id)
    REFERENCES public.orders(id) ON DELETE CASCADE;

ALTER TABLE public.payments
    ADD CONSTRAINT payments_order_id_fkey FOREIGN KEY (order_id)
    REFERENCES public.orders(id) ON DELETE CASCADE;

ALTER TABLE public.products
    ADD CONSTRAINT fk_products_category FOREIGN KEY (category_id)
    REFERENCES public.category(id);

ALTER TABLE public.stock_movement
    ADD CONSTRAINT stock_movement_variant_id_fkey FOREIGN KEY (variant_id)
    REFERENCES public.variants(id);

ALTER TABLE public.subcategories
    ADD CONSTRAINT subcategories_category_id_fkey FOREIGN KEY (category_id)
    REFERENCES public.category(id) ON DELETE CASCADE;

---------------------------------------------------------
-- TRIGGERS
---------------------------------------------------------

CREATE TRIGGER trigger_update_cart_updated_at
BEFORE UPDATE ON public.carts
FOR EACH ROW EXECUTE FUNCTION public.update_cart_updated_at();

CREATE TRIGGER trigger_update_order_item_updated_at
BEFORE UPDATE ON public.order_items
FOR EACH ROW EXECUTE FUNCTION public.update_order_item_updated_at();

CREATE TRIGGER trigger_update_order_updated_at
BEFORE UPDATE ON public.orders
FOR EACH ROW EXECUTE FUNCTION public.update_order_updated_at();

CREATE TRIGGER trigger_update_payments_updated_at
BEFORE UPDATE ON public.payments
FOR EACH ROW EXECUTE FUNCTION public.update_payments_updated_at();










-- +migrate Down

DROP TRIGGER IF EXISTS trigger_update_payments_updated_at ON public.payments;
DROP TRIGGER IF EXISTS trigger_update_order_updated_at ON public.orders;
DROP TRIGGER IF EXISTS trigger_update_order_item_updated_at ON public.order_items;
DROP TRIGGER IF EXISTS trigger_update_cart_updated_at ON public.carts;

DROP TABLE IF EXISTS public.stock_movement CASCADE;
DROP TABLE IF EXISTS public.product_variant_map CASCADE;
DROP TABLE IF EXISTS public.variants CASCADE;
DROP TABLE IF EXISTS public.products CASCADE;
DROP TABLE IF EXISTS public.subcategories CASCADE;
DROP TABLE IF EXISTS public.category CASCADE;
DROP TABLE IF EXISTS public.order_items CASCADE;
DROP TABLE IF EXISTS public.orders CASCADE;
DROP TABLE IF EXISTS public.payments CASCADE;
DROP TABLE IF EXISTS public.carts CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;
DROP TABLE IF EXISTS public.schema_migrations CASCADE;

DROP SEQUENCE IF EXISTS public.carts_id_seq;
DROP SEQUENCE IF EXISTS public.order_items_id_seq;
DROP SEQUENCE IF EXISTS public.orders_id_seq;
DROP SEQUENCE IF EXISTS public.payments_id_seq;
DROP SEQUENCE IF EXISTS public.users_id_seq;

DROP FUNCTION IF EXISTS public.update_cart_updated_at();
DROP FUNCTION IF EXISTS public.update_order_item_updated_at();
DROP FUNCTION IF EXISTS public.update_order_updated_at();
DROP FUNCTION IF EXISTS public.update_payments_updated_at();

DROP EXTENSION IF EXISTS pgcrypto;
