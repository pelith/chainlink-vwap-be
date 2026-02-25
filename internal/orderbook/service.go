package orderbook

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Service provides orderbook use cases (create, query, expire).
type Service struct {
	repo     Repository
	verifier *Verifier
}

// NewService returns a new Orderbook Service.
func NewService(repo Repository, verifier *Verifier) *Service {
	return &Service{repo: repo, verifier: verifier}
}

// CreateOrderInput is the DTO for creating an order.
type CreateOrderInput struct {
	Maker          string
	MakerIsSellETH bool
	AmountIn       string
	MinAmountOut   string
	DeltaBps       int32
	Salt           string
	Deadline       int64
	Signature      []byte
}

// CreateOrder validates (EIP-712, deadline, deltaBps, duplicate), builds the aggregate and persists.
func (s *Service) CreateOrder(ctx context.Context, in CreateOrderInput) (*Order, error) {
	now := time.Now().UTC()
	if in.Deadline <= now.Unix() {
		return nil, ErrExpired
	}
	if 10000+in.DeltaBps <= 0 {
		return nil, ErrInvalidDeltaBps
	}

	order := &Order{
		Maker:          in.Maker,
		MakerIsSellETH: in.MakerIsSellETH,
		AmountIn:       in.AmountIn,
		MinAmountOut:   in.MinAmountOut,
		DeltaBps:       in.DeltaBps,
		Salt:           in.Salt,
		Deadline:       in.Deadline,
		Signature:      in.Signature,
		Status:         OrderStatusActive,
		CreatedAt:      now,
	}
	digest := s.verifier.OrderDigest(order)
	order.OrderHash = "0x" + hex.EncodeToString(digest.Bytes())

	recovered, err := s.verifier.RecoverOrderSigner(order, in.Signature)
	if err != nil {
		slog.Info("orderbook: signature recovery failed (SigToPub)", "digest", digest.Hex(), "maker", in.Maker, "err", err)
		return nil, ErrInvalidSignature
	}
	expectedMaker := common.HexToAddress(in.Maker)
	if expectedMaker != recovered {
		slog.Info("orderbook: signer mismatch", "expected_maker", expectedMaker.Hex(), "recovered_signer", recovered.Hex(), "digest", digest.Hex())
		return nil, ErrInvalidSignature
	}

	exists, err := s.repo.Exists(ctx, order.OrderHash)
	if err != nil {
		return nil, fmt.Errorf("check order exists: %w", err)
	}
	if exists {
		return nil, ErrDuplicateOrderHash
	}

	if err := s.repo.Save(ctx, order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}
	return order, nil
}

// ListOrders returns orders matching the filter.
func (s *Service) ListOrders(ctx context.Context, filter Filter) ([]*Order, error) {
	return s.repo.FindByFilter(ctx, filter)
}

// OrderByHash returns a single order by hash.
func (s *Service) OrderByHash(ctx context.Context, orderHash string) (*Order, error) {
	return s.repo.GetByHash(ctx, orderHash)
}

// CancelOrder marks an active order as cancelled. Only the maker may cancel.
// Returns ErrNotFound if order does not exist, ErrInvalidStateTransition if not active.
func (s *Service) CancelOrder(ctx context.Context, orderHash string, maker string) (*Order, error) {
	order, err := s.repo.GetByHash(ctx, orderHash)
	if err != nil {
		return nil, err
	}
	if !stringsEqualIgnoreCase(order.Maker, maker) {
		return nil, ErrUnauthorized
	}
	if err := order.MarkCancelled(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, order); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}
	return order, nil
}

func stringsEqualIgnoreCase(a, b string) bool {
	return strings.EqualFold(strings.TrimPrefix(a, "0x"), strings.TrimPrefix(b, "0x"))
}

// ExpireActiveOrders finds active orders with deadline < now, calls Order.Expire and persists.
func (s *Service) ExpireActiveOrders(ctx context.Context, now time.Time) (int, error) {
	list, err := s.repo.FindActiveBefore(ctx, now.Unix())
	if err != nil {
		return 0, fmt.Errorf("find active before: %w", err)
	}
	n := 0
	for _, o := range list {
		if err := o.Expire(now); err != nil {
			if errors.Is(err, ErrNotExpired) {
				continue
			}
			return n, err
		}
		if err := s.repo.UpdateStatus(ctx, o); err != nil {
			return n, fmt.Errorf("update order status: %w", err)
		}
		n++
	}
	return n, nil
}
