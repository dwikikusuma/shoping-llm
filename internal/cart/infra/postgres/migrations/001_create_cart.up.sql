CREATE TABLE IF NOT EXISTS carts (
                                     id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    status      TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- Ensure single ACTIVE cart per user (simple rule)
CREATE UNIQUE INDEX IF NOT EXISTS ux_carts_user_active
    ON carts(user_id)
    WHERE status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS ix_carts_user_id ON carts(user_id);

CREATE TABLE IF NOT EXISTS cart_items (
                                          id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id     UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    quantity    INT NOT NULL CHECK (quantity > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(cart_id, product_id)
    );

CREATE INDEX IF NOT EXISTS ix_cart_items_cart_id ON cart_items(cart_id);
CREATE INDEX IF NOT EXISTS ix_cart_items_product_id ON cart_items(product_id);
