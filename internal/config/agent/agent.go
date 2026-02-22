package agent

import (
	"vwap/internal/config/api"
)

type Config struct {
	Name     string       `mapstructure:"name" structs:"name"`
	Ethereum api.Ethereum `mapstructure:"ethereum" structs:"ethereum"`
	Agent    AgentConfig  `mapstructure:"agent" structs:"agent"`
}

type AgentConfig struct {
	// RebalanceSchedule is the cron schedule for the rebalance agent
	RebalanceSchedule string `mapstructure:"rebalance_schedule" structs:"rebalance_schedule"`

	// MaxGasPriceGwei is the maximum gas price in Gwei allowed for transactions
	MaxGasPriceGwei float64 `mapstructure:"max_gas_price_gwei" structs:"max_gas_price_gwei"`

	// SwapSlippageBps is the slippage tolerance for swaps in basis points (1 bps = 0.01%)
	SwapSlippageBps int64 `mapstructure:"swap_slippage_bps" structs:"swap_slippage_bps"`
}
