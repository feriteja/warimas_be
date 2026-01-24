package packages

import (
	"context"
	"testing"
	"warimas-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRepository struct {
	mock.Mock
}

func mockContextWithRole(role string) context.Context {
	return utils.SetUserContext(context.Background(), 1, "test@example.com", role)
}

func (m *MockRepository) CreatePackage(ctx context.Context, input CreatePackageInput, userID uint) (*Package, error) {
	args := m.Called(ctx, input, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Package), args.Error(1)
}

func (m *MockRepository) GetPackages(ctx context.Context, filter *PackageFilterInput, sort *PackageSortInput, limit, page int32, includeDisabled bool, viewerID *uint) ([]*Package, int64, error) {
	args := m.Called(ctx, filter, sort, limit, page, includeDisabled, viewerID)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*Package), args.Get(1).(int64), args.Error(2)
}

func TestService_GetPackages(t *testing.T) {
	ctx := mockContextWithRole("USER")

	t.Run("Success", func(t *testing.T) {
		limit := int32(20)
		page := int32(1)

		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		// Service defaults limit=20, page=1, includeDisabled=false (for USER)
		expectedPkgs := []*Package{{ID: "1", Name: "test"}}
		mockRepo.On("GetPackages", ctx, (*PackageFilterInput)(nil), (*PackageSortInput)(nil), limit, page, false, mock.MatchedBy(func(id *uint) bool { return id != nil && *id == 1 })).
			Return(expectedPkgs, int64(1), nil)

		_, _, err := svc.GetPackages(ctx, nil, nil, 0, 0)
		assert.NoError(t, err)
	})

	t.Run("Admin", func(t *testing.T) {
		limit := int32(20)
		page := int32(1)

		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		adminCtx := mockContextWithRole("ADMIN")
		mockRepo.On("GetPackages", adminCtx, (*PackageFilterInput)(nil), (*PackageSortInput)(nil), limit, page, true, mock.MatchedBy(func(id *uint) bool { return id != nil && *id == 1 })).
			Return([]*Package{}, int64(0), nil)

		_, _, err := svc.GetPackages(adminCtx, nil, nil, 0, 0)
		assert.NoError(t, err)
	})

	t.Run("Pagination", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("GetPackages", ctx, mock.Anything, mock.Anything, int32(100), int32(2), false, mock.Anything).
			Return([]*Package{}, int64(0), nil)

		_, _, err := svc.GetPackages(ctx, nil, nil, 100, 2)
		require.NoError(t, err)
	})
}
