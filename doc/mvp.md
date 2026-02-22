VWAP-RFQ Spot Backend
DDD 架構 PRD（中心化模組）

版本：v1.1
範圍：Orderbook + Indexer + Trade Query + API Layer
不包含：CRE Workflow / Oracle

一、Bounded Context 劃分

本系統拆為三個 Bounded Context：

Orderbook Context

Trade Context

Blockchain Sync Context

API Layer 為 Application 層，僅作為入口，不包含業務邏輯。

二、Orderbook Context
2.1 核心職責

管理 Maker 離線簽名訂單的生命週期：

驗證

建立

狀態更新

查詢

2.2 Aggregate Root：Order
Order Aggregate 定義

Order 為一致性邊界。

屬性

orderHash (Value Object)

maker (Address VO)

makerIsSellETH (bool)

amountIn (Amount VO)

minAmountOut (Amount VO)

deltaBps (int32)

salt (uint256)

deadline (timestamp)

signature (bytes)

status (Enum)

createdAt

filledAt

cancelledAt

expiredAt

2.3 狀態模型（State Machine）

Order Status：

Active

Filled

Cancelled

Expired

狀態轉移規則：

Active → Filled

Active → Cancelled

Active → Expired

其他轉移禁止

Expired 為時間驅動轉換。

2.4 Domain Rules
1️⃣ 建立訂單

必須滿足：

EIP-712 簽名驗證成功

deadline > now

deltaBps 有效（10000 + deltaBps > 0）

orderHash 不存在

違反時拋出 Domain Error。

2️⃣ 過期處理

當 deadline < now 且 status == Active：

轉為 Expired。

此為 Domain 行為，不應直接由 Repository 更新。

2.5 Repository 介面

OrderRepository 需要支援：

Save(Order)

GetByHash(orderHash)

Exists(orderHash)

FindByFilter(...)

FindActiveBefore(timestamp)

資料庫為 PostgreSQL，但 Repository 不暴露 SQL。

2.6 Application Service
CreateOrderService

職責：

接收 DTO

建立 Order Aggregate

呼叫驗證邏輯

Persist

發出 Domain Event: OrderCreated

ExpireOrdersService

職責：

查找過期 Active 訂單

呼叫 Order.expire()

Persist

此服務由排程觸發。

三、Trade Context
3.1 核心職責

管理鏈上成交後的 Trade 狀態。

Trade 不由 Backend 主動創建。
Trade 來源為鏈上事件。

3.2 Aggregate Root：Trade
屬性

tradeId (orderHash)

maker

taker

makerIsSellETH

makerAmountIn

takerDeposit

deltaBps

startTime

endTime

status (Open / Settled / Refunded)

settlementPrice

makerPayout

takerPayout

makerRefund

takerRefund

createdAt

settledAt

refundedAt

3.3 狀態模型

Trade Onchain Status：

Open

Settled

Refunded

API Display Status（衍生，不存 DB）：

Locking

Ready

Refundable

Settled

Refunded

3.4 Domain Rules
狀態轉移

Open → Settled
Open → Refunded

不可逆。

3.5 衍生狀態計算（Domain Policy）

DisplayStatus 計算邏輯：

if status == Settled → settled
if status == Refunded → refunded
if status == Open:
    if now < endTime → locking
    if now >= endTime && now < endTime + grace → ready
    if now >= endTime + grace → refundable


grace 為可配置參數。

此邏輯為 Domain Policy，不應寫在 Controller。

3.6 Repository

TradeRepository：

Save(Trade)

GetById(tradeId)

FindByAddress(address)

FindByStatus(status)

3.7 Application Service
TradeQueryService

負責：

查詢 trades

計算 DisplayStatus

組裝 Response DTO

四、Blockchain Sync Context（Indexer）

此 Context 負責：

監聽鏈上事件

轉換為 Domain Command

更新 Aggregate

4.1 責任範圍

WebSocket / RPC 監聽

Block checkpoint

重啟續跑

冪等保證

Reorg 基本處理

4.2 Domain Mapping
Filled Event → Command

Order.markFilled()

Create Trade Aggregate

Cancelled Event → Command

Order.markCancelled()

Settled Event → Command

Trade.markSettled(...)

Refunded Event → Command

Trade.markRefunded(...)

4.3 冪等策略

使用：

eventId = txHash + logIndex

processed_events 表

唯一鍵限制

確保相同事件只處理一次。

4.4 Checkpoint

保存：

lastProcessedBlock

lastProcessedTxIndex

啟動時：

從 checkpoint - N blocks 重掃（避免輕微 reorg）

五、API Layer（Application Boundary）

API 不含業務邏輯。

職責：

輸入驗證

呼叫 Application Service

統一錯誤格式

Rate limit

API 對應 Application Service
API	Service
POST /orders	CreateOrderService
GET /orders	OrderQueryService
GET /orders/:hash	OrderQueryService
GET /trades	TradeQueryService
GET /trades/:id	TradeQueryService
六、Infrastructure Layer

包含：

PostgreSQL implementation of Repository

Blockchain RPC client

EIP-712 verifier

Scheduler

Logger

Config

不得包含：

業務邏輯

狀態轉換

七、技術原則

Aggregate 保持不變式

Application Service 不直接操作 DB

Controller 不包含業務邏輯

顯示狀態為計算值，不存資料庫

所有金額使用最小單位字串儲存

八、最終分層圖
API Layer
    ↓
Application Layer
    ↓
Domain Layer (Aggregates + Policies)
    ↓
Repository Interface
    ↓
Infrastructure (Postgres / RPC)
