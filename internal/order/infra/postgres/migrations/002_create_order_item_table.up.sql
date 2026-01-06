CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,

    product_id UUID NOT NULL,
    name TEXT NOT NULL,

    unit_amount BIGINT NOT NULL CHECK (unit_amount >= 0),
    quantity INT NOT NULL CHECK (quantity > 0),
    line_total_amount BIGINT NOT NULL CHECK (line_total_amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id
    ON order_items(order_id);