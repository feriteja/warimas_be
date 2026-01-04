package address

import "warimas-be/internal/graph/model"

func MapAddressToGraphQL(a *Address) *model.Address {
	return &model.Address{
		ID:           a.ID.String(),
		Name:         a.Name,
		Phone:        a.Phone,
		AddressLine1: a.Address1,
		AddressLine2: a.Address2,
		City:         a.City,
		Province:     a.Province,
		PostalCode:   a.Postal,
		Country:      a.Country,
		IsDefault:    a.IsDefault,
	}
}
