-- name: GetCheckpoint :one
SELECT id, last_processed_block, last_processed_tx_index
FROM checkpoint WHERE id = 'default';

-- name: UpsertCheckpoint :exec
INSERT INTO checkpoint (id, last_processed_block, last_processed_tx_index)
VALUES ('default', $1, $2)
ON CONFLICT (id) DO UPDATE SET
    last_processed_block = EXCLUDED.last_processed_block,
    last_processed_tx_index = EXCLUDED.last_processed_tx_index;
