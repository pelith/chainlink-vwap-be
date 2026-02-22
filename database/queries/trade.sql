-- name: CreateTrade :exec
INSERT INTO trades (
    trade_id, maker, taker, maker_is_sell_eth, maker_amount_in, taker_deposit,
    delta_bps, start_time, end_time, status, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: GetTradeById :one
SELECT * FROM trades WHERE trade_id = $1;

-- name: FindTradesByAddress :many
SELECT * FROM trades
WHERE maker = $1 OR taker = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindTradesByStatus :many
SELECT * FROM trades
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindTradesByAddressAndStatus :many
SELECT * FROM trades
WHERE (maker = $1 OR taker = $1) AND status = $2
ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: UpdateTradeSettled :exec
UPDATE trades
SET status = 'settled', settlement_price = $2, maker_payout = $3, taker_payout = $4,
    maker_refund = $5, taker_refund = $6, settled_at = $7
WHERE trade_id = $1;

-- name: UpdateTradeRefunded :exec
UPDATE trades
SET status = 'refunded', maker_refund = $2, taker_refund = $3, refunded_at = $4
WHERE trade_id = $1;
