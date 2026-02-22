package trade

import (
	"context"
	"errors"
	"fmt"

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

// Save persists a new trade.
func (r *PostgresRepository) Save(ctx context.Context, t *Trade) error {
	err := r.q.CreateTrade(ctx, db.CreateTradeParams{
		TradeID:        t.TradeID,
		Maker:          t.Maker,
		Taker:          t.Taker,
		MakerIsSellEth: t.MakerIsSellETH,
		MakerAmountIn:  t.MakerAmountIn,
		TakerDeposit:   t.TakerDeposit,
		DeltaBps:       int(t.DeltaBps),
		StartTime:      t.StartTime,
		EndTime:        t.EndTime,
		Status:         string(t.Status),
		CreatedAt:      t.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create trade: %w", err)
	}
	return nil
}

// GetByID returns a trade by tradeID.
func (r *PostgresRepository) GetByID(ctx context.Context, tradeID string) (*Trade, error) {
	row, err := r.q.GetTradeById(ctx, tradeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get trade by id: %w", err)
	}
	return tradeFromDB(&row), nil
}

// FindByFilter returns trades matching the filter (address and/or status, limit, offset).
// When both Address and Status are empty, returns empty list.
func (r *PostgresRepository) FindByFilter(ctx context.Context, filter Filter) ([]*Trade, error) {
	if filter.Address == "" && filter.Status == "" {
		return []*Trade{}, nil
	}
	limit, offset := clampLimitOffset(filter.Limit, filter.Offset)

	var rows []db.Trade
	var err error
	if filter.Address != "" && filter.Status != "" {
		rows, err = r.q.FindTradesByAddressAndStatus(ctx, db.FindTradesByAddressAndStatusParams{
			Maker:  filter.Address,
			Status: string(filter.Status),
			Limit:  limit,
			Offset: offset,
		})
	} else if filter.Address != "" {
		rows, err = r.q.FindTradesByAddress(ctx, db.FindTradesByAddressParams{
			Maker:  filter.Address,
			Limit:  limit,
			Offset: offset,
		})
	} else {
		rows, err = r.q.FindTradesByStatus(ctx, db.FindTradesByStatusParams{
			Status: string(filter.Status),
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("find trades: %w", err)
	}
	out := make([]*Trade, len(rows))
	for i := range rows {
		out[i] = tradeFromDB(&rows[i])
	}
	return out, nil
}

// UpdateSettled persists Settled status and payout fields.
func (r *PostgresRepository) UpdateSettled(ctx context.Context, t *Trade) error {
	if t.SettledAt == nil {
		return fmt.Errorf("settled_at required for status settled")
	}
	return r.q.UpdateTradeSettled(ctx, db.UpdateTradeSettledParams{
		TradeID:         t.TradeID,
		SettlementPrice: pgtype.Text{String: t.SettlementPrice, Valid: t.SettlementPrice != ""},
		MakerPayout:     pgtype.Text{String: t.MakerPayout, Valid: t.MakerPayout != ""},
		TakerPayout:     pgtype.Text{String: t.TakerPayout, Valid: t.TakerPayout != ""},
		MakerRefund:     pgtype.Text{String: t.MakerRefund, Valid: t.MakerRefund != ""},
		TakerRefund:     pgtype.Text{String: t.TakerRefund, Valid: t.TakerRefund != ""},
		SettledAt:       pgtype.Timestamp{Time: *t.SettledAt, Valid: true},
	})
}

// UpdateRefunded persists Refunded status and refund fields.
func (r *PostgresRepository) UpdateRefunded(ctx context.Context, t *Trade) error {
	if t.RefundedAt == nil {
		return fmt.Errorf("refunded_at required for status refunded")
	}
	return r.q.UpdateTradeRefunded(ctx, db.UpdateTradeRefundedParams{
		TradeID:     t.TradeID,
		MakerRefund: pgtype.Text{String: t.MakerRefund, Valid: t.MakerRefund != ""},
		TakerRefund: pgtype.Text{String: t.TakerRefund, Valid: t.TakerRefund != ""},
		RefundedAt:  pgtype.Timestamp{Time: *t.RefundedAt, Valid: true},
	})
}

func tradeFromDB(row *db.Trade) *Trade {
	out := &Trade{
		TradeID:        row.TradeID,
		Maker:          row.Maker,
		Taker:          row.Taker,
		MakerIsSellETH: row.MakerIsSellEth,
		MakerAmountIn:  row.MakerAmountIn,
		TakerDeposit:   row.TakerDeposit,
		DeltaBps:       int32(row.DeltaBps),
		StartTime:      row.StartTime,
		EndTime:        row.EndTime,
		Status:         TradeStatus(row.Status),
		CreatedAt:      row.CreatedAt,
	}
	if row.SettlementPrice.Valid {
		out.SettlementPrice = row.SettlementPrice.String
	}
	if row.MakerPayout.Valid {
		out.MakerPayout = row.MakerPayout.String
	}
	if row.TakerPayout.Valid {
		out.TakerPayout = row.TakerPayout.String
	}
	if row.MakerRefund.Valid {
		out.MakerRefund = row.MakerRefund.String
	}
	if row.TakerRefund.Valid {
		out.TakerRefund = row.TakerRefund.String
	}
	if row.SettledAt.Valid {
		out.SettledAt = &row.SettledAt.Time
	}
	if row.RefundedAt.Valid {
		out.RefundedAt = &row.RefundedAt.Time
	}
	return out
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
