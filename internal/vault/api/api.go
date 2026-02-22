package api

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"

	"vwap/internal/liquidity"
	"vwap/internal/vault"
)

// VaultFactory creates a vault client for the given address.
type VaultFactory func(addr common.Address) (vault.Vault, error)

// AddRoutes registers vault-related routes on the provided router.
func AddRoutes(r chi.Router, vaultFactory VaultFactory, liquiditySvc liquidity.Service) {
	_ = vaultFactory
	_ = liquiditySvc // add routes when implementing (e.g. r.Route("/vaults", ...))
}
