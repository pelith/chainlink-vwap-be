# VWAP-RFQ 後端 API 文件

本文件供前端開發對接使用，描述後端提供的 REST API 規格。

---

## 一、基本資訊

| 項目 | 說明 |
|------|------|
| Base Path | `/v1` |
| Content-Type | `application/json; charset=utf-8` |
| 錯誤回應格式 | `{"error": "<錯誤訊息>"}` |

**範例 Base URL**：`https://api.example.com/v1`

---

## 二、訂單 API（Orders）

### 2.1 建立訂單

```
POST /v1/orders
```

**Request Body**

| 欄位 | 型別 | 必填 | 說明 |
|------|------|------|------|
| maker | string | ✓ | Maker 錢包地址（0x 開頭） |
| maker_is_sell_eth | boolean | ✓ | true = Maker 賣 WETH，false = Maker 賣 USDC |
| amount_in | string | ✓ | 存入數量（最小單位字串） |
| min_amount_out | string | ✓ | 最少可接受數量（最小單位字串） |
| delta_bps | integer | ✓ | 價格偏移基點，10000 + delta_bps 須 > 0 |
| salt | string | ✓ | 隨機鹽值 |
| deadline | integer | ✓ | 到期時間（Unix timestamp，秒） |
| signature | string | ✓ | EIP-712 簽名，hex 編碼（可含或不含 0x 前綴） |

**Response**

- **201 Created**：回傳 `Order` 物件
- **400 Bad Request**：invalid body、invalid signature hex、order expired、invalid deltaBps、order hash already exists
- **500 Internal Server Error**

**Order 物件**

| 欄位 | 型別 | 說明 |
|------|------|------|
| order_hash | string | 訂單唯一識別 |
| maker | string | Maker 地址 |
| maker_is_sell_eth | boolean | Maker 是否賣 WETH |
| amount_in | string | 存入數量 |
| min_amount_out | string | 最少可接受數量 |
| delta_bps | integer | 價格偏移基點 |
| salt | string | 鹽值 |
| deadline | integer | 到期時間戳 |
| status | string | `active` \| `filled` \| `cancelled` \| `expired` |
| created_at | string | ISO 8601 時間 |
| filled_at | string \| null | 成交時間（若有） |
| cancelled_at | string \| null | 取消時間（若有） |
| expired_at | string \| null | 過期時間（若有） |

---

### 2.2 訂單列表

```
GET /v1/orders
```

**Query Parameters**

| 參數 | 型別 | 必填 | 說明 |
|------|------|------|------|
| maker | string | 否 | 篩選指定 Maker 的訂單（My Quotes 用） |
| status | string | 否 | 篩選狀態：`active` \| `filled` \| `cancelled` \| `expired` |
| limit | integer | 否 | 每頁筆數，預設 20，最大 100 |
| offset | integer | 否 | 分頁偏移，預設 0 |

**Response**

- **200 OK**：回傳 `Order[]` 陣列

**使用範例**

- 市集訂單列表：`GET /v1/orders?status=active`
- My Quotes 我的報價：`GET /v1/orders?maker=0x當前address`

---

### 2.3 單筆訂單

```
GET /v1/orders/{hash}
```

**Path Parameters**

| 參數 | 說明 |
|------|------|
| hash | 訂單 hash（order_hash），可含 0x 前綴 |

**Response**

- **200 OK**：回傳 `Order` 物件
- **400 Bad Request**：invalid param: hash
- **404 Not Found**：訂單不存在
- **500 Internal Server Error**

---

### 2.4 取消訂單

```
PATCH /v1/orders/{hash}/cancel
```

**Path Parameters**

| 參數 | 說明 |
|------|------|
| hash | 訂單 hash（order_hash） |

**Request Body**

| 欄位 | 型別 | 必填 | 說明 |
|------|------|------|------|
| maker | string | ✓ | 訂單的 Maker 地址，須與訂單本人一致 |

**Response**

- **200 OK**：回傳更新後的 `Order` 物件
- **400 Bad Request**：invalid body、maker is required、invalid state transition（訂單已非 active）
- **403 Forbidden**：only maker may cancel order（maker 不符）
- **404 Not Found**：訂單不存在
- **500 Internal Server Error**

---

## 三、交易 API（Trades）

Trade 由鏈上 Fill 事件經 Indexer 寫入，前端透過 API 查詢。`display_status` 由後端依時間計算，前端不需重算。

### 3.1 交易列表

