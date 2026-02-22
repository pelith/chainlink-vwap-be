package trade

import (
	"context"
)

// Filter holds optional filters for listing trades.
type Filter struct {
	Address string
	Status  TradeStatus
	Limit   int
	Offset  int
}

// Repository abstracts trade persistence.
type Repository interface {
	Save(ctx context.Context, t *Trade) error
	GetByID(ctx context.Context, tradeID string) (*Trade, error)
	FindByFilter(ctx context.Context, filter Filter) ([]*Trade, error)
	UpdateSettled(ctx context.Context, t *Trade) error
	UpdateRefunded(ctx context.Context, t *Trade) error
}
