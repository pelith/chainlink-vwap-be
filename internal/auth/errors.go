package auth

import "errors"

var (
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenNotFound = errors.New("token not found")
)
