package orderbook

import (
	"time"
)

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusActive    OrderStatus = "active"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusExpired   OrderStatus = "expired"
)

// Order is the aggregate root for the Orderbook context.
// It represents a Maker's offline-signed order and enforces consistency boundaries.
type Order struct {
	OrderHash     string
	Maker         string
	MakerIsSellETH bool
	AmountIn      string
	MinAmountOut  string
	DeltaBps      int32
	Salt          string
	Deadline      int64
	Signature     []byte
	Status        OrderStatus
	CreatedAt     time.Time
	FilledAt      *time.Time
	CancelledAt   *time.Time
	ExpiredAt     *time.Time
}

// MarkFilled transitions the order to Filled status.
// Returns error if the order is not Active.
func (o *Order) MarkFilled() error {
	if o.Status != OrderStatusActive {
		return ErrInvalidStateTransition
	}
	now := time.Now().UTC()
	o.Status = OrderStatusFilled
	o.FilledAt = &now
	return nil
}

// MarkCancelled transitions the order to Cancelled status.
// Returns error if the order is not Active.
func (o *Order) MarkCancelled() error {
	if o.Status != OrderStatusActive {
		return ErrInvalidStateTransition
	}
	now := time.Now().UTC()
	o.Status = OrderStatusCancelled
	o.CancelledAt = &now
	return nil
}

// Expire transitions the order to Expired status when deadline has passed.
// Returns error if the order is not Active or deadline has not passed.
func (o *Order) Expire(now time.Time) error {
	if o.Status != OrderStatusActive {
		return ErrInvalidStateTransition
	}
	deadlineTime := time.Unix(o.Deadline, 0)
	if !now.After(deadlineTime) {
		return ErrNotExpired
	}
	expiredAt := now.UTC()
	o.Status = OrderStatusExpired
	o.ExpiredAt = &expiredAt
	return nil
}
