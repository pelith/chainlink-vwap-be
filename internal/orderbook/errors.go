package orderbook

import "errors"

var (
	ErrNotFound               = errors.New("order not found")
	ErrUnauthorized           = errors.New("only maker may cancel order")
	ErrInvalidInput           = errors.New("invalid input")
	ErrDuplicateOrderHash     = errors.New("order hash already exists")
	ErrExpired                = errors.New("order expired")
	ErrInvalidSignature       = errors.New("invalid EIP-712 signature")
	ErrInvalidDeltaBps        = errors.New("invalid deltaBps: 10000 + deltaBps must be > 0")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrNotExpired             = errors.New("order deadline has not passed")
)
