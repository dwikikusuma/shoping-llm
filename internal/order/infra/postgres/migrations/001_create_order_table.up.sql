CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,

    status TEXT NOT NULL,
    currency TEXT NOT NULL,

    subtotal_amount BIGINT NOT NULL CHECK (subtotal_amount >= 0),
    shipping_amount BIGINT NOT NULL CHECK (shipping_amount >= 0),
    total_amount BIGINT NOT NULL CHECK (total_amount >= 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

    CHECK (status in ('PENDING','PAID','CANCELLED','FULFILLED'))
);

CREATE INDEX IF NOT EXISTS idx_orders_user_created_at ON orders(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);