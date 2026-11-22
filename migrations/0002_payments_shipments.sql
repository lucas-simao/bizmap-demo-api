CREATE TABLE payments (
  id            bigserial PRIMARY KEY,
  order_id      text NOT NULL REFERENCES orders(id),
  provider_ref  text NOT NULL,
  amount_cents  bigint NOT NULL,
  paid_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE shipments (
  id             bigserial PRIMARY KEY,
  order_id       text NOT NULL REFERENCES orders(id),
  carrier        text NOT NULL,
  tracking_code  text NOT NULL,
  shipped_at     timestamptz NOT NULL DEFAULT now()
);
