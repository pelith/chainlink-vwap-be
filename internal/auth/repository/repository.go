package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"vwap/internal/auth"
)

type Repository interface {
	UpsertAuthToken(ctx context.Context, params *UpsertAuthTokenParams) error
	GetAuthToken(ctx context.Context, token string) (*auth.Token, error)
	DeleteAuthToken(ctx context.Context, token string) error
}

type UpsertAuthTokenParams struct {
	UserID    uuid.UUID
	Token     string
	CreatedAt time.Time
	ExpiredAt time.Time
}
