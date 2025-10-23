package product

import (
	"errors"
)

type Service interface {
	GetAll() ([]Product, error)
	Create(name string, price float64, stock int) (Product, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetAll() ([]Product, error) {
	return s.repo.GetAll()
}

func (s *service) Create(name string, price float64, stock int) (Product, error) {
	if name == "" {
		return Product{}, errors.New("name cannot be empty")
	}
	if price <= 0 {
		return Product{}, errors.New("price must be positive")
	}

	newProduct := Product{
		Name:  name,
		Price: price,
		Stock: stock,
	}
	return s.repo.Create(newProduct)
}
