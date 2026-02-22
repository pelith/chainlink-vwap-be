package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"vwap/internal/auth"
	"vwap/internal/auth/repository"
)

type Service struct {
	repo repository.Repository
}

var _ auth.Service = (*Service)(nil)

func New(repo repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Login(ctx context.Context, userID uuid.UUID) (*auth.Token, error) {
	now := time.Now()

	token := &auth.Token{
		UserID:    userID,
		Token:     generateToken(auth.DefaultTokenLength),
		CreatedAt: now,
		ExpiredAt: now.Add(time.Hour * auth.DefaultExpiredDurationInHours),
	}

	params := &repository.UpsertAuthTokenParams{
		UserID:    token.UserID,
		Token:     token.Token,
		CreatedAt: token.CreatedAt,
		ExpiredAt: token.ExpiredAt,
	}

	err := s.repo.UpsertAuthToken(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("upsert auth token: %w", err)
	}

	return token, nil
}

func (s *Service) ValidateToken(ctx context.Context, token string) (*auth.Token, error) {
	t, err := s.repo.GetAuthToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, auth.ErrTokenNotFound
		}

		return nil, fmt.Errorf("get auth token: %w", err)
	}

	if !t.IsValid() {
		return nil, auth.ErrTokenExpired
	}

	return t, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	err := s.repo.DeleteAuthToken(ctx, token)
	if err != nil {
		return fmt.Errorf("delete auth token: %w", err)
	}

	return nil
}

func generateToken(length int) string {
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(b)
}
