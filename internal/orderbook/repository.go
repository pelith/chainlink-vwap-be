package orderbook

import (
	"context"
)

// Filter holds optional filters for listing orders.
type Filter struct {
	Maker  string
	Status OrderStatus
	Limit  int
	Offset int
}

// Repository abstracts order persistence.
type Repository interface {
	Save(ctx context.Context, order *Order) error
	GetByHash(ctx context.Context, orderHash string) (*Order, error)
	Exists(ctx context.Context, orderHash string) (bool, error)
	FindByFilter(ctx context.Context, filter Filter) ([]*Order, error)
	FindActiveBefore(ctx context.Context, deadline int64) ([]*Order, error)
	UpdateStatus(ctx context.Context, order *Order) error
}
