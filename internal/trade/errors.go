package trade

import "errors"

var (
	ErrNotFound               = errors.New("trade not found")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)
