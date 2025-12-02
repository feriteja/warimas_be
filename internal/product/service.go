package product

import (
	"context"
	"errors"
	"warimas-be/internal/graph/model"
	servicepkg "warimas-be/internal/service"
)

type Service interface {
	GetAll(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]model.CategoryProduct, error)
	Create(name string, price float64, stock int) (model.Product, error)
	GetPackages(ctx context.Context, filter *model.PackageFilterInput, sort *model.PackageSortInput, limit, page int32) ([]*model.Package, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]model.CategoryProduct, error) {
	return s.repo.GetAll(opts)
}

func (s *service) Create(name string, price float64, stock int) (model.Product, error) {
	if name == "" {
		return model.Product{}, errors.New("name cannot be empty")
	}
	if price <= 0 {
		return model.Product{}, errors.New("price must be positive")
	}

	newProduct := model.Product{
		Name:  name,
		Price: price,
		Stock: int32(stock),
	}
	return s.repo.Create(newProduct)
}

func (s *service) GetPackages(
	ctx context.Context,
	filter *model.PackageFilterInput,
	sort *model.PackageSortInput,
	limit, page int32,
) ([]*model.Package, error) {

	offset := page * limit

	return s.repo.GetPackages(ctx, filter, sort, limit, offset)
}