```
GET /v1/trades
```

**Query Parameters**

| 參數 | 型別 | 必填 | 說明 |
|------|------|------|------|
| address | string | 建議 | 篩選參與者（maker 或 taker 為此地址）。**不傳則回傳空陣列** |
| status | string | 否 | 篩選鏈上狀態：`open` \| `settled` \| `refunded` |
| limit | integer | 否 | 每頁筆數，預設 20，最大 100 |
| offset | integer | 否 | 分頁偏移，預設 0 |

**Response**

- **200 OK**：回傳 `Trade[]` 陣列

**使用範例**

- My Trades：`GET /v1/trades?address=0x當前address`

---

### 3.2 單筆交易

```
GET /v1/trades/{id}
```

**Path Parameters**

| 參數 | 說明 |
|------|------|
| id | 交易 ID（trade_id，等同 order_hash） |

**Response**

- **200 OK**：回傳 `Trade` 物件
- **400 Bad Request**：invalid param: id
- **404 Not Found**：交易不存在
- **500 Internal Server Error**

---

### 3.3 Trade 物件

| 欄位 | 型別 | 說明 |
|------|------|------|
| trade_id | string | 交易唯一識別（等同 order_hash） |
| maker | string | Maker 地址 |
| taker | string | Taker 地址 |
| maker_is_sell_eth | boolean | Maker 是否賣 WETH |
| maker_amount_in | string | Maker 存入數量 |
| taker_deposit | string | Taker 存入數量 |
| delta_bps | integer | 價格偏移基點 |
| start_time | integer | 鎖倉開始時間（Unix 秒） |
| end_time | integer | 結算視窗結束時間（Unix 秒） |
| status | string | 鏈上狀態：`open` \| `settled` \| `refunded` |
| display_status | string | **前端顯示用**，見下表 |
| settlement_price | string | 結算價 VWAP（已結算時） |
| maker_payout | string | Maker 結算所得（已結算時） |
| taker_payout | string | Taker 結算所得（已結算時） |
| maker_refund | string | Maker 退還金額（已退還時） |
| taker_refund | string | Taker 退還金額（已退還時） |
| created_at | string | ISO 8601 時間 |
| settled_at | string \| null | 結算時間（若有） |
| refunded_at | string \| null | 退還時間（若有） |

**display_status 值（前端依此決定 UI 與按鈕）**

| 值 | 說明 | 前端對應 |
|----|------|----------|
| `locking` | 鎖倉中，尚未到結算視窗 | 顯示「鎖倉中」 |
| `ready_to_settle` | 可結算（endTime ≤ now < endTime+grace） | 顯示「Settle」按鈕 |
| `expired_refundable` | 可退還（已過 grace 期） | 顯示「Refund」按鈕 |
| `settled` | 已結算 | 顯示結算結果 |
| `refunded` | 已退還 | 顯示退還結果 |

**前端 role 判斷**：比對 `maker`、`taker` 與當前錢包 address，決定 `Maker` 或 `Taker` 角色。`deposited_amount` / `received_amount` 依 role 取 `maker_amount_in` 或 `taker_deposit`、`maker_payout` 或 `taker_payout`。

---

## 四、錯誤回應

所有錯誤回應格式為：

```json
{
  "error": "錯誤訊息字串"
}
```

**常見 HTTP 狀態碼**

| 狀態碼 | 說明 |
|--------|------|
| 400 | 請求參數或 body 錯誤 |
| 403 | 權限不足（如取消訂單時 maker 不符） |
| 404 | 資源不存在 |
| 500 | 伺服器內部錯誤 |

---

## 五、與鏈上流程的關係

| 操作 | 觸發方式 | 說明 |
|------|----------|------|
| 建立訂單 | 後端 POST /orders | 前端組 EIP-712 簽名後送 API |
| 取消訂單 | 後端 PATCH /orders/:hash/cancel **或** 鏈上 Cancel tx | 依產品實作 |
| 填單（Fill） | 鏈上 | 使用者送 Fill tx → Indexer 建 Trade → 前端輪詢 GET /trades |
| 結算（Settle） | 鏈上 | 使用者送 Settle tx → Indexer 更新 → 前端輪詢 GET /trades/:id |
| 退還（Refund） | 鏈上 | 使用者送 Refund tx → Indexer 更新 → 前端輪詢 GET /trades/:id |

訂單過期由後端排程處理，前端不需計算，只讀 `GET /orders` 回傳的 `status`。
