package indexer

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"vwap/internal/orderbook"
	"vwap/internal/trade"
)

// tradeABIJSON is the minimal ABI needed to call getTrade(bytes32).
const tradeABIJSON = `[{
	"name":"getTrade","type":"function",
	"inputs":[{"name":"tradeId","type":"bytes32"}],
	"outputs":[{"name":"","type":"tuple","components":[
		{"name":"maker","type":"address"},
		{"name":"taker","type":"address"},
		{"name":"makerIsSellETH","type":"bool"},
		{"name":"makerAmountIn","type":"uint256"},
		{"name":"takerDeposit","type":"uint256"},
		{"name":"deltaBps","type":"int32"},
		{"name":"startTime","type":"uint64"},
		{"name":"endTime","type":"uint64"},
		{"name":"status","type":"uint8"}
	]}]
}]`

// onChainTrade mirrors the contract Trade struct for ABI unpacking.
type onChainTrade struct {
	Maker          common.Address
	Taker          common.Address
	MakerIsSellETH bool
	MakerAmountIn  *big.Int
	TakerDeposit   *big.Int
	DeltaBps       int32
	StartTime      uint64
	EndTime        uint64
	Status         uint8
}

// fetchOnChainTrade calls getTrade(tradeId) on the VWAPRFQSpot contract.
func (s *Indexer) fetchOnChainTrade(ctx context.Context, tradeID string) (*onChainTrade, error) {
	parsed, err := abi.JSON(strings.NewReader(tradeABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}

	tradeIDBytes := common.HexToHash(tradeID)
	calldata, err := parsed.Pack("getTrade", [32]byte(tradeIDBytes))
	if err != nil {
		return nil, fmt.Errorf("pack calldata: %w", err)
	}

	to := s.cfg.ContractAddress
	result, err := s.client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: calldata}, nil)
	if err != nil {
		return nil, fmt.Errorf("eth_call getTrade: %w", err)
	}

	out, err := parsed.Unpack("getTrade", result)
	if err != nil {
		return nil, fmt.Errorf("unpack getTrade: %w", err)
	}

	// Unpack returns the tuple as a struct via reflection; use map approach for safety.
	type tradeResult struct {
		Maker          common.Address
		Taker          common.Address
		MakerIsSellETH bool
		MakerAmountIn  *big.Int
		TakerDeposit   *big.Int
		DeltaBps       int32
		StartTime      uint64
		EndTime        uint64
		Status         uint8
	}
	raw, ok := out[0].(struct {
		Maker          common.Address `abi:"maker"`
		Taker          common.Address `abi:"taker"`
		MakerIsSellETH bool           `abi:"makerIsSellETH"`
		MakerAmountIn  *big.Int       `abi:"makerAmountIn"`
		TakerDeposit   *big.Int       `abi:"takerDeposit"`
		DeltaBps       int32          `abi:"deltaBps"`
		StartTime      uint64         `abi:"startTime"`
		EndTime        uint64         `abi:"endTime"`
		Status         uint8          `abi:"status"`
	})
	if !ok {
		return nil, fmt.Errorf("unexpected getTrade return type %T", out[0])
	}
	return &onChainTrade{
		Maker:          raw.Maker,
		Taker:          raw.Taker,
		MakerIsSellETH: raw.MakerIsSellETH,
		MakerAmountIn:  raw.MakerAmountIn,
		TakerDeposit:   raw.TakerDeposit,
		DeltaBps:       raw.DeltaBps,
		StartTime:      raw.StartTime,
		EndTime:        raw.EndTime,
		Status:         raw.Status,
	}, nil
}

// syntheticTrade builds a trade.Trade from on-chain data when no Filled event was indexed.
func syntheticTrade(tradeID string, ct *onChainTrade) *trade.Trade {
	makerAmountIn := "0"
	if ct.MakerAmountIn != nil {
		makerAmountIn = ct.MakerAmountIn.String()
	}
	takerDeposit := "0"
	if ct.TakerDeposit != nil {
		takerDeposit = ct.TakerDeposit.String()
	}
	return &trade.Trade{
		TradeID:        tradeID,
		Maker:          ct.Maker.Hex(),
		Taker:          ct.Taker.Hex(),
		MakerIsSellETH: ct.MakerIsSellETH,
		MakerAmountIn:  makerAmountIn,
		TakerDeposit:   takerDeposit,
		DeltaBps:       ct.DeltaBps,
		StartTime:      int64(floorHour(ct.StartTime)),
		EndTime:        int64(floorHour(ct.EndTime)),
		Status:         trade.TradeStatusOpen,
		CreatedAt:      time.Now().UTC(),
	}
}

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

	startTime := floorHour(binary.BigEndian.Uint64(l.Data[24:32]))
	endTime := floorHour(binary.BigEndian.Uint64(l.Data[56:64]))
	makerAmountIn := new(big.Int).SetBytes(l.Data[64:96]).String()
	takerDeposit := new(big.Int).SetBytes(l.Data[96:128]).String()
	makerIsSellETH := l.Data[159] != 0
	deltaBps := int32(binary.BigEndian.Uint32(l.Data[188:192]))

	// Orders filled directly on-chain (scripts, tooling) may not exist in the DB.
	// Skip order status update in that case — we still index the trade from event data.
	order, err := s.orderRepo.GetByHash(ctx, orderHash)
	if err != nil && !errors.Is(err, orderbook.ErrNotFound) {
		return fmt.Errorf("get order: %w", err)
	}
	if order != nil {
		if err := order.MarkFilled(); err != nil {
			return fmt.Errorf("mark filled: %w", err)
		}
		if err := s.orderRepo.UpdateStatus(ctx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}
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
		if errors.Is(err, orderbook.ErrNotFound) {
			slog.WarnContext(ctx, "cancelled event for unknown order, skipping", "orderHash", orderHash)
			return nil
		}
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
		if !errors.Is(err, trade.ErrNotFound) {
			return fmt.Errorf("get trade: %w", err)
		}
		// Trade not in DB (Filled event not indexed) — reconstruct from chain.
		slog.WarnContext(ctx, "settled event for unknown trade, fetching from chain", "tradeID", tradeID)
		ct, cerr := s.fetchOnChainTrade(ctx, tradeID)
		if cerr != nil {
			return fmt.Errorf("fetch on-chain trade for settle: %w", cerr)
		}
		tr = syntheticTrade(tradeID, ct)
		if serr := s.tradeRepo.Save(ctx, tr); serr != nil {
			return fmt.Errorf("save synthetic trade: %w", serr)
		}
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
		if !errors.Is(err, trade.ErrNotFound) {
			return fmt.Errorf("get trade: %w", err)
		}
		// Trade not in DB (Filled event not indexed) — reconstruct from chain.
		slog.WarnContext(ctx, "refunded event for unknown trade, fetching from chain", "tradeID", tradeID)
		ct, cerr := s.fetchOnChainTrade(ctx, tradeID)
		if cerr != nil {
			return fmt.Errorf("fetch on-chain trade for refund: %w", cerr)
		}
		tr = syntheticTrade(tradeID, ct)
		if serr := s.tradeRepo.Save(ctx, tr); serr != nil {
			return fmt.Errorf("save synthetic trade: %w", serr)
		}
	}
	if err := tr.MarkRefunded(makerRefund, takerRefund); err != nil {
		return fmt.Errorf("mark refunded: %w", err)
	}
	return s.tradeRepo.UpdateRefunded(ctx, tr)
}

// floorHour floors a unix timestamp (seconds) down to the nearest hour boundary.
func floorHour(t uint64) uint64 { return (t / 3600) * 3600 }
