package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"vwap/internal/httpwrap"
	"vwap/internal/trade"
)

const defaultLimit = 20
const maxLimit = 100

// AddRoutes registers trade routes: GET /trades, GET /trades/:id.
func AddRoutes(r chi.Router, svc *trade.Service) {
	r.Get("/trades", httpwrap.Handler(listTrades(svc)))
	r.Get("/trades/{id}", httpwrap.Handler(getTradeByID(svc)))
}

// TradeResponse is the API response for a trade (with display_status).
type TradeResponse struct {
	TradeID        string     `json:"trade_id"`
	Maker          string     `json:"maker"`
	Taker          string     `json:"taker"`
	MakerIsSellETH bool       `json:"maker_is_sell_eth"`
	MakerAmountIn  string    `json:"maker_amount_in"`
	TakerDeposit   string     `json:"taker_deposit"`
	DeltaBps       int32     `json:"delta_bps"`
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	Status         string    `json:"status"`
	DisplayStatus  string    `json:"display_status"`
	SettlementPrice string   `json:"settlement_price,omitempty"`
	MakerPayout    string    `json:"maker_payout,omitempty"`
	TakerPayout    string    `json:"taker_payout,omitempty"`
	MakerRefund    string    `json:"maker_refund,omitempty"`
	TakerRefund    string    `json:"taker_refund,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	SettledAt      *time.Time `json:"settled_at,omitempty"`
	RefundedAt     *time.Time `json:"refunded_at,omitempty"`
}

func tradeToResponse(t *trade.TradeWithDisplay) *TradeResponse {
	return &TradeResponse{
		TradeID:         t.TradeID,
		Maker:           t.Maker,
		Taker:           t.Taker,
		MakerIsSellETH:  t.MakerIsSellETH,
		MakerAmountIn:   t.MakerAmountIn,
		TakerDeposit:    t.TakerDeposit,
		DeltaBps:        t.DeltaBps,
		StartTime:       t.StartTime,
		EndTime:         t.EndTime,
		Status:          string(t.Status),
		DisplayStatus:   string(t.DisplayStatus),
		SettlementPrice: t.SettlementPrice,
		MakerPayout:     t.MakerPayout,
		TakerPayout:     t.TakerPayout,
		MakerRefund:     t.MakerRefund,
		TakerRefund:     t.TakerRefund,
		CreatedAt:       t.CreatedAt,
		SettledAt:       t.SettledAt,
		RefundedAt:      t.RefundedAt,
	}
}

func listTrades(svc *trade.Service) httpwrap.HandlerFunc {
	return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
		filter := trade.Filter{
			Address: r.URL.Query().Get("address"),
			Status:  trade.TradeStatus(r.URL.Query().Get("status")),
			Limit:   defaultLimit,
			Offset:  0,
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				filter.Limit = n
				if filter.Limit > maxLimit {
					filter.Limit = maxLimit
				}
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				filter.Offset = n
			}
		}
		list, err := svc.ListByFilter(r.Context(), filter)
		if err != nil {
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ErrorMsg: "internal error", Err: err}
		}
		body := make([]*TradeResponse, len(list))
		for i := range list {
			body[i] = tradeToResponse(list[i])
		}
		return &httpwrap.Response{StatusCode: http.StatusOK, Body: body}, nil
	}
}

func getTradeByID(svc *trade.Service) httpwrap.HandlerFunc {
	return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
		id := chi.URLParam(r, "id")
		if id == "" {
			return nil, httpwrap.NewInvalidParamErrorResponse("id")
		}
		t, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, trade.ErrNotFound) {
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusNotFound, ErrorMsg: "not found", Err: err}
			}
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ErrorMsg: "internal error", Err: err}
		}
		return &httpwrap.Response{StatusCode: http.StatusOK, Body: tradeToResponse(t)}, nil
	}
}
