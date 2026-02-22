package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultExpiredDurationInHours = 24
	DefaultTokenLength            = 32
)

type Service interface {
	Login(ctx context.Context, userID uuid.UUID) (*Token, error)
	ValidateToken(ctx context.Context, token string) (*Token, error)
	Logout(ctx context.Context, token string) error
}

type Token struct {
	UserID    uuid.UUID
	Token     string
	CreatedAt time.Time
	ExpiredAt time.Time
}

func (t *Token) IsValid() bool {
	return t.ExpiredAt.After(time.Now())
}
