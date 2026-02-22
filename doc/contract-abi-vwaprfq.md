# VWAPRFQSpot 合約 ABI 與 Go 綁定

說明如何從 `contract/VWAPRFQSpot.sol` 取得 ABI、產生 Go 綁定，以及在專案中何處可呼叫合約。

---

## 一、目前專案與合約的關係

| 功能 | 是否「call 合約」 | 說明 |
|------|------------------|------|
| **Orderbook（建立訂單）** | 否 | 僅做 EIP-712 驗證與寫 DB；真正 fill 由使用者/前端對合約送 tx |
| **Indexer** | 否（只讀鏈上事件） | 用 `eth_getLogs` 輪詢 event logs，手動解析；未用合約 instance 呼叫 |
| **查詢鏈上狀態** | 可選 | 若需要「這張單是否已在鏈上 used」「某 trade 的鏈上狀態」，可 call `used()` / `getTrade()` |

因此：**目前沒有任何地方必須 call 合約**；若之後要讀鏈上狀態或改寫 Indexer 用綁定解析 event，就需要 ABI + Go 綁定。

---

## 二、用現成 ABI 產生 Go 綁定（建議先做）

專案裡已放一份**最小 ABI**（只有 events + 讀取函數），可直接用來產生綁定，無需先編譯 Solidity。

### 1. ABI 檔案位置

- **路徑**：`contract/abi/VWAPRFQSpot.json`
- **內容**：  
  - **Events**：`Filled`, `Cancelled`, `Settled`, `Refunded`（給 Indexer 或之後用綁定解析 log）  
  - **Read**：`used(address, bytes32)`, `getTrade(bytes32)`（給需要查鏈上狀態的程式用）

### 2. 產生 Go 綁定

需先安裝 [abigen](https://github.com/ethereum/go-ethereum/tree/master/cmd/abigen)（go-ethereum 內建）：

```bash
go install github.com/ethereum/go-ethereum/cmd/abigen@latest
```

再在專案根目錄執行：

```bash
make abigen-vwap
```

會產生：

- **輸出檔**：`internal/contracts/vwaprfqspot/binding.go`
- **型別**：`VWAPRFQSpot`（struct）、`VWAPRFQSpotCaller`、`VWAPRFQSpotFilterer` 等

### 3. 在程式裡「call 合約」（讀取）

綁定產生後，任何需要讀鏈上狀態的套件都可以：

- 用 **contract address + ethclient** 建立 instance
- 呼叫 **只讀方法**（不會送 tx），例如：

```go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    "vwap/internal/contracts/vwaprfqspot"
)

// 建立綁定 instance（需要合約地址與 RPC client）
addr := common.HexToAddress(cfg.VWAPRFQContractAddr)
instance, err := vwaprfqspot.NewVWAPRFQSpot(addr, ethClient)

// 查詢訂單是否已在鏈上被使用
used, err := instance.Used(&bind.CallOpts{Context: ctx}, makerAddr, orderHashBytes)

// 查詢某筆 trade 的鏈上資料
trade, err := instance.GetTrade(&bind.CallOpts{Context: ctx}, tradeIDBytes)
```

合約地址與 RPC 可沿用現有 config（`ethereum.vwap_rfq_contract_addr`、`ethereum.rpc_url`）。

---

## 三、從 Solidity 編譯出「完整 ABI」（可選）

若之後要送 **fill / cancel / settle / refund** 等寫入交易，需要合約的**完整 ABI**（含這些 function）。做法有二。

### 方式 A：用 Foundry 編譯

1. 在專案裡初始化 Foundry（若尚未有）：

   ```bash
   forge init --no-commit
   ```

2. 把 `contract/VWAPRFQSpot.sol` 與 `IVWAPOracle.sol` 放到 `src/`，並安裝 OpenZeppelin：

   ```bash
   forge install OpenZeppelin/openzeppelin-contracts --no-commit
   ```

3. 在 `foundry.toml` 設定 `remappings`，例如：

   ```toml
   remappings = [
       "@openzeppelin/contracts/=lib/openzeppelin-contracts/contracts/"
   ]
   ```

4. 編譯：

   ```bash
   forge build
   ```

5. 從編譯結果取出 ABI，覆蓋現有最小 ABI 或另存新檔：

   - ABI 在：`out/VWAPRFQSpot.sol/VWAPRFQSpot.json` 的 `"abi"` 欄位
   - 可手動複製該 JSON 陣列，貼到 `contract/abi/VWAPRFQSpot.json`（或另建 `VWAPRFQSpot.full.json`）

6. 若要改用完整 ABI 產生綁定，可改 Makefile 的 `abigen-vwap` 指向該檔，再執行：

   ```bash
   make abigen-vwap
   ```

### 方式 B：用 solc 指令列

1. 安裝 [solc](https://github.com/ethereum/solidity/releases)（或 `solc-select`），並安裝 OpenZeppelin 依賴（例如用 npm）。
2. 用 `solc` 的 `--abi`、`--base-path`、`--include-path` 編譯 `VWAPRFQSpot.sol`，讓編譯器能解析 `@openzeppelin` 與 `IVWAPOracle`。
3. 從編譯輸出取出 ABI JSON，同樣可覆蓋或另存，再讓 `abigen-vwap` 使用該檔。

---

## 四、總結

| 步驟 | 動作 |
|------|------|
| 1 | 使用現成 `contract/abi/VWAPRFQSpot.json`（events + used + getTrade） |
| 2 | 執行 `make abigen-vwap` 產生 `internal/contracts/vwaprfqspot/binding.go` |
| 3 | 在需要「讀鏈上狀態」或「解析 event」的程式裡 import `vwaprfqspot`，用合約地址 + ethclient 建立 instance 並呼叫 `Used` / `GetTrade` 或解析 log |
| 4 | （可選）若要送 fill/cancel/settle/refund，用 Foundry 或 solc 編譯出完整 ABI，替換或新增 ABI 檔後再跑一次 `make abigen-vwap` |

目前後端**沒有**必須 call 合約的地方；ABI + 綁定是預先備好，讓之後要查鏈上狀態或改用綁定解析 event 時可直接使用。
