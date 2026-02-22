package trade

import (
	"time"
)

// TradeStatus represents the on-chain status of a trade.
type TradeStatus string

const (
	TradeStatusOpen     TradeStatus = "open"
	TradeStatusSettled  TradeStatus = "settled"
	TradeStatusRefunded TradeStatus = "refunded"
)

// Trade is the aggregate root for the Trade context.
// It represents a chain-settled trade and is created from blockchain events.
type Trade struct {
	TradeID         string
	Maker           string
	Taker           string
	MakerIsSellETH  bool
	MakerAmountIn   string
	TakerDeposit    string
	DeltaBps        int32
	StartTime       int64
	EndTime         int64
	Status          TradeStatus
	SettlementPrice string
	MakerPayout     string
	TakerPayout     string
	MakerRefund     string
	TakerRefund     string
	CreatedAt       time.Time
	SettledAt       *time.Time
	RefundedAt      *time.Time
}

// MarkSettled transitions the trade to Settled status with payout amounts.
func (t *Trade) MarkSettled(settlementPrice, makerPayout, takerPayout, makerRefund, takerRefund string) error {
	if t.Status != TradeStatusOpen {
		return ErrInvalidStateTransition
	}
	now := time.Now().UTC()
	t.Status = TradeStatusSettled
	t.SettlementPrice = settlementPrice
	t.MakerPayout = makerPayout
	t.TakerPayout = takerPayout
	t.MakerRefund = makerRefund
	t.TakerRefund = takerRefund
	t.SettledAt = &now
	return nil
}

// MarkRefunded transitions the trade to Refunded status.
func (t *Trade) MarkRefunded(makerRefund, takerRefund string) error {
	if t.Status != TradeStatusOpen {
		return ErrInvalidStateTransition
	}
	now := time.Now().UTC()
	t.Status = TradeStatusRefunded
	t.MakerRefund = makerRefund
	t.TakerRefund = takerRefund
	t.RefundedAt = &now
	return nil
}
