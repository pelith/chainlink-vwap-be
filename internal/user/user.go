package user

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks . Repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User is the domain entity. No external dependencies.
type User struct {
	ID        uuid.UUID
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Service defines the use cases for user. No external dependencies.
type Service interface {
	ByID(ctx context.Context, id uuid.UUID) (User, error)
}

// Repository abstracts user persistence. Implementations may depend on db.
type Repository interface {
	GetUser(ctx context.Context, id uuid.UUID) (User, error)
}
