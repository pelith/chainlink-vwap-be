package repository

// Config holds configuration for the liquidity repository.
type Config struct {
	RPCURL          string
	ContractAddress string
}

// Repository provides liquidity data (e.g. from chain or mock).
type Repository struct {
	// Add backend fields when implementing (e.g. RPC client, contract bindings).
}

// New creates a repository that uses the given config (e.g. Ethereum RPC).
func New(cfg Config) (*Repository, error) {
	_ = cfg // use when implementing real RPC/contract client

	return &Repository{}, nil
}

// NewMock returns a mock repository for testing or when UseMock is true.
func NewMock() *Repository {
	return &Repository{}
}

// Close releases resources. No-op for mock.
func (r *Repository) Close() {}
