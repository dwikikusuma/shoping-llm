-- name: CreateProduct :one
INSERT INTO products (name, description, currency, price_amount)
VALUES ($1, $2, $3, $4)
    RETURNING id, name, description, currency, price_amount, created_at, updated_at;

-- name: GetProduct :one
SELECT id, name, description, currency, price_amount, created_at, updated_at
FROM products
WHERE id = $1;

-- name: ListProducts :many
SELECT id, name, description, currency, price_amount, created_at, updated_at
FROM products
WHERE (sqlc.arg(query) = '' OR name ILIKE '%' || sqlc.arg(query) || '%')
  AND (sqlc.narg(cursor) IS NULL OR id < sqlc.narg(cursor))
ORDER BY id DESC
    LIMIT sqlc.arg('limit')::int; -- Add the cast here