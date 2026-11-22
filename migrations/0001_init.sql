CREATE TABLE customers (
  id          text PRIMARY KEY,
  email       text NOT NULL UNIQUE,
  name        text NOT NULL,
  tier        text NOT NULL DEFAULT 'standard', -- standard | gold
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE products (
  sku             text PRIMARY KEY,
  name            text NOT NULL,
  price_cents     bigint NOT NULL,
  stock_on_hand   integer NOT NULL DEFAULT 0,
  stock_reserved  integer NOT NULL DEFAULT 0,
  active          boolean NOT NULL DEFAULT true
);

CREATE TABLE orders (
  id              text PRIMARY KEY,
  customer_id     text NOT NULL REFERENCES customers(id),
  status          text NOT NULL DEFAULT 'pending', -- pending | paid | shipped | cancelled | expired
  subtotal_cents  bigint NOT NULL,
  discount_cents  bigint NOT NULL DEFAULT 0,
  total_cents     bigint NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
  order_id          text NOT NULL REFERENCES orders(id),
  sku               text NOT NULL REFERENCES products(sku),
  qty               integer NOT NULL,
  unit_price_cents  bigint NOT NULL,
  PRIMARY KEY (order_id, sku)
);
