package order

import "errors"

var (
	ErrAddressNotFound = errors.New("address not found")
	ErrOrderNotFound   = errors.New("order not found")
	ErrUnauthorized    = errors.New("unauthorized")
)
