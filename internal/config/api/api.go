package api

import (
	"time"
)

type Config struct {
	HTTP struct {
		Addr         string        `mapstructure:"addr" structs:"addr"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout" structs:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout" structs:"write_timeout"`
		CORS         struct {
			Enable           bool     `mapstructure:"enable" structs:"enable"`
			AllowedOrigins   []string `mapstructure:"allowed_origins" structs:"allowed_origins"`
			AllowedMethods   []string `mapstructure:"allowed_methods" structs:"allowed_methods"`
			AllowedHeaders   []string `mapstructure:"allowed_headers" structs:"allowed_headers"`
			ExposedHeaders   []string `mapstructure:"exposed_headers" structs:"exposed_headers"`
			MaxAge           int      `mapstructure:"max_age" structs:"max_age"`
			AllowCredentials bool     `mapstructure:"allow_credentials" structs:"allow_credentials"`
		} `mapstructure:"cors" structs:"cors"`
	} `mapstructure:"http" structs:"http"`
	Log struct {
		Level string `mapstructure:"level" structs:"level"`
	} `mapstructure:"log" structs:"log"`
	PostgreSQL PostgreSQL `mapstructure:"postgresql" structs:"postgresql"`
	Redis      Redis      `mapstructure:"redis" structs:"redis"`
	Ethereum   Ethereum   `mapstructure:"ethereum" structs:"ethereum"`
}

type PostgreSQL struct {
	Database string `mapstructure:"database" structs:"database"`
	Host     string `mapstructure:"host" structs:"host"`
	Port     string `mapstructure:"port" structs:"port"`
	User     string `mapstructure:"user" structs:"user"`
	Password string `mapstructure:"password" structs:"password"`
}

type Redis struct {
	Host     string `mapstructure:"host" structs:"host"`
	Port     string `mapstructure:"port" structs:"port"`
	Password string `mapstructure:"password" structs:"password"`
	DB       int    `mapstructure:"db" structs:"db"`
}

type Ethereum struct {
	RPCURL                  string `mapstructure:"rpc_url" structs:"rpc_url"`
	StateViewContractAddr   string `mapstructure:"stateview_contract_addr" structs:"stateview_contract_addr"`
	VWAPRFQContractAddr     string `mapstructure:"vwap_rfq_contract_addr" structs:"vwap_rfq_contract_addr"`
	ChainID                 int64  `mapstructure:"chain_id" structs:"chain_id"`
	UseMock                 bool   `mapstructure:"use_mock" structs:"use_mock"`
}
