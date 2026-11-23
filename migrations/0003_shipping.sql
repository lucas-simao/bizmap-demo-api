-- Shipping fee on the order, and the carrier's own name for the tracking id.
ALTER TABLE orders ADD COLUMN shipping_cents bigint NOT NULL DEFAULT 0;
ALTER TABLE shipments RENAME COLUMN tracking_code TO tracking_number;
