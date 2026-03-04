# VWAP-RFQ Backend

A decentralized Request-for-Quote (RFQ) matching backend powered by VWAP (Volume-Weighted Average Price) settlement.

Makers publish signed quotes via EIP-712, Takers fill orders on-chain, and the backend indexes contract events to track trade lifecycle. Settlement prices are computed by the Chainlink CRE DON across multiple CEX data sources and written back on-chain — anyone can then trigger settlement.

---

## Table of Contents

- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [API Endpoints](#api-endpoints)
- [Background Services](#background-services)
- [Development](#development)
- [Project Structure](#project-structure)

---

## Architecture

```
Frontend / Taker
    │
    │  EIP-712 signed order
    ▼
┌────────────────────────────────────┐
│         VWAP-RFQ Backend           │
│                                    │
│  ┌────────────┐  ┌──────────────┐  │
│  │ Orderbook  │  │    Trade     │  │
│  │  Service   │  │   Service    │  │
│  └────────────┘  └──────────────┘  │
│        │                 ▲         │
│        ▼                 │         │
│  ┌────────────┐  ┌──────────────┐  │
│  │ PostgreSQL │  │   Indexer    │  │
│  └────────────┘  └──────────────┘  │
│                          │         │
└──────────────────────────┼─────────┘
                           │ eth_getLogs (poll every 15s)
                           ▼
              ┌────────────────────────┐
              │   VWAPRFQSpot (EVM)    │
              │  Filled / Cancelled    │
              │  Settled / Refunded    │
              └────────────────────────┘
                           │
              ┌────────────▼────────────┐
              │   Chainlink CRE DON     │
              │  Decentralized VWAP     │
              │  5 CEX volume-weighted  │
              └─────────────────────────┘
```

**Full settlement flow (10 steps):**

1. Taker calls `fill()` on-chain → funds are locked in the contract
2. Indexer detects `Filled` event → creates a Trade record in the database
3. Backend cron scans trades where `endTime` has passed and settlement price is not yet available
4. Backend triggers the Chainlink CRE Workflow via HTTP POST (ECDSA-signed request)
5. Each CRE DON node independently fetches historical OHLCV data from 5 CEXes (Binance, OKX, Bybit, Coinbase, Bitget)
6. Each node computes VWAP and applies circuit breakers (≥3 venues, 30-min staleness, 15% flash-crash guard)
7. Nodes reach OCR consensus; the Forwarder writes a signed report to `ManualVWAPOracle`
8. Anyone calls `settle()` → contract reads the VWAP price and distributes funds
9. Indexer detects `Settled` event → updates Trade status in the database

---

## Tech Stack

| Category | Technology |
|----------|------------|
| **Language** | Go 1.25+ |
| **HTTP Router** | [chi v5](https://github.com/go-chi/chi) |
| **Database** | PostgreSQL 16 |
| **DB Driver** | [pgx/v5](https://github.com/jackc/pgx) with connection pooling |
| **Query Layer** | [sqlc](https://sqlc.dev/) — type-safe SQL code generation |
| **Migrations** | [golang-migrate](https://github.com/golang-migrate/migrate) |
| **Blockchain** | [go-ethereum](https://github.com/ethereum/go-ethereum) — EIP-712, ethclient, FilterLogs |
| **Configuration** | [viper](https://github.com/spf13/viper) — YAML + environment variable overlay |
| **Observability** | [OpenTelemetry](https://opentelemetry.io/) via otelchi (distributed tracing) |
| **Logging** | `log/slog` — structured JSON logs |
| **Containerization** | Docker + Docker Compose |
| **Linting** | golangci-lint v2, gci, gofumpt |
| **Git Hooks** | lefthook |
| **Testing** | gomock, goleak (goroutine leak detection) |

---

## Quick Start

### Prerequisites

- [Go 1.25+](https://golang.org/dl/)
- [Docker](https://www.docker.com/)

```bash
brew install go
```

Install dev tools (golangci-lint, gci, lefthook, etc.):

```bash
go install tool
lefthook install
```

---

### Option 1: Local Development (recommended)

**Step 1: Start the database**

```bash
docker compose up -d postgres
```

**Step 2: Copy environment variables**

```bash
cp .env.example .env
# Default values work out of the box for local development
```

**Step 3: Run database migrations**

```bash
ENV=local go run ./cmd/migration
```

**Step 4: Start the API server**

```bash
ENV=local go run ./cmd/api
```

The API listens on **http://127.0.0.1:8080**.

**Health check:**

```bash
curl http://127.0.0.1:8080/health
# 200 OK
```

---

### Option 2: Docker Compose (all-in-one)

```bash
# Build the Docker image
make docker

# Start all services: postgres + migration + api
make docker-up
```

Or use the fast build (compile locally, then package — faster iteration):

```bash
make docker-up-fast
```

---

### Option 3: Build Binary

```bash
# API server
make build app=api
./vwap_api

# Migration binary
make build app=migration
./vwap_migration
```

---

## Environment Variables

Copy `.env.example` to `.env`. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ENV` | `local` | Environment name — maps to `config/api/{ENV}.yaml` |
| `APP_CONFIG_HTTP_ADDR` | `127.0.0.1:8080` | HTTP listen address |
| `APP_CONFIG_POSTGRESQL_HOST` | `127.0.0.1` | PostgreSQL host |
| `APP_CONFIG_POSTGRESQL_PORT` | `5432` | PostgreSQL port |
| `APP_CONFIG_POSTGRESQL_DATABASE` | `vwap_local` | Database name |
| `APP_CONFIG_POSTGRESQL_USER` | `postgres` | Database user |
| `APP_CONFIG_POSTGRESQL_PASSWORD` | `postgres` | Database password |
| `APP_CONFIG_ETHEREUM_RPC_URL` | — | Ethereum RPC URL (required for Indexer) |
| `APP_CONFIG_ETHEREUM_CHAIN_ID` | `1` | Chain ID (`1`=Mainnet, `84532`=Base Sepolia) |
| `APP_CONFIG_ETHEREUM_VWAP_RFQ_CONTRACT_ADDR` | — | VWAPRFQSpot contract address (enables orderbook + indexer) |
| `APP_CONFIG_SETTLER_URL` | — | Chainlink CRE Settler service URL (enables oracle trigger) |

> **Note:** Never commit `.env` to Git. Config key mapping rule: `app_config.ethereum.rpc_url` → `APP_CONFIG_ETHEREUM_RPC_URL`.

---

## API Endpoints

**Base URL:** `http://127.0.0.1:8080`  
**Content-Type:** `application/json`

---

### Health

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Service health check |

```bash
curl http://127.0.0.1:8080/health
# 200 OK
```

---

### Users

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/users/{id}` | Get a user by UUID |

---

### Orders (Orderbook)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orders` | Create an order (EIP-712 signed) |
| `GET` | `/v1/orders` | List orders with optional filters |
| `GET` | `/v1/orders/{hash}` | Get a single order by `order_hash` |
| `PATCH` | `/v1/orders/{hash}/cancel` | Cancel an order (Maker only) |

**POST /v1/orders — Request Body**

```json
{
  "maker": "0xYourAddress",
  "maker_is_sell_eth": true,
  "amount_in": "1000000000000000000",
  "min_amount_out": "1900000000",
  "delta_bps": 0,
  "salt": "0xRandomSalt",
  "deadline": 1735689600,
  "signature": "0xEIP712Signature"
}
```

**GET /v1/orders — Query Parameters**

| Parameter | Description |
|-----------|-------------|
| `maker` | Filter by maker address (for My Quotes view) |
| `status` | `active` \| `filled` \| `cancelled` \| `expired` |
| `limit` | Page size — default `20`, max `100` |
| `offset` | Pagination offset — default `0` |

```bash
# All active orders (market view)
curl "http://127.0.0.1:8080/v1/orders?status=active"

# My quotes
curl "http://127.0.0.1:8080/v1/orders?maker=0xYourAddress"
```

**PATCH /v1/orders/{hash}/cancel — Request Body**

```json
{
  "maker": "0xYourAddress"
}
```

---

### Trades

Trades are created by the Indexer when a `Filled` event is detected on-chain. The `display_status` field is computed server-side.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/trades` | List trades (pass `address` to filter) |
| `GET` | `/v1/trades/{id}` | Get a single trade by `trade_id` |

**GET /v1/trades — Query Parameters**

| Parameter | Description |
|-----------|-------------|
| `address` | Filter by participant (maker or taker). Returns empty array if omitted |
| `status` | `open` \| `settled` \| `refunded` |
| `limit` | Page size — default `20`, max `100` |
| `offset` | Pagination offset — default `0` |

```bash
# My trades
curl "http://127.0.0.1:8080/v1/trades?address=0xYourAddress"
```

**`display_status` values**

| Value | Description | Frontend action |
|-------|-------------|-----------------|
| `locking` | Funds locked, settlement window not yet open | Show "Locking" |
| `ready_to_settle` | Settlement window open (`endTime ≤ now < endTime + grace`) | Show "Settle" button |
| `expired_refundable` | Grace period passed, eligible for refund | Show "Refund" button |
| `settled` | Trade has been settled | Show settlement result |
| `refunded` | Trade has been refunded | Show refund result |

---

### Oracle

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/oracle/trigger` | Manually trigger VWAP settlement (rate-limited: 10 min) |

> Requires `APP_CONFIG_SETTLER_URL` to be set.

```bash
# Trigger settlement for the latest pending trades
curl -X POST http://127.0.0.1:8080/oracle/trigger

# Specify a custom endTime
curl -X POST http://127.0.0.1:8080/oracle/trigger \
  -H "Content-Type: application/json" \
  -d '{"endTime": 1735689600}'
```

---

### Error Response Format

All errors return:

```json
{
  "error": "error message"
}
```

| Status Code | Description |
|-------------|-------------|
| `400` | Invalid request body or parameters |
| `403` | Forbidden (e.g. non-Maker attempting to cancel) |
| `404` | Resource not found |
| `429` | Rate limited (Oracle trigger: once per 10 minutes) |
| `500` | Internal server error |
| `502` | Upstream Settler service request failed |

---

## Background Services

The following background services start automatically with the API server:

| Service | Interval | Description |
|---------|----------|-------------|
| **Order expiry scheduler** | Every 1 minute | Marks `active` orders whose `deadline` has passed as `expired` |
| **Settle scheduler** | Every 1 hour | Triggers the Settler service to settle trades from the past 12 hours |
| **Blockchain Indexer** | Every 15 seconds | Polls `VWAPRFQSpot` for `Filled`, `Cancelled`, `Settled`, and `Refunded` events and syncs the database |

> The Indexer starts only when `APP_CONFIG_ETHEREUM_VWAP_RFQ_CONTRACT_ADDR` is set.  
> The Settle scheduler starts only when `APP_CONFIG_SETTLER_URL` is set.

---

## Development

```bash
# Format imports
make gci-format

# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Run tests (with race detector)
make test

# View test coverage
make coverage

# Generate a new migration file pair
make gen-migration-sql

# Regenerate sqlc code (after modifying SQL queries)
make sqlc

# Regenerate contract Go bindings (after modifying ABI)
make abigen-vwap
```

### Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) managed via [cocogitto](https://docs.cocogitto.io/):

```bash
brew install cocogitto
cog commit feat "add new endpoint"
```

---

## Project Structure

```
vwap/
├── cmd/
│   ├── api/              # API server entry point
│   └── migration/        # Migration runner entry point
├── config/
│   ├── api/              # API config (base.yaml + per-env overrides)
│   └── migration/        # Migration config
├── contract/
│   ├── VWAPRFQSpot.sol   # Smart contract source
│   └── abi/              # ABI JSON files
├── cre/
│   └── chainlink-vwap-contract-cre/  # Chainlink CRE Workflow (VWAP computation)
├── database/
│   ├── migrations/       # SQL migration files (YYYYMMDDHHMMSS_name.up/down.sql)
│   ├── queries/          # sqlc SQL query definitions
│   └── seeds/            # Seed data (organized by environment)
├── doc/                  # API docs and frontend integration guides
└── internal/
    ├── api/              # HTTP server, routing, middleware
    ├── config/           # Config loading and typed structs
    ├── db/               # sqlc-generated type-safe query code
    ├── httpwrap/         # HTTP response helpers
    ├── indexer/          # Blockchain event indexer
    ├── oracle/           # Chainlink Settler HTTP client
    ├── orderbook/        # Orderbook domain (EIP-712, Service, Repository)
    ├── trade/            # Trade domain (Service, Repository, display_status)
    └── user/             # User domain
```

---

## Related Docs

- [`doc/api.md`](doc/api.md) — Full REST API reference with field descriptions
- [`doc/contract-abi-vwaprfq.md`](doc/contract-abi-vwaprfq.md) — VWAPRFQSpot ABI reference
- [`doc/fe.md`](doc/fe.md) — Frontend integration guide
- [`cre/chainlink-vwap-contract-cre/ARCHITECTURE.md`](cre/chainlink-vwap-contract-cre/ARCHITECTURE.md) — Chainlink CRE VWAP computation design
