package address

import (
	"context"
	"errors"
	"testing"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetByUserID(ctx context.Context, userID uint) ([]*Address, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Address), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Address, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Address), args.Error(1)
}

func (m *MockRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]Address, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Address), args.Error(1)
}

func (m *MockRepository) Create(ctx context.Context, addr *Address) error {
	args := m.Called(ctx, addr)
	return args.Error(0)
}

func (m *MockRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ClearDefault(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepository) SetDefault(ctx context.Context, userID uint, addressID uuid.UUID) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

// --- Helpers ---

// mockContextWithUser creates a context that utils.GetUserIDFromContext should recognize.
// ADJUST THIS implementation to match how your utils package extracts the user ID.
func mockContextWithUser(userID uint) context.Context {
	// Attempting to use a common pattern. If utils uses a specific key type,
	// you might need to import that key or use utils.NewContext(userID) if available.
	// For now, we assume we can mock the utils behavior or that utils reads from this value.
	// If utils.GetUserIDFromContext fails, tests will return "unauthenticated".

	// Since we cannot modify utils, we rely on the fact that we are mocking the behavior
	// around the context. However, the service calls utils.GetUserIDFromContext directly.
	// We will assume for this test file that we can inject it via a known key or that
	// the user will adjust this helper.

	// Placeholder: Assuming utils uses a string key "userID" or similar.
	// If utils is using a private context key, you must expose a setter in utils
	// or use a test helper in the utils package.
	return utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
}

// Since utils.NewContextWithUserID might not exist in your codebase,
// here is a fallback mock for the sake of compilation if you need to create it in utils:
// func NewContextWithUserID(ctx context.Context, id uint) context.Context {
//     return context.WithValue(ctx, "user_id", id)
// }
//
// If you cannot change utils, ensure this test file is in a package that can access the key,
// or that utils exports a setter.

// --- Tests ---

func TestService_List(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := []*Address{{ID: uuid.New(), UserID: userID}}
		mockRepo.On("GetByUserID", ctx, userID).Return(expected, nil)

		result, err := svc.List(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		// Empty context
		_, err := svc.List(context.Background())
		assert.Error(t, err)
		assert.Equal(t, "unauthenticated", err.Error())
	})
}

func TestService_Get(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)
	addrID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := &Address{ID: addrID, UserID: userID, IsActive: true}
		mockRepo.On("GetByID", ctx, addrID).Return(expected, nil)

		result, err := svc.Get(ctx, addrID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("NotFound_Repo", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("GetByID", ctx, addrID).Return(nil, errors.New("db error"))

		_, err := svc.Get(ctx, addrID)
		assert.Error(t, err)
	})

	t.Run("Unauthorized_WrongUser", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		otherUserAddr := &Address{ID: addrID, UserID: 999, IsActive: true}
		mockRepo.On("GetByID", ctx, addrID).Return(otherUserAddr, nil)

		_, err := svc.Get(ctx, addrID)
		if assert.Error(t, err) {
			assert.Equal(t, "address not found", err.Error()) // Security: return not found
		}
	})

	t.Run("Unauthorized_Inactive", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		inactiveAddr := &Address{ID: addrID, UserID: userID, IsActive: false}
		mockRepo.On("GetByID", ctx, addrID).Return(inactiveAddr, nil)

		_, err := svc.Get(ctx, addrID)
		if assert.Error(t, err) {
			assert.Equal(t, "address not found", err.Error())
		}
	})
}

func TestService_Create(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)

	input := CreateAddressInput{
		Name:         "Home",
		ReceiverName: "John",
		SetAsDefault: true,
	}

	t.Run("Success_WithDefault", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("ClearDefault", ctx, userID).Return(nil)

		// Use MatchedBy because ID is generated inside Create
		mockRepo.On("Create", ctx, mock.MatchedBy(func(a *Address) bool {
			return a.UserID == userID && a.Name == input.Name && a.IsDefault == true && a.IsActive == true
		})).Return(nil)

		res, err := svc.Create(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_NoDefault", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		inputNoDefault := input
		inputNoDefault.SetAsDefault = false

		mockRepo.On("Create", ctx, mock.MatchedBy(func(a *Address) bool {
			return a.IsDefault == false
		})).Return(nil)

		res, err := svc.Create(ctx, inputNoDefault)

		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		inputNoDefault := input
		inputNoDefault.SetAsDefault = false
		mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("db error"))

		_, err := svc.Create(ctx, inputNoDefault)
		assert.Error(t, err)
	})
}

func TestService_Update(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)

	oldID := uuid.New()
	input := UpdateAddressInput{
		AddressID:    oldID.String(),
		Name:         "New Home",
		SetAsDefault: true,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		oldAddr := &Address{ID: oldID, UserID: userID, IsActive: true}

		// 1. Get old
		mockRepo.On("GetByID", ctx, oldID).Return(oldAddr, nil)
		// 2. Deactivate old
		mockRepo.On("Deactivate", ctx, oldID).Return(nil)
		// 3. Clear default (since input.SetAsDefault is true)
		mockRepo.On("ClearDefault", ctx, userID).Return(nil)
		// 4. Create new
		mockRepo.On("Create", ctx, mock.MatchedBy(func(a *Address) bool {
			return a.ID != oldID && a.Name == input.Name && a.IsDefault == true
		})).Return(nil)

		res, err := svc.Update(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEqual(t, oldID, res.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("InvalidID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		badInput := input
		badInput.AddressID = "invalid-uuid"
		_, err := svc.Update(ctx, badInput)
		assert.Error(t, err)
		assert.Equal(t, "invalid address id", err.Error())
	})

	t.Run("NotFoundOrUnauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("GetByID", ctx, oldID).Return(nil, errors.New("not found"))
		_, err := svc.Update(ctx, input)
		assert.Error(t, err)
		assert.Equal(t, "address not found", err.Error())
	})
}

func TestService_Delete(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)
	addrID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		addr := &Address{ID: addrID, UserID: userID}
		mockRepo.On("GetByID", ctx, addrID).Return(addr, nil)
		mockRepo.On("Deactivate", ctx, addrID).Return(nil)

		err := svc.Delete(ctx, addrID)
		assert.NoError(t, err)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		addr := &Address{ID: addrID, UserID: 999}
		mockRepo.On("GetByID", ctx, addrID).Return(addr, nil)

		err := svc.Delete(ctx, addrID)
		assert.Error(t, err)
		assert.Equal(t, "address not found", err.Error())
	})
}

func TestService_SetDefaultAddress(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)
	addrID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		addr := &Address{ID: addrID, UserID: userID}
		mockRepo.On("GetByID", ctx, addrID).Return(addr, nil)
		mockRepo.On("ClearDefault", ctx, userID).Return(nil)
		mockRepo.On("SetDefault", ctx, userID, addrID).Return(nil)

		err := svc.SetDefaultAddress(ctx, addrID)
		assert.NoError(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("GetByID", ctx, addrID).Return(nil, errors.New("not found"))
		err := svc.SetDefaultAddress(ctx, addrID)
		assert.Error(t, err)
	})
}
