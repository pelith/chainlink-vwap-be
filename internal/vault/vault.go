package vault

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Vault represents a vault contract client.
// Add methods as needed when implementing (e.g. balance, deposit).
type Vault interface{}

// Client is a concrete vault client (implements Vault).
type Client struct {
	Addr   common.Address
	Client *ethclient.Client
}

// NewClient creates a vault client for the given address and eth client.
func NewClient(addr common.Address, client *ethclient.Client, _ interface{}) (Vault, error) {
	return &Client{Addr: addr, Client: client}, nil
}
