package packages

import (
	"context"
	"testing"
	"warimas-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func mockContextWithRole(role string) context.Context {
	return utils.SetUserContext(context.Background(), 1, "test@example.com", role)
}

func (m *MockRepository) GetPackages(ctx context.Context, filter *PackageFilterInput, sort *PackageSortInput, limit, page int32, includeDisabled bool) ([]*Package, error) {
	args := m.Called(ctx, filter, sort, limit, page, includeDisabled)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Package), args.Error(1)
}

func TestService_GetPackages(t *testing.T) {
	ctx := mockContextWithRole("USER")

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		// Service defaults limit=20, page=1, includeDisabled=false (for USER)
		mockRepo.On("GetPackages", ctx, (*PackageFilterInput)(nil), (*PackageSortInput)(nil), int32(20), int32(0), false).
			Return([]*Package{}, nil)

		_, err := svc.GetPackages(ctx, nil, nil, 0, 0)
		assert.NoError(t, err)
	})

	t.Run("Admin", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		adminCtx := mockContextWithRole("ADMIN")
		mockRepo.On("GetPackages", adminCtx, (*PackageFilterInput)(nil), (*PackageSortInput)(nil), int32(20), int32(0), true).
			Return([]*Package{}, nil)

		_, err := svc.GetPackages(adminCtx, nil, nil, 0, 0)
		assert.NoError(t, err)
	})

	t.Run("PaginationLogic", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		// Test limit capping and page normalization
		// Limit 150 -> 100, Page -1 -> 1
		// Offset = (1-1)*100 = 0
		mockRepo.On("GetPackages", ctx, mock.Anything, mock.Anything, int32(100), int32(0), false).
			Return([]*Package{}, nil)

		_, err := svc.GetPackages(ctx, nil, nil, 150, -1)
		assert.NoError(t, err)
	})
}
