# MVP 實作狀態（對照 doc/mvp.md）

本文件對照 `doc/mvp.md` 的 PRD，說明目前專案**已完成**與**尚未完成**的項目，並簡述各功能行為。

---

## 一、Bounded Context 與分層

| 項目 | 狀態 | 說明 |
|------|------|------|
| Orderbook Context | ✅ 已完成 | 訂單生命週期、驗證、建立、狀態更新、查詢 |
| Trade Context | ✅ 已完成 | 鏈上成交後的 Trade 狀態與查詢、DisplayStatus |
| Blockchain Sync Context | ✅ 已完成 | Indexer 監聽事件、更新 Order/Trade、冪等與 checkpoint |
| API Layer 僅作入口 | ✅ 已完成 | 輸入驗證、呼叫 Application Service、統一錯誤格式 |

---

## 二、Orderbook Context — 已完成項目

### 2.1 核心職責

- **驗證**：EIP-712 簽名驗證（`internal/orderbook/eip712.go`），與合約 `_hashOrder` + `ECDSA.recover` 一致。
- **建立**：`CreateOrderService` 接收 DTO → 驗證 → 建立 Order Aggregate → 寫入 DB。
- **狀態更新**：`MarkFilled` / `MarkCancelled` / `Expire` 由 Domain 提供，Repository 只做持久化（含 Indexer 觸發的更新）。
- **查詢**：`OrderQueryService` 提供依 hash 單筆查詢與依條件列表查詢。

### 2.2 Aggregate Root：Order

- **檔案**：`internal/orderbook/order.go`
- **屬性**：orderHash, maker, makerIsSellETH, amountIn, minAmountOut, deltaBps, salt, deadline, signature, status, createdAt, filledAt, cancelledAt, expiredAt。
- **狀態**：Active → Filled | Cancelled | Expired；僅允許上述轉移，Expired 為時間驅動。

### 2.3 狀態模型

- 狀態常數與轉移方法已實作：`MarkFilled()`、`MarkCancelled()`、`Expire(now time.Time)`，違規時回傳 Domain Error。

### 2.4 Domain Rules

- **建立訂單**：  
  - EIP-712 驗證成功（Verifier.RecoverOrderSigner == maker）  
  - deadline > now  
  - 10000 + deltaBps > 0  
  - orderHash 不存在（Repository.Exists）  
  違反時回傳對應 Domain Error（`internal/orderbook/errors.go`）。
- **過期處理**：由 `Order.Expire(now)` 負責轉為 Expired，Repository 僅執行 `UpdateStatus`，不包含過期邏輯。

### 2.5 Repository 介面與實作

- **介面**：`internal/orderbook/repository.go`  
  - Save, GetByHash, Exists, FindByFilter, FindActiveBefore, UpdateStatus。
- **PostgreSQL 實作**：`internal/orderbook/repository_postgres.go`，使用 sqlc 產生的 `db.Queries`，不暴露 SQL。

### 2.6 Application Service

- **CreateOrderService**（`internal/orderbook/service.go`）：  
  - 接收 `CreateOrderInput`（maker, makerIsSellETH, amountIn, minAmountOut, deltaBps, salt, deadline, signature）  
  - 組裝 Order、計算 orderHash（EIP-712 digest）、驗證簽名與規則、寫入 DB。  
  - **未實作**：PRD 提到的「發出 Domain Event: OrderCreated」— 目前僅 Persist，無事件發佈。
- **ExpireOrdersService**：  
  - `ExpireActiveOrders(ctx, now)`：查 `FindActiveBefore(now.Unix())`，對每筆呼叫 `Order.Expire(now)` 後 `UpdateStatus`。  
  - 由排程每分鐘觸發（`internal/api/api.go` 的 `runExpireOrdersScheduler`）。

---

## 三、Trade Context — 已完成項目

### 3.1 核心職責

- Trade 僅由鏈上事件建立（Indexer 處理 Filled 時建立），Backend 不主動創建。

### 3.2 Aggregate Root：Trade

