-- name: DeleteAuthTokensByUserID :exec
DELETE FROM auth_token WHERE user_id = $1;

-- name: CreateAuthToken :exec
INSERT INTO auth_token (id, user_id, token_hash, expires_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAuthTokenByHash :one
SELECT id, user_id, token_hash, expires_at, created_at, updated_at
FROM auth_token
WHERE token_hash = $1 AND expires_at > $2;

-- name: DeleteAuthTokenByHash :exec
DELETE FROM auth_token WHERE token_hash = $1;
