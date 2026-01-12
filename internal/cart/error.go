package cart

import "errors"

var (
	ErrInvalidQuantity      = errors.New("invalid cart quantity")
	ErrCartItemNotFound     = errors.New("cart item not found")
	ErrFailedUpdateCart     = errors.New("failed to update cart item")
	ErrFailedRemoveCart     = errors.New("failed to remove cart item")
	ErrCartEmpty            = errors.New("cart is already empty")
	ErrFailedClearCart      = errors.New("failed to clear cart")
	ErrFailedGetCartItem    = errors.New("failed to get cart item")
	ErrFailedUpdateCartItem = errors.New("failed to update cart item")
	ErrCartItemAlreadyExist = errors.New("cart item already exists")
	ErrFailedCreateCartItem = errors.New("failed to create cart item")
	ErrFailedGetCartRows    = errors.New("failed to get cart rows")
	PgUniqueViolation       = "23505"
)
