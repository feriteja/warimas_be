package graph

import (
	"context"
	"warimas-be/internal/packages"

	"github.com/stretchr/testify/mock"
)

type MockPackageService struct {
	mock.Mock
}

func (m *MockPackageService) GetPackages(ctx context.Context, filter *packages.PackageFilterInput, sort *packages.PackageSortInput, limit, page int32) ([]*packages.Package, error) {
	return nil, nil
}
