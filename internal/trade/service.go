package trade

import (
	"context"
	"fmt"
	"time"
)

// Service provides trade query use cases with DisplayStatus.
type Service struct {
	repo   Repository
	policy *DisplayStatusPolicy
}

// NewService returns a new Trade Service. graceSeconds is used for DisplayStatus (ready → refundable).
func NewService(repo Repository, graceSeconds int64) *Service {
	return &Service{
		repo:   repo,
		policy: &DisplayStatusPolicy{GraceSeconds: graceSeconds},
	}
}

// TradeWithDisplay is a trade plus its API display status.
type TradeWithDisplay struct {
	Trade
	DisplayStatus DisplayStatus
}

// GetByID returns a trade by ID and its display status at current time.
func (s *Service) GetByID(ctx context.Context, tradeID string) (*TradeWithDisplay, error) {
	t, err := s.repo.GetByID(ctx, tradeID)
	if err != nil {
		return nil, fmt.Errorf("get trade: %w", err)
	}
	now := time.Now().UTC().Unix()
	return &TradeWithDisplay{
		Trade:         *t,
		DisplayStatus: s.policy.Compute(t, now),
	}, nil
}

// ListByFilter returns trades matching the filter, each with DisplayStatus at current time.
func (s *Service) ListByFilter(ctx context.Context, filter Filter) ([]*TradeWithDisplay, error) {
	list, err := s.repo.FindByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find trades: %w", err)
	}
	now := time.Now().UTC().Unix()
	out := make([]*TradeWithDisplay, len(list))
	for i := range list {
		out[i] = &TradeWithDisplay{
			Trade:         *list[i],
			DisplayStatus: s.policy.Compute(list[i], now),
		}
	}
	return out, nil
}
