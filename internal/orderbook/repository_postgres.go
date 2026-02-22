package orderbook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"vwap/internal/db"
)

// PostgresRepository implements Repository using PostgreSQL via sqlc.
type PostgresRepository struct {
	q *db.Queries
}

// NewPostgresRepository returns a new PostgresRepository.
func NewPostgresRepository(q *db.Queries) *PostgresRepository {
	return &PostgresRepository{q: q}
}

// Save persists a new order (CreateOrder). For updates use UpdateStatus.
func (r *PostgresRepository) Save(ctx context.Context, order *Order) error {
	err := r.q.CreateOrder(ctx, db.CreateOrderParams{
		OrderHash:      order.OrderHash,
		Maker:          order.Maker,
		MakerIsSellEth: order.MakerIsSellETH,
		AmountIn:       order.AmountIn,
		MinAmountOut:   order.MinAmountOut,
		DeltaBps:       int(order.DeltaBps),
		Salt:           order.Salt,
		Deadline:       order.Deadline,
		Signature:      order.Signature,
		Status:         string(order.Status),
		CreatedAt:      order.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

// GetByHash returns an order by orderHash.
func (r *PostgresRepository) GetByHash(ctx context.Context, orderHash string) (*Order, error) {
	row, err := r.q.GetOrderByHash(ctx, orderHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get order by hash: %w", err)
	}
	return orderFromDB(&row), nil
}

// Exists returns whether an order with the given hash exists.
func (r *PostgresRepository) Exists(ctx context.Context, orderHash string) (bool, error) {
	exists, err := r.q.OrderExists(ctx, orderHash)
	if err != nil {
		return false, fmt.Errorf("order exists: %w", err)
	}
	return exists, nil
}

// FindByFilter returns orders matching the filter (maker and/or status, limit, offset).
func (r *PostgresRepository) FindByFilter(ctx context.Context, filter Filter) ([]*Order, error) {
	limit, offset := clampLimitOffset(filter.Limit, filter.Offset)

	var rows []db.Order
	var err error
	switch {
	case filter.Maker != "" && filter.Status != "":
		rows, err = r.q.FindOrdersByMaker(ctx, db.FindOrdersByMakerParams{
			Maker:  filter.Maker,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("find orders by maker: %w", err)
		}
		// Filter by status in memory (sqlc has no FindOrdersByMakerAndStatus).
		out := make([]*Order, 0, len(rows))
		statusStr := string(filter.Status)
		for i := range rows {
			if rows[i].Status == statusStr {
				out = append(out, orderFromDB(&rows[i]))
			}
		}
		return out, nil
	case filter.Maker != "":
		rows, err = r.q.FindOrdersByMaker(ctx, db.FindOrdersByMakerParams{
			Maker:  filter.Maker,
			Limit:  limit,
			Offset: offset,
		})
	case filter.Status != "":
		rows, err = r.q.FindOrdersByStatus(ctx, db.FindOrdersByStatusParams{
			Status: string(filter.Status),
			Limit:  limit,
			Offset: offset,
		})
	default:
		rows, err = r.q.FindOrdersAll(ctx, db.FindOrdersAllParams{
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("find orders: %w", err)
	}
	out := make([]*Order, len(rows))
	for i := range rows {
		out[i] = orderFromDB(&rows[i])
	}
	return out, nil
}

// FindActiveBefore returns active orders with deadline < the given timestamp.
func (r *PostgresRepository) FindActiveBefore(ctx context.Context, deadline int64) ([]*Order, error) {
	rows, err := r.q.FindActiveOrdersBeforeDeadline(ctx, deadline)
	if err != nil {
		return nil, fmt.Errorf("find active orders before deadline: %w", err)
	}
	out := make([]*Order, len(rows))
	for i := range rows {
		out[i] = orderFromDB(&rows[i])
	}
	return out, nil
}

// UpdateStatus persists status change (Filled, Cancelled, Expired) for the order.
func (r *PostgresRepository) UpdateStatus(ctx context.Context, order *Order) error {
	switch order.Status {
	case OrderStatusActive:
		return fmt.Errorf("cannot update status for active order")
	case OrderStatusFilled:
		if order.FilledAt == nil {
			return fmt.Errorf("filled_at required for status filled")
		}
		return r.q.UpdateOrderFilled(ctx, db.UpdateOrderFilledParams{
			OrderHash: order.OrderHash,
			FilledAt:  timeToPgTimestamp(order.FilledAt),
		})
	case OrderStatusCancelled:
		if order.CancelledAt == nil {
			return fmt.Errorf("cancelled_at required for status cancelled")
		}
		return r.q.UpdateOrderCancelled(ctx, db.UpdateOrderCancelledParams{
			OrderHash:   order.OrderHash,
			CancelledAt: timeToPgTimestamp(order.CancelledAt),
		})
	case OrderStatusExpired:
		if order.ExpiredAt == nil {
			return fmt.Errorf("expired_at required for status expired")
		}
		return r.q.UpdateOrderExpired(ctx, db.UpdateOrderExpiredParams{
			OrderHash: order.OrderHash,
			ExpiredAt: timeToPgTimestamp(order.ExpiredAt),
		})
	default:
		return fmt.Errorf("unsupported status update: %s", order.Status)
	}
}

func orderFromDB(o *db.Order) *Order {
	out := &Order{
		OrderHash:      o.OrderHash,
		Maker:          o.Maker,
		MakerIsSellETH: o.MakerIsSellEth,
		AmountIn:       o.AmountIn,
		MinAmountOut:   o.MinAmountOut,
		DeltaBps:       int32(o.DeltaBps),
		Salt:           o.Salt,
		Deadline:       o.Deadline,
		Signature:      o.Signature,
		Status:         OrderStatus(o.Status),
		CreatedAt:      o.CreatedAt,
	}
	if o.FilledAt.Valid {
		out.FilledAt = &o.FilledAt.Time
	}
	if o.CancelledAt.Valid {
		out.CancelledAt = &o.CancelledAt.Time
	}
	if o.ExpiredAt.Valid {
		out.ExpiredAt = &o.ExpiredAt.Time
	}
	return out
}

func timeToPgTimestamp(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

func clampLimitOffset(limit, offset int) (int32, int32) {
	const maxLimit = 100
	if limit <= 0 {
		limit = maxLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return int32(limit), int32(offset)
}
