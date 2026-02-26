package api

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"vwap/internal/httpwrap"
	"vwap/internal/orderbook"
)

const defaultLimit = 20
const maxLimit = 100

// AddRoutes registers orderbook routes: POST/GET /orders, GET /orders/:hash, PATCH /orders/:hash/cancel.
func AddRoutes(r chi.Router, svc *orderbook.Service) {
	r.Post("/orders", httpwrap.Handler(createOrder(svc)))
	r.Get("/orders", httpwrap.Handler(listOrders(svc)))
	r.Get("/orders/{hash}", httpwrap.Handler(getOrderByHash(svc)))
	r.Patch("/orders/{hash}/cancel", httpwrap.Handler(cancelOrder(svc)))
}

// CreateOrderRequest is the JSON body for POST /orders.
type CreateOrderRequest struct {
	Maker          string `json:"maker"`
	MakerIsSellETH bool   `json:"maker_is_sell_eth"`
	AmountIn       string `json:"amount_in"`
	MinAmountOut   string `json:"min_amount_out"`
	DeltaBps       int32  `json:"delta_bps"`
	Salt           string `json:"salt"`
	Deadline       int64  `json:"deadline"`
	Signature      string `json:"signature"` // hex-encoded (with or without 0x)
}

// OrderResponse is the API response for an order.
type OrderResponse struct {
	OrderHash      string     `json:"order_hash"`
	Maker          string     `json:"maker"`
	MakerIsSellETH bool       `json:"maker_is_sell_eth"`
	AmountIn       string     `json:"amount_in"`
	MinAmountOut   string     `json:"min_amount_out"`
	DeltaBps       int32      `json:"delta_bps"`
	Salt           string     `json:"salt"`
	Deadline       int64      `json:"deadline"`
	Signature      string     `json:"signature"` // hex-encoded, for Taker to call contract.fill()
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	FilledAt       *time.Time `json:"filled_at,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
}

func orderToResponse(o *orderbook.Order) *OrderResponse {
	sigHex := ""
	if len(o.Signature) > 0 {
		sigHex = "0x" + hex.EncodeToString(o.Signature)
	}
	return &OrderResponse{
		OrderHash:      o.OrderHash,
		Maker:          o.Maker,
		MakerIsSellETH: o.MakerIsSellETH,
		AmountIn:       o.AmountIn,
		MinAmountOut:   o.MinAmountOut,
		DeltaBps:       o.DeltaBps,
		Salt:           o.Salt,
		Deadline:       o.Deadline,
		Signature:      sigHex,
		Status:         string(o.Status),
		CreatedAt:      o.CreatedAt,
		FilledAt:       o.FilledAt,
		CancelledAt:    o.CancelledAt,
		ExpiredAt:      o.ExpiredAt,
	}
}

func createOrder(svc *orderbook.Service) httpwrap.HandlerFunc {
	return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
		var req CreateOrderRequest
		if err := decodeBody(r, &req); err != nil {
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ErrorMsg: "invalid body", Err: err}
		}
		sig, err := hex.DecodeString(trimHexPrefix(req.Signature))
		if err != nil {
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ErrorMsg: "invalid signature hex", Err: err}
		}
		in := orderbook.CreateOrderInput{
			Maker:          req.Maker,
			MakerIsSellETH: req.MakerIsSellETH,
			AmountIn:       req.AmountIn,
			MinAmountOut:   req.MinAmountOut,
			DeltaBps:       req.DeltaBps,
			Salt:           req.Salt,
			Deadline:       req.Deadline,
			Signature:      sig,
		}
		order, err := svc.CreateOrder(r.Context(), in)
		if err != nil {
			switch {
			case errors.Is(err, orderbook.ErrExpired), errors.Is(err, orderbook.ErrInvalidDeltaBps), errors.Is(err, orderbook.ErrInvalidSignature), errors.Is(err, orderbook.ErrDuplicateOrderHash):
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ErrorMsg: err.Error(), Err: err}
			default:
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ErrorMsg: "internal error", Err: err}
			}
		}
		return &httpwrap.Response{StatusCode: http.StatusCreated, Body: orderToResponse(order)}, nil
	}
}

func listOrders(svc *orderbook.Service) httpwrap.HandlerFunc {
	return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
		filter := orderbook.Filter{
			Maker:  r.URL.Query().Get("maker"),
			Status: orderbook.OrderStatus(r.URL.Query().Get("status")),
			Limit:  defaultLimit,
			Offset: 0,
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
		list, err := svc.ListOrders(r.Context(), filter)
		if err != nil {
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ErrorMsg: "internal error", Err: err}
		}
		body := make([]*OrderResponse, len(list))
		for i := range list {
			body[i] = orderToResponse(list[i])
		}
		return &httpwrap.Response{StatusCode: http.StatusOK, Body: body}, nil
	}
}

func getOrderByHash(svc *orderbook.Service) httpwrap.HandlerFunc {
	return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			return nil, httpwrap.NewInvalidParamErrorResponse("hash")
		}
		order, err := svc.OrderByHash(r.Context(), hash)
		if err != nil {
			if errors.Is(err, orderbook.ErrNotFound) {
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusNotFound, ErrorMsg: "not found", Err: err}
			}
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ErrorMsg: "internal error", Err: err}
		}
		return &httpwrap.Response{StatusCode: http.StatusOK, Body: orderToResponse(order)}, nil
	}
}

// CancelOrderRequest is the JSON body for PATCH /orders/:hash/cancel.
type CancelOrderRequest struct {
	Maker string `json:"maker"`
}

func cancelOrder(svc *orderbook.Service) httpwrap.HandlerFunc {
	return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			return nil, httpwrap.NewInvalidParamErrorResponse("hash")
		}
		var req CancelOrderRequest
		if err := decodeBody(r, &req); err != nil {
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ErrorMsg: "invalid body", Err: err}
		}
		if req.Maker == "" {
			return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ErrorMsg: "maker is required", Err: nil}
		}
		order, err := svc.CancelOrder(r.Context(), hash, req.Maker)
		if err != nil {
			switch {
			case errors.Is(err, orderbook.ErrNotFound):
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusNotFound, ErrorMsg: "not found", Err: err}
			case errors.Is(err, orderbook.ErrUnauthorized):
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusForbidden, ErrorMsg: err.Error(), Err: err}
			case errors.Is(err, orderbook.ErrInvalidStateTransition):
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ErrorMsg: err.Error(), Err: err}
			default:
				return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ErrorMsg: "internal error", Err: err}
			}
		}
		return &httpwrap.Response{StatusCode: http.StatusOK, Body: orderToResponse(order)}, nil
	}
}

func trimHexPrefix(s string) string {
	if len(s) >= 2 && (s[0:2] == "0x" || s[0:2] == "0X") {
		return s[2:]
	}
	return s
}

func decodeBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