- **檔案**：`internal/trade/trade.go`
- **屬性**：tradeId, maker, taker, makerIsSellETH, makerAmountIn, takerDeposit, deltaBps, startTime, endTime, status, settlementPrice, makerPayout, takerPayout, makerRefund, takerRefund, createdAt, settledAt, refundedAt。
- **狀態**：Open → Settled | Refunded，不可逆。

### 3.3 狀態模型與 Display Status

- **Onchain Status**：Open, Settled, Refunded（存 DB）。
- **API Display Status**（衍生，不存 DB）：  
  - **檔案**：`internal/trade/display_status.go`  
  - **邏輯**：  
    - status == Settled → `settled`  
    - status == Refunded → `refunded`  
    - status == Open：  
      - now < endTime → `locking`  
      - endTime ≤ now < endTime + grace → `ready`  
      - now ≥ endTime + grace → `refundable`  
  - grace 為可配置（目前 7 天），由 `TradeQueryService` 使用，不在 Controller 內計算。

### 3.4 Domain Rules

- 狀態轉移僅透過 `MarkSettled(...)`、`MarkRefunded(...)`，由 Indexer 呼叫後再 Persist。

### 3.5 Repository 介面與實作

- **介面**：`internal/trade/repository.go`  
  - Save, GetByID, FindByFilter（可依 Address、Status 篩選）, UpdateSettled, UpdateRefunded。
- **PostgreSQL 實作**：`internal/trade/repository_postgres.go`，使用 sqlc。

### 3.6 TradeQueryService

- **檔案**：`internal/trade/service.go`
- **職責**：  
  - `GetByID`：單筆 Trade + 當前時間的 DisplayStatus。  
  - `ListByFilter`：依 address / status / limit / offset 查詢，每筆附 DisplayStatus。  
  - 組裝為 API 使用的 DTO（在 `internal/trade/api/api.go` 中轉成 JSON）。

---

## 四、Blockchain Sync Context（Indexer）— 已完成項目

### 4.1 責任範圍

- **RPC 監聽**：使用 `eth_getLogs`（FilterLogs）輪詢，未使用 WebSocket 訂閱。
- **Block checkpoint**：`checkpoint` 表存 `last_processed_block`、`last_processed_tx_index`。
- **重啟續跑**：啟動時從 checkpoint 讀取，fromBlock = checkpoint - ReorgBlocks（預設 10）。
- **冪等**：`eventId = txHash.Hex() + ":" + logIndex`，寫入 `processed_events`，重複事件不重複處理。
- **Reorg**：每次輪詢從 checkpoint - N 開始重掃，可覆蓋輕微 reorg。

### 4.2 Domain Mapping

- **檔案**：`internal/indexer/process.go`  
  - **Filled**：Order.MarkFilled() → UpdateStatus；建立 Trade Aggregate → Save。  
  - **Cancelled**：Order.MarkCancelled() → UpdateStatus。  
  - **Settled**：Trade.MarkSettled(...) → UpdateSettled。  
  - **Refunded**：Trade.MarkRefunded(...) → UpdateRefunded。

### 4.3 冪等策略

- `eventId = txHash + ":" + logIndex`；先查 `processed_events`，已存在則跳過；處理成功後寫入。

### 4.4 Checkpoint

- 每輪處理完後 `UpsertCheckpoint(lastProcessedBlock, lastProcessedTxIndex)`。  
- 表為空時使用 config 的 `StartBlock`（目前為 0）。

### 4.5 啟動條件

- 當 config 設有 `ethereum.vwap_rfq_contract_addr` 且 RPC 連線成功時，API server 在 `Start()` 時以 goroutine 啟動 Indexer，shutdown 時透過 cancel 結束。

---

## 五、API Layer — 已完成項目

### 5.1 職責

- **輸入驗證**：URL/query/body 解析與基本驗證（如 signature hex、limit/offset）。  
- **呼叫 Application Service**：無業務邏輯，僅轉呼叫。  
- **統一錯誤格式**：使用 `httpwrap` 回傳 JSON `{ "error": "..." }` 與適當 HTTP 狀態碼。  
- **Rate limit**：**未實作**（PRD 有列，目前無 middleware）。

### 5.2 API 與 Service 對應

