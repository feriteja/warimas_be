package address

import (
	"github.com/google/uuid"
)

type Address struct {
	ID     uuid.UUID
	UserID uint

	Name  string
	Phone string

	Address1 string
	Address2 *string

	City     string
	Province string
	Postal   string
	Country  string

	IsDefault bool
	IsActive  bool
}

type CreateAddressInput struct {
	Name         string
	Phone        string
	AddressLine1 string
	AddressLine2 *string
	City         string
	Province     string
	PostalCode   string
	Country      string
	SetAsDefault bool
}

type UpdateAddressInput struct {
	AddressID    string
	Name         string
	Phone        string
	AddressLine1 string
	AddressLine2 *string
	City         string
	Province     string
	PostalCode   string
	Country      string
	SetAsDefault bool
}
