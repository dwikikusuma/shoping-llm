-- name: CreateOrder :one
INSERT INTO orders (
    id,
    user_id,
    status,
    currency,
    subtotal_amount,
    shipping_amount,
    total_amount
) VALUES (
     $1, $2, $3, $4,
$5, $6, $7
 ) RETURNING *;

-- name: AddOrderItem :one
INSERT INTO order_items (
    id,
    order_id,
    product_id,
    name,
    unit_amount,
    quantity,
    line_total_amount
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
         )
RETURNING *;

-- name: GetOrderById :one
SELECT * FROM orders WHERE id = $1;

-- name: ListOrderItem :many
SELECT * FROM order_items WHERE order_id = $1;

-- name: ListOrderByUserId :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;