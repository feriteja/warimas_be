package user

import (
	"errors"
	"log"
)

type Service interface {
	Register(email, password string) (string, User, error)
	Login(email, password string) (string, User, error)
	GetUserByEmail(email string) (User, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Register(email, password string) (string, User, error) {
	hashed, err := HashPassword(password)
	if err != nil {
		return "", User{}, err
	}

	u, err := s.repo.Create(email, hashed, string(RoleUser))
	if err != nil {
		return "", User{}, err
	}

	token, err := GenerateJWT(u.ID, string(u.Role), email)
	return token, u, err
}

func (s *service) Login(email, password string) (string, User, error) {
	u, err := s.repo.FindByEmail(email)
	if err != nil {
		log.Println("email not found")
		return "", User{}, errors.New("invalid email or password")
	}

	if !CheckPasswordHash(password, u.Password) {
		log.Println("password not match")
		return "", User{}, errors.New("invalid email or password")
	}

	token, err := GenerateJWT(u.ID, string(u.Role), email)
	return token, u, err
}

func (s *service) GetUserByEmail(email string) (User, error) {
	return s.repo.FindByEmail(email)
}
