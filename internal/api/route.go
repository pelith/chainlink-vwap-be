package api

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v3"
	"github.com/riandyrn/otelchi"

	"vwap/internal/api/middleware"
	apiconfig "vwap/internal/config/api"
	"vwap/internal/liquidity"
	liquidityapi "vwap/internal/liquidity/api"
	"vwap/internal/orderbook"
	orderbookapi "vwap/internal/orderbook/api"
	"vwap/internal/trade"
	tradeapi "vwap/internal/trade/api"
	"vwap/internal/user"
	userapi "vwap/internal/user/api"
	vaultapi "vwap/internal/vault/api"
)

// RouteDeps holds services required for routing.
type RouteDeps struct {
	UserSvc       user.Service
	LiquiditySvc  liquidity.Service
	OrderbookSvc  *orderbook.Service
	TradeSvc      *trade.Service
	VaultFactory  vaultapi.VaultFactory
}

// AddRoutes registers API routes on the provided router (central routing).
func AddRoutes(r chi.Router, cfg *apiconfig.Config, deps RouteDeps) {
	userSvc := deps.UserSvc
	liquiditySvc := deps.LiquiditySvc
	vaultFactory := deps.VaultFactory
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Log.Level),
	}))

	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(otelchi.Middleware("api", otelchi.WithChiRoutes(r), otelchi.WithRequestMethodInSpanName(true)))
	r.Use(httplog.RequestLogger(logger, nil))
	r.Use(chimiddleware.Recoverer)

	if cfg.HTTP.CORS.Enable {
		r.Use(cors.Handler(cors.Options{ //nolint:exhaustruct // AllowOriginFunc, OptionsPassthrough, Debug not need to config
			AllowedOrigins:   cfg.HTTP.CORS.AllowedOrigins,
			AllowedMethods:   cfg.HTTP.CORS.AllowedMethods,
			AllowedHeaders:   cfg.HTTP.CORS.AllowedHeaders,
			ExposedHeaders:   cfg.HTTP.CORS.ExposedHeaders,
			AllowCredentials: cfg.HTTP.CORS.AllowCredentials,
			MaxAge:           cfg.HTTP.CORS.MaxAge,
		}))
	}

	r.Route("/v1", func(r chi.Router) {
		userapi.AddRoutes(r, userSvc)
		liquidityapi.AddRoutes(r, liquiditySvc)
		vaultapi.AddRoutes(r, vaultFactory, liquiditySvc)
		if deps.OrderbookSvc != nil {
			orderbookapi.AddRoutes(r, deps.OrderbookSvc)
		}
		if deps.TradeSvc != nil {
			tradeapi.AddRoutes(r, deps.TradeSvc)
		}
	})

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
