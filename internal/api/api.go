package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apiconfig "vwap/internal/config/api"
	"vwap/internal/db"
	liquidityrepo "vwap/internal/liquidity/repository"
	liquiditysvc "vwap/internal/liquidity/service"
	"vwap/internal/orderbook"
	"vwap/internal/trade"
	"vwap/internal/user/repository"
	"vwap/internal/user/service"
	"vwap/internal/indexer"
	"vwap/internal/vault"
	vaultapi "vwap/internal/vault/api"
)

type Server struct {
	config        *apiconfig.Config
	httpServer    *http.Server
	pool          *pgxpool.Pool
	liquidityRepo *liquidityrepo.Repository
	ethClient     *ethclient.Client
	orderbookSvc  *orderbook.Service
	indexerCancel context.CancelFunc
}

func NewServer(ctx context.Context, cfg *apiconfig.Config) (*Server, error) {
	pool, err := newPgxPool(ctx, cfg.PostgreSQL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	queries := db.New(pool)

	userSvc := service.New(repository.New(queries))

	var liquidityRepo *liquidityrepo.Repository

	if cfg.Ethereum.UseMock {
		slog.InfoContext(ctx, "using mock liquidity repository") //nolint:sloglint // startup config logging, no logger instance available at this scope

		liquidityRepo = liquidityrepo.NewMock()
	} else {
		var err error

		liquidityRepo, err = liquidityrepo.New(liquidityrepo.Config{
			RPCURL:          cfg.Ethereum.RPCURL,
			ContractAddress: cfg.Ethereum.StateViewContractAddr,
		})
		if err != nil {
			pool.Close()

			return nil, fmt.Errorf("create liquidity repository: %w", err)
		}
	}

	liquiditySvc := liquiditysvc.New(liquidityRepo)

	orderbookRepo := orderbook.NewPostgresRepository(queries)
	var orderbookSvc *orderbook.Service
	if cfg.Ethereum.VWAPRFQContractAddr != "" && cfg.Ethereum.ChainID != 0 {
		verifier := orderbook.NewVerifier(big.NewInt(cfg.Ethereum.ChainID), common.HexToAddress(cfg.Ethereum.VWAPRFQContractAddr))
		orderbookSvc = orderbook.NewService(orderbookRepo, verifier)
	}

	tradeRepo := trade.NewPostgresRepository(queries)
	const tradeDisplayGraceSeconds = 7 * 24 * 3600 // 7 days
	tradeSvc := trade.NewService(tradeRepo, tradeDisplayGraceSeconds)

	var (
		vaultFactory vaultapi.VaultFactory
		ethClient    *ethclient.Client
	)

	if !cfg.Ethereum.UseMock && cfg.Ethereum.RPCURL != "" {
		ethClient, err = ethclient.Dial(cfg.Ethereum.RPCURL)
		if err != nil {
			pool.Close()

			if liquidityRepo != nil {
				liquidityRepo.Close()
			}

			return nil, fmt.Errorf("dial ethereum: %w", err)
		}

		vaultFactory = func(addr common.Address) (vault.Vault, error) {
			return vault.NewClient(addr, ethClient, nil)
		}
	}

	r := chi.NewRouter()
	AddRoutes(r, cfg, RouteDeps{
		UserSvc:      userSvc,
		LiquiditySvc: liquiditySvc,
		OrderbookSvc: orderbookSvc,
		TradeSvc:     tradeSvc,
		VaultFactory: vaultFactory,
	})

	return &Server{
		config:        cfg,
		httpServer:    &http.Server{Addr: cfg.HTTP.Addr, ReadTimeout: cfg.HTTP.ReadTimeout, WriteTimeout: cfg.HTTP.WriteTimeout, Handler: r},
		pool:          pool,
		liquidityRepo: liquidityRepo,
		ethClient:     ethClient,
		orderbookSvc:  orderbookSvc,
	}, nil
}

func (s *Server) Start() func(context.Context) error {
	go func() {
		slog.Info("starting http server", slog.String("addr", s.httpServer.Addr))

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("start http server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	if s.orderbookSvc != nil {
		go runExpireOrdersScheduler(s.orderbookSvc)
	}

	if s.config.Ethereum.VWAPRFQContractAddr != "" && s.ethClient != nil {
		idxQueries := db.New(s.pool)
		idxOrderRepo := orderbook.NewPostgresRepository(idxQueries)
		idxTradeRepo := trade.NewPostgresRepository(idxQueries)
		idxCtx, cancel := context.WithCancel(context.Background())
		s.indexerCancel = cancel
		go runIndexer(idxCtx, indexer.Config{
			ContractAddress: common.HexToAddress(s.config.Ethereum.VWAPRFQContractAddr),
			ReorgBlocks:     10,
			StartBlock:      0,
			PollInterval:    15 * time.Second,
		}, s.ethClient, idxQueries, idxOrderRepo, idxTradeRepo)
	}

	return func(ctx context.Context) error {
		if s.indexerCancel != nil {
			s.indexerCancel()
		}
		if s.ethClient != nil {
			s.ethClient.Close()
		}

		if s.liquidityRepo != nil {
			s.liquidityRepo.Close()
		}

		if s.pool != nil {
			s.pool.Close()
		}

		return s.httpServer.Shutdown(ctx)
	}
}

func newPgxPool(ctx context.Context, pg apiconfig.PostgreSQL) (*pgxpool.Pool, error) {
	hostAndPort := net.JoinHostPort(pg.Host, pg.Port)
	connectURI := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		pg.User,
		pg.Password,
		hostAndPort,
		pg.Database,
	)

	pool, err := pgxpool.New(ctx, connectURI)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}

	return pool, nil
}

// runIndexer runs the blockchain indexer until ctx is cancelled.
func runIndexer(ctx context.Context, cfg indexer.Config, client *ethclient.Client, q *db.Queries, orderRepo orderbook.Repository, tradeRepo trade.Repository) {
	idx := indexer.New(cfg, client, q, orderRepo, tradeRepo)
	if err := idx.Run(ctx); err != nil && ctx.Err() == nil {
		slog.ErrorContext(ctx, "indexer stopped", slog.Any("error", err))
	}
}

// runExpireOrdersScheduler runs ExpireActiveOrders every minute.
func runExpireOrdersScheduler(svc *orderbook.Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()
		n, err := svc.ExpireActiveOrders(ctx, time.Now().UTC())
		if err != nil {
			slog.ErrorContext(ctx, "expire active orders failed", slog.Any("error", err))
		} else if n > 0 {
			slog.InfoContext(ctx, "expired orders", slog.Int("count", n))
		}
	}
}
