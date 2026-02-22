package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"vwap/internal/auth"
	"vwap/internal/db"
)

type repo struct {
	q *db.Queries
}

// New returns a Repository implemented with *db.Queries.
//
//nolint:ireturn // returns repository interface
func New(q *db.Queries) Repository {
	return &repo{q: q}
}

func (r *repo) UpsertAuthToken(ctx context.Context, params *UpsertAuthTokenParams) error {
	err := r.q.DeleteAuthTokensByUserID(ctx, params.UserID)
	if err != nil {
		return fmt.Errorf("delete existing auth tokens: %w", err)
	}

	err = r.q.CreateAuthToken(ctx, db.CreateAuthTokenParams{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: params.Token,
		ExpiresAt: params.ExpiredAt,
		CreatedAt: params.CreatedAt,
		UpdatedAt: params.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create auth token: %w", err)
	}

	return nil
}

func (r *repo) GetAuthToken(ctx context.Context, token string) (*auth.Token, error) {
	row, err := r.q.GetAuthTokenByHash(ctx, db.GetAuthTokenByHashParams{
		TokenHash: token,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}

		return nil, fmt.Errorf("get auth token: %w", err)
	}

	return &auth.Token{
		UserID:    row.UserID,
		Token:     row.TokenHash,
		CreatedAt: row.CreatedAt,
		ExpiredAt: row.ExpiresAt,
	}, nil
}

func (r *repo) DeleteAuthToken(ctx context.Context, token string) error {
	err := r.q.DeleteAuthTokenByHash(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("delete auth token: %w", err)
	}

	return nil
}
