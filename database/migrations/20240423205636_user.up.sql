CREATE TABLE IF NOT EXISTS "user" (
    "id" UUID PRIMARY KEY,
    "address" VARCHAR(42) UNIQUE NOT NULL,
    "created_at" TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP NOT NULL
);
