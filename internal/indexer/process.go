package indexer

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"vwap/internal/trade"
)

// Filled: Topics[1]=maker, [2]=taker, [3]=orderHash. Data: startTime(32), endTime(32), makerAmountIn(32), takerDeposit(32), makerIsSellETH(32), deltaBps(32) = 192 bytes.
const filledDataLen = 192

func (s *Indexer) processFilled(ctx context.Context, l types.Log) error {
	if len(l.Topics) < 4 {
		return fmt.Errorf("filled: expected 4 topics")
	}
	if len(l.Data) < filledDataLen {
		return fmt.Errorf("filled: data too short")
	}
	maker := common.BytesToAddress(l.Topics[1][12:])
	taker := common.BytesToAddress(l.Topics[2][12:])
	orderHash := "0x" + common.Bytes2Hex(l.Topics[3].Bytes())

	startTime := binary.BigEndian.Uint64(l.Data[24:32])
	endTime := binary.BigEndian.Uint64(l.Data[56:64])
	makerAmountIn := new(big.Int).SetBytes(l.Data[64:96]).String()
	takerDeposit := new(big.Int).SetBytes(l.Data[96:128]).String()
	makerIsSellETH := l.Data[159] != 0
	deltaBps := int32(binary.BigEndian.Uint32(l.Data[188:192]))

	order, err := s.orderRepo.GetByHash(ctx, orderHash)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if err := order.MarkFilled(); err != nil {
		return fmt.Errorf("mark filled: %w", err)
	}
	if err := s.orderRepo.UpdateStatus(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	tr := &trade.Trade{
		TradeID:        orderHash,
		Maker:          maker.Hex(),
		Taker:          taker.Hex(),
		MakerIsSellETH: makerIsSellETH,
		MakerAmountIn:  makerAmountIn,
		TakerDeposit:   takerDeposit,
		DeltaBps:      deltaBps,
		StartTime:      int64(startTime),
		EndTime:        int64(endTime),
		Status:         trade.TradeStatusOpen,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.tradeRepo.Save(ctx, tr); err != nil {
		return fmt.Errorf("save trade: %w", err)
	}
	return nil
}

// Cancelled: Topics[1]=maker, [2]=orderHash. No data.
func (s *Indexer) processCancelled(ctx context.Context, l types.Log) error {
	if len(l.Topics) < 3 {
		return fmt.Errorf("cancelled: expected 3 topics")
	}
	orderHash := "0x" + common.Bytes2Hex(l.Topics[2].Bytes())

	order, err := s.orderRepo.GetByHash(ctx, orderHash)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if err := order.MarkCancelled(); err != nil {
		return fmt.Errorf("mark cancelled: %w", err)
	}
	return s.orderRepo.UpdateStatus(ctx, order)
}

// Settled: Topics[1]=tradeId. Data: 6 uint256 = 192 bytes.
const settledDataLen = 192

func (s *Indexer) processSettled(ctx context.Context, l types.Log) error {
	if len(l.Topics) < 2 {
		return fmt.Errorf("settled: expected 2 topics")
	}
	if len(l.Data) < settledDataLen {
		return fmt.Errorf("settled: data too short")
	}
	tradeID := "0x" + common.Bytes2Hex(l.Topics[1].Bytes())

	settlementPrice := new(big.Int).SetBytes(l.Data[0:32]).String()
	makerPayout := new(big.Int).SetBytes(l.Data[32:64]).String()
	takerPayout := new(big.Int).SetBytes(l.Data[64:96]).String()
	makerRefund := new(big.Int).SetBytes(l.Data[96:128]).String()
	takerRefund := new(big.Int).SetBytes(l.Data[128:160]).String()

	tr, err := s.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return fmt.Errorf("get trade: %w", err)
	}
	if err := tr.MarkSettled(settlementPrice, makerPayout, takerPayout, makerRefund, takerRefund); err != nil {
		return fmt.Errorf("mark settled: %w", err)
	}
	return s.tradeRepo.UpdateSettled(ctx, tr)
}

// Refunded: Topics[1]=tradeId. Data: makerRefund(32), takerRefund(32) = 64 bytes.
const refundedDataLen = 64

func (s *Indexer) processRefunded(ctx context.Context, l types.Log) error {
	if len(l.Topics) < 2 {
		return fmt.Errorf("refunded: expected 2 topics")
	}
	if len(l.Data) < refundedDataLen {
		return fmt.Errorf("refunded: data too short")
	}
	tradeID := "0x" + common.Bytes2Hex(l.Topics[1].Bytes())
	makerRefund := new(big.Int).SetBytes(l.Data[0:32]).String()
	takerRefund := new(big.Int).SetBytes(l.Data[32:64]).String()

	tr, err := s.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return fmt.Errorf("get trade: %w", err)
	}
	if err := tr.MarkRefunded(makerRefund, takerRefund); err != nil {
		return fmt.Errorf("mark refunded: %w", err)
	}
	return s.tradeRepo.UpdateRefunded(ctx, tr)
}