| API | Service | 檔案 |
|-----|--------|------|
| POST /v1/orders | CreateOrderService | orderbook/api/api.go |
| GET /v1/orders | OrderQueryService（ListOrders） | orderbook/api/api.go |
| GET /v1/orders/:hash | OrderQueryService（OrderByHash） | orderbook/api/api.go |
| GET /v1/trades | TradeQueryService（ListByFilter） | trade/api/api.go |
| GET /v1/trades/:id | TradeQueryService（GetByID） | trade/api/api.go |

- 上述路由在 `internal/api/route.go` 註冊；Orderbook/Trade 需 config 有 `vwap_rfq_contract_addr`（Order 相關）時才會註冊 order 路由，trade 路由則始終註冊。

---

## 六、Infrastructure Layer — 已完成項目

| 項目 | 狀態 | 說明 |
|------|------|------|
| PostgreSQL Repository | ✅ | orderbook、trade 的 Postgres 實作，經 sqlc 產生 SQL 層 |
| Blockchain RPC client | ✅ | ethclient 用於 Indexer FilterLogs 與 Orderbook EIP-712 所需鏈參數 |
| EIP-712 verifier | ✅ | internal/orderbook/eip712.go，domain separator + order struct hash + 簽名 recover |
| Scheduler | ✅ | 每分鐘執行 ExpireActiveOrders |
| Logger | ✅ | slog，用於 API 與 Indexer |
| Config | ✅ | Viper，含 postgres、ethereum（rpc_url, chain_id, vwap_rfq_contract_addr 等） |

業務邏輯與狀態轉移留在 Domain / Application，Infrastructure 僅實作儲存與 RPC。

---

## 七、技術原則對照

| 原則 | 狀態 |
|------|------|
| Aggregate 保持不變式 | ✅ 狀態僅透過 Order/Trade 的 Mark* / Expire 改變 |
| Application Service 不直接操作 DB | ✅ 透過 Repository 介面 |
| Controller 不包含業務邏輯 | ✅ API handler 只做解析與呼叫 Service |
| 顯示狀態為計算值，不存 DB | ✅ DisplayStatus 由 DisplayStatusPolicy 依 Trade + now 計算 |
| 金額以最小單位字串儲存 | ✅ 欄位為 string，與 PRD 一致 |

---

## 八、尚未完成或可再補強項目

1. **OrderCreated Domain Event**  
   - PRD：CreateOrder 後「發出 Domain Event: OrderCreated」。  
   - 現狀：僅 Persist，無事件匯流排或發佈。  
   - 若要補：可加 in-process 或 outbox 事件發佈，供後續訂閱者使用。

2. **API Rate limit**  
   - PRD：API Layer 列有 Rate limit。  
   - 現狀：未實作。  
   - 若要補：可在 chi 上加 rate limit middleware（依 IP 或 token）。

3. **Indexer 使用 WebSocket**  
   - PRD：提到「WebSocket / RPC 監聽」。  
   - 現狀：僅 RPC 輪詢（FilterLogs）。  
   - 若要補：可改用 SubscribeFilterLogs 以降低延遲與 RPC 次數。

4. **Checkpoint 表初始資料**  
   - 若從未寫入過 checkpoint，目前從 block 0 開始掃。  
   - 可選：在 config 增加 `indexer.start_block`（例如合約部署 block），減少首次同步時間。

5. **Orderbook API 啟用條件**  
   - 需同時設定 `ethereum.chain_id` 與 `ethereum.vwap_rfq_contract_addr`，POST/GET orders 與排程過期才會啟用；僅設 contract 未設 chain_id 時不會建立 Verifier，order 相關功能不會上線。

---

## 九、總結

- **Orderbook / Trade / Blockchain Sync / API / Infrastructure** 依 PRD 的主流程與分層皆已實作並接好：訂單建立與過期、Trade 查詢與 DisplayStatus、Indexer 事件處理與冪等、REST API 與 config、EIP-712、Postgres、Scheduler。  
- **明確未做**：OrderCreated 事件發佈、API Rate limit；**可選強化**：Indexer WebSocket、可配置 StartBlock / 更細的 indexer 參數。  
- 部署時記得在 config 設定 `vwap_rfq_contract_addr`（與必要時 `chain_id`），並確保 DB 已跑過 migrations（含 orders、trades、processed_events、checkpoint）。
