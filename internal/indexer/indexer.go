package indexer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5"

	"vwap/internal/db"

	"vwap/internal/orderbook"
	"vwap/internal/trade"
)

// maxBlockRange is the maximum blocks per FilterLogs call (many RPCs limit to 50000).
const maxBlockRange = 50_000

// Config holds indexer configuration.
type Config struct {
	ContractAddress common.Address
	ReorgBlocks     int64  // re-scan from checkpoint - ReorgBlocks on startup
	StartBlock      uint64 // used when checkpoint table is empty (optional, 0 = from 0)
	PollInterval    time.Duration
}

// Indexer processes VWAPRFQ Filled/Cancelled/Settled/Refunded events and updates orders/trades.
type Indexer struct {
	cfg       Config
	client    *ethclient.Client
	q         *db.Queries
	orderRepo orderbook.Repository
	tradeRepo trade.Repository
}

// New creates an Indexer. Client must be non-nil and connected.
func New(cfg Config, client *ethclient.Client, q *db.Queries, orderRepo orderbook.Repository, tradeRepo trade.Repository) *Indexer {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 15 * time.Second
	}
	if cfg.ReorgBlocks <= 0 {
		cfg.ReorgBlocks = 10
	}
	return &Indexer{
		cfg:       cfg,
		client:    client,
		q:         q,
		orderRepo: orderRepo,
		tradeRepo: tradeRepo,
	}
}

// Run runs the indexer loop until ctx is cancelled.
func (s *Indexer) Run(ctx context.Context) error {
	for {
		if err := s.runOnce(ctx); err != nil {
			slog.ErrorContext(ctx, "indexer run failed", slog.Any("error", sanitizeRPCError(err)))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.cfg.PollInterval):
		}
	}
}

func (s *Indexer) runOnce(ctx context.Context) error {
	fromBlock, toBlock, err := s.nextRange(ctx)
	if err != nil {
		return err
	}
	if fromBlock > toBlock {
		return nil
	}

	logs, err := s.fetchLogsInChunks(ctx, fromBlock, toBlock)
	if err != nil {
		return fmt.Errorf("filter logs: %w", err)
	}
	if len(logs) == 0 {
		if toBlock > fromBlock {
			_ = s.saveCheckpoint(ctx, toBlock, 0)
		}
		return nil
	}

	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		if logs[i].TxIndex != logs[j].TxIndex {
			return logs[i].TxIndex < logs[j].TxIndex
		}
		return logs[i].Index < logs[j].Index
	})

	var lastBlock uint64
	var lastTxIndex uint

	for _, l := range logs {
		eventID := eventID(l.TxHash, l.Index)
		exists, err := s.q.ProcessedEventExists(ctx, eventID)
		if err != nil {
			return fmt.Errorf("processed event exists: %w", err)
		}
		if exists {
			lastBlock = l.BlockNumber
			lastTxIndex = l.TxIndex
			continue
		}
		if err := s.processLog(ctx, l); err != nil {
			if errors.Is(err, orderbook.ErrInvalidStateTransition) || errors.Is(err, trade.ErrInvalidStateTransition) {
				slog.WarnContext(ctx, "skipping event: invalid state transition",
					slog.String("event", eventID),
					slog.Any("error", err),
				)
			} else {
				return fmt.Errorf("process log %s: %w", eventID, err)
			}
		}
		if err := s.q.InsertProcessedEvent(ctx, eventID); err != nil {
			return fmt.Errorf("insert processed event: %w", err)
		}
		lastBlock = l.BlockNumber
		lastTxIndex = l.TxIndex
	}

	return s.saveCheckpoint(ctx, lastBlock, lastTxIndex)
}

func (s *Indexer) fetchLogsInChunks(ctx context.Context, fromBlock, toBlock uint64) ([]types.Log, error) {
	var all []types.Log
	for from := fromBlock; from <= toBlock; {
		to := from + maxBlockRange - 1
		if to > toBlock {
			to = toBlock
		}
		query := ethereum.FilterQuery{
			Addresses: []common.Address{s.cfg.ContractAddress},
			Topics:    [][]common.Hash{AllVWAPRFQTopics()},
			FromBlock: big.NewInt(int64(from)),
			ToBlock:   big.NewInt(int64(to)),
		}
		chunk, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			return nil, err
		}
		all = append(all, chunk...)
		from = to + 1
	}
	return all, nil
}

func eventID(txHash common.Hash, logIndex uint) string {
	return txHash.Hex() + ":" + strconv.FormatUint(uint64(logIndex), 10)
}

// sanitizeRPCError shortens errors that contain HTML (e.g. Cloudflare 522 pages).
func sanitizeRPCError(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	if strings.Contains(s, "<!DOCTYPE") || strings.Contains(s, "<html") {
		// Extract HTTP status if present (e.g. "522 :")
		prefix := ""
		if i := strings.Index(s, ": "); i > 0 && i < 10 {
			prefix = strings.TrimSpace(s[:i]) + ": "
		}
		return fmt.Errorf("%srpc returned HTML (connection/timeout error)", prefix) //nolint:perfsprint // need prefix
	}
	return err
}

func (s *Indexer) nextRange(ctx context.Context) (fromBlock, toBlock uint64, err error) {
	header, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("latest block: %w", err)
	}
	toBlock = header.Number.Uint64()

	// Default: start from the current latest block (only watch future events).
	// If StartBlock is set (non-zero), use it as a lower bound for historical indexing.
	fromBlock = toBlock
	if s.cfg.StartBlock != 0 {
		fromBlock = uint64(s.cfg.StartBlock)
	}

	cp, err := s.q.GetCheckpoint(ctx)
	if err == nil {
		start := cp.LastProcessedBlock - s.cfg.ReorgBlocks
		if start < 0 {
			start = 0
		}
		if uint64(start) > fromBlock {
			fromBlock = uint64(start)
		}
	} else if !isNoRows(err) {
		return 0, 0, fmt.Errorf("get checkpoint: %w", err)
	}

	return fromBlock, toBlock, nil
}

func isNoRows(err error) bool {
	return err != nil && errors.Is(err, pgx.ErrNoRows)
}

func (s *Indexer) saveCheckpoint(ctx context.Context, block uint64, txIndex uint) error {
	return s.q.UpsertCheckpoint(ctx, db.UpsertCheckpointParams{
		LastProcessedBlock:   int64(block),
		LastProcessedTxIndex: int(txIndex),
	})
}

func (s *Indexer) processLog(ctx context.Context, l types.Log) error { //nolint:cyclop // switch on topic
	if len(l.Topics) == 0 {
		return fmt.Errorf("missing topic0")
	}
	topic0 := l.Topics[0]
	switch topic0 {
	case TopicFilled:
		return s.processFilled(ctx, l)
	case TopicCancelled:
		return s.processCancelled(ctx, l)
	case TopicSettled:
		return s.processSettled(ctx, l)
	case TopicRefunded:
		return s.processRefunded(ctx, l)
	default:
		return fmt.Errorf("unknown topic %s", topic0.Hex())
	}
}
