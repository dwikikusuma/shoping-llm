-- name: GetActiveCartByUserID :one
SELECT * FROM carts
WHERE user_id = $1 AND status = 'ACTIVE'
    LIMIT 1;

-- name: CreateActiveCart :one
INSERT INTO carts (user_id, status)
VALUES ($1, 'ACTIVE')
    RETURNING *;

-- name: TouchCartUpdatedAt :exec
UPDATE carts SET updated_at = now()
WHERE id = $1;

-- name: ListCartItems :many
SELECT * FROM cart_items
WHERE cart_id = $1
ORDER BY created_at ASC;

-- name: UpsertAddItemIncrement :one
INSERT INTO cart_items (cart_id, product_id, quantity)
VALUES ($1, $2, $3)
    ON CONFLICT (cart_id, product_id)
DO UPDATE SET
    quantity   = cart_items.quantity + EXCLUDED.quantity,
           updated_at = now()
           RETURNING *;

-- name: SetItemQuantity :one
UPDATE cart_items
SET quantity = $3, updated_at = now()
WHERE cart_id = $1 AND product_id = $2
    RETURNING *;

-- name: RemoveItem :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: ClearCart :exec
DELETE FROM cart_items
WHERE cart_id = $1;
