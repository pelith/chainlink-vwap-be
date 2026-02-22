CREATE TABLE IF NOT EXISTS auth_token (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_auth_token_user_id ON auth_token(user_id);
CREATE INDEX idx_auth_token_token_hash ON auth_token(token_hash);
