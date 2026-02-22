package config

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidLogLevel = errors.New("invalid log level")
	ErrInvalidUUID     = errors.New("invalid UUID format")
)

type UnsupportedTypeError struct {
	Type string
}

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("unsupported type: %s", e.Type)
}
