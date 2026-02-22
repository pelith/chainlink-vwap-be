# StateView Contract & abigen Reference

Use when working with `contracts/`, Uniswap v4 StateView, abigen bindings, or liquidity/on-chain state. This project uses go-ethereum's **abigen** to generate Go bindings from contract ABI.

## Contract Information

| Field | Value |
|-------|--------|
| **Name** | StateView |
| **Purpose** | View-only contract for reading Uniswap v4 pool state |
| **Repository** | https://github.com/Uniswap/v4-periphery |
| **Docs** | https://docs.uniswap.org/contracts/v4/reference/periphery/lens/StateView |
| **Source** | https://github.com/Uniswap/v4-periphery/blob/main/src/lens/StateView.sol |

## Contract Layout (contracts/)

```
contracts/
  stateview/
    README.md       # Contract notes
    StateView.abi   # ABI (minimal)
    StateView.json  # ABI for abigen (use this)
```

Generated output: `internal/liquidity/repository/contracts/stateview.go` (package `contracts`, type `StateView`).

## Generate Go Bindings

**Preferred:** use the Makefile target:

```bash
make abigen
```

**Manual:**

```bash
abigen \
  --abi contracts/stateview/StateView.json \
  --pkg contracts \
  --type StateView \
  --out internal/liquidity/repository/contracts/stateview.go
```

After changing `contracts/stateview/StateView.json`, run `make abigen` and commit the updated `stateview.go`.

## Deployed Addresses

| Network | Address | Explorer |
|---------|---------|----------|
| Ethereum Mainnet | `0x7ffe42c4a5deea5b0fec41c94c136cf115597227` | https://etherscan.io/address/0x7ffe42c4a5deea5b0fec41c94c136cf115597227 |
| Unichain (Chain 130) | `0x86e8631a016f9068c3f085faf484ee3f5fdee8f2` | — |

## Core Functions (StateView)

- **getSlot0(poolKey)**  
  Returns pool state: `sqrtPriceX96` (Q96), `tick`, `protocolFee`, `lpFee`.

- **getTickBitmap(poolKey, int16)**  
  Returns tick bitmap for a word (256 ticks per word).

- **getTickInfo(poolKey, int24)**  
  Returns tick liquidity: `liquidityGross`, `liquidityNet`, `feeGrowthOutside0X128`, `feeGrowthOutside1X128`.

The ABI in this repo includes only the functions needed for liquidity distribution; use the generated `*contracts.StateView` binding for type-safe calls.
