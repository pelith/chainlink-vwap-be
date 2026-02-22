-- name: GetUser :one
SELECT * FROM "user" WHERE "id" = $1;

-- name: GetUserByAddress :one
SELECT * FROM "user" WHERE "address" = $1;

-- name: InsertUser :exec
INSERT INTO "user" ("id", "address", "created_at", "updated_at")
VALUES ($1, $2, $3, $4);

-- name: ListUsers :many
SELECT "id" FROM "user";
