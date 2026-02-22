package service

import (
	"vwap/internal/liquidity"
	liquidityrepo "vwap/internal/liquidity/repository"
)

type Service struct {
	repo *liquidityrepo.Repository
}

// New builds a liquidity service from the given repository.
func New(repo *liquidityrepo.Repository) liquidity.Service {
	return &Service{repo: repo}
}
