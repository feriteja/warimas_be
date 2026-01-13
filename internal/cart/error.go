package cart

import "errors"

var (
	// -- Authentication/Authorization --
	ErrUserNotAuthenticated = errors.New("user not authenticated")

	// -- Validation & Input --
	ErrInvalidQuantity        = errors.New("invalid cart quantity")
	ErrInvalidRemoveCartInput = errors.New("invalid remove cart input")

	// -- Resource State --
	ErrCartItemNotFound     = errors.New("cart item not found")
	ErrCartItemAlreadyExist = errors.New("cart item already exists")
	ErrCartEmpty            = errors.New("cart is already empty")

	// -- Database & Operation Failures --
	ErrFailedGetCartItem    = errors.New("failed to get cart item")
	ErrFailedGetCartRows    = errors.New("failed to get cart rows")
	ErrFailedCreateCartItem = errors.New("failed to create cart item")
	ErrFailedUpdateCart     = errors.New("failed to update cart item")
	ErrFailedRemoveCart     = errors.New("failed to remove cart item")
	ErrFailedClearCart      = errors.New("failed to clear cart")

	// -- Constants (External Systems) --
	PgUniqueViolation = "23505"
)
