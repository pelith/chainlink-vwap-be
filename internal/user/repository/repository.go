package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"vwap/internal/db"
	"vwap/internal/user"
)

type Repository struct {
	q *db.Queries
}

func New(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) GetUser(ctx context.Context, id uuid.UUID) (user.User, error) {
	u, err := r.q.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}

		return user.User{}, fmt.Errorf("get user: %w", err)
	}

	return toDomain(u), nil
}

func toDomain(u db.User) user.User {
	return user.User{
		ID:        u.ID,
		Address:   u.Address,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
