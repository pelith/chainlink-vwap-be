package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"vwap/internal/user"
)

type Service struct {
	repo user.Repository
}

func New(repo user.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	u, err := s.repo.GetUser(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return u, nil
}
