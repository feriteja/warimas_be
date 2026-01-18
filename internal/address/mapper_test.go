package address

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMapAddressToGraphQL(t *testing.T) {
	id := uuid.New()
	addr2 := "Apt 1"
	addr := &Address{
		ID:           id,
		Name:         "Home",
		ReceiverName: "John",
		Phone:        "123",
		Address1:     "Street 1",
		Address2:     &addr2,
		City:         "City",
		Province:     "Prov",
		Postal:       "12345",
		Country:      "ID",
		IsDefault:    true,
	}

	gqlAddr := MapAddressToGraphQL(addr)

	assert.Equal(t, id.String(), gqlAddr.ID)
	assert.Equal(t, "Home", gqlAddr.Name)
	assert.Equal(t, "Apt 1", *gqlAddr.AddressLine2)
	assert.True(t, gqlAddr.IsDefault)
}
