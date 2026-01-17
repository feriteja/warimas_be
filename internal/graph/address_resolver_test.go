package graph

import (
	"context"
	"errors"
	"testing"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockAddressService struct {
	mock.Mock
}

func (m *MockAddressService) Create(ctx context.Context, input address.CreateAddressInput) (*address.Address, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*address.Address), args.Error(1)
}

func (m *MockAddressService) Update(ctx context.Context, input address.UpdateAddressInput) (*address.Address, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*address.Address), args.Error(1)
}

func (m *MockAddressService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAddressService) SetDefaultAddress(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAddressService) List(ctx context.Context) ([]*address.Address, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*address.Address), args.Error(1)
}

func (m *MockAddressService) Get(ctx context.Context, id uuid.UUID) (*address.Address, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*address.Address), args.Error(1)
}

// --- Helpers ---

func boolPtr(b bool) *bool {
	return &b
}

// --- Tests ---

func TestMutationResolver_CreateAddress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

		input := model.CreateAddressInput{
			Address: &model.AddressInput{
				Name:         "Home",
				ReceiverName: "John",
				Phone:        "123",
				AddressLine1: "Street",
				City:         "City",
				Province:     "Prov",
				PostalCode:   "12345",
				Country:      "ID",
			},
			SetAsDefault: boolPtr(true),
		}

		expectedAddr := &address.Address{
			ID:   uuid.New(),
			Name: "Home",
		}

		mockSvc.On("Create", ctx, mock.MatchedBy(func(arg address.CreateAddressInput) bool {
			return arg.Name == "Home" && arg.SetAsDefault == true
		})).Return(expectedAddr, nil)

		res, err := mr.CreateAddress(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, expectedAddr.ID.String(), res.Address.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}

		_, err := mr.CreateAddress(context.Background(), model.CreateAddressInput{})
		assert.Error(t, err)
		assert.Equal(t, "unauthorized", err.Error())
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		mockSvc.On("Create", ctx, mock.Anything).Return(nil, errors.New("db error"))
		_, err := mr.CreateAddress(ctx, model.CreateAddressInput{})
		assert.Error(t, err)
	})
}

func TestMutationResolver_UpdateAddress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()

		input := model.UpdateAddressInput{
			AddressID: addrID.String(),
			Address: &model.AddressInput{
				Name: "New Home",
			},
			SetAsDefault: boolPtr(false),
		}

		expectedAddr := &address.Address{
			ID:   addrID,
			Name: "New Home",
		}

		mockSvc.On("Update", ctx, mock.MatchedBy(func(arg address.UpdateAddressInput) bool {
			return arg.AddressID == addrID.String() && arg.Name == "New Home"
		})).Return(expectedAddr, nil)

		res, err := mr.UpdateAddress(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, "New Home", res.Address.Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		mockSvc.On("Update", ctx, mock.Anything).Return(nil, errors.New("db error"))
		_, err := mr.UpdateAddress(ctx, model.UpdateAddressInput{})
		assert.Error(t, err)
	})
}

func TestMutationResolver_DeleteAddress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()
		input := model.DeleteAddressInput{AddressID: addrID.String()}

		mockSvc.On("Delete", ctx, addrID).Return(nil)

		res, err := mr.DeleteAddress(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()
		mockSvc.On("Delete", ctx, addrID).Return(errors.New("db error"))
		_, err := mr.DeleteAddress(ctx, model.DeleteAddressInput{AddressID: addrID.String()})
		assert.Error(t, err)
	})
}

func TestMutationResolver_SetDefaultAddress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()

		mockSvc.On("SetDefaultAddress", ctx, addrID).Return(nil)

		res, err := mr.SetDefaultAddress(ctx, addrID.String())

		assert.NoError(t, err)
		assert.True(t, res)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		mr := &mutationResolver{resolver}
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()
		mockSvc.On("SetDefaultAddress", ctx, addrID).Return(errors.New("db error"))
		_, err := mr.SetDefaultAddress(ctx, addrID.String())
		assert.Error(t, err)
	})
}

func TestQueryResolver_Addresses(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		expectedList := []*address.Address{
			{ID: uuid.New(), Name: "Home"},
		}

		mockSvc.On("List", ctx).Return(expectedList, nil)

		res, err := qr.Addresses(ctx)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Home", res[0].Name)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		qr := &queryResolver{resolver}
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		mockSvc.On("List", ctx).Return(nil, errors.New("db error"))
		_, err := qr.Addresses(ctx)
		assert.Error(t, err)
	})
}

func TestQueryResolver_Address(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()
		expectedAddr := &address.Address{ID: addrID, Name: "Home"}

		mockSvc.On("Get", ctx, addrID).Return(expectedAddr, nil)

		res, err := qr.Address(ctx, addrID.String())

		assert.NoError(t, err)
		assert.Equal(t, "Home", res.Name)
		assert.Equal(t, addrID.String(), res.ID)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockAddressService)
		resolver := &Resolver{AddressSvc: mockSvc}
		qr := &queryResolver{resolver}
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		addrID := uuid.New()
		mockSvc.On("Get", ctx, addrID).Return(nil, errors.New("db error"))
		_, err := qr.Address(ctx, addrID.String())
		assert.Error(t, err)
	})
}
