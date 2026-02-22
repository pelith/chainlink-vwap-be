-- name: CreateOrder :exec
INSERT INTO orders (
    order_hash, maker, maker_is_sell_eth, amount_in, min_amount_out,
    delta_bps, salt, deadline, signature, status, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: GetOrderByHash :one
SELECT * FROM orders WHERE order_hash = $1;

-- name: OrderExists :one
SELECT EXISTS(SELECT 1 FROM orders WHERE order_hash = $1);

-- name: FindOrdersByMaker :many
SELECT * FROM orders WHERE maker = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: FindOrdersByStatus :many
SELECT * FROM orders WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: FindOrdersAll :many
SELECT * FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: FindActiveOrdersBeforeDeadline :many
SELECT * FROM orders
WHERE status = 'active' AND deadline < $1
ORDER BY deadline ASC;

-- name: UpdateOrderFilled :exec
UPDATE orders SET status = 'filled', filled_at = $2 WHERE order_hash = $1;

-- name: UpdateOrderCancelled :exec
UPDATE orders SET status = 'cancelled', cancelled_at = $2 WHERE order_hash = $1;

-- name: UpdateOrderExpired :exec
UPDATE orders SET status = 'expired', expired_at = $2 WHERE order_hash = $1;
