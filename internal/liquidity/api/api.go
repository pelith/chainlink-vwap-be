package api

import (
	"github.com/go-chi/chi/v5"

	"vwap/internal/liquidity"
)

// AddRoutes registers liquidity-related routes on the provided router.
func AddRoutes(r chi.Router, liquiditySvc liquidity.Service) {
	_ = liquiditySvc // add routes when implementing (e.g. r.Get("/liquidity/...", ...))
}
