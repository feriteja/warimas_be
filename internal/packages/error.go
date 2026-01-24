package packages

import "errors"

var (
	ErrPackagesNotFound = errors.New("order not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrForbidden        = errors.New("unauthorized")
)
