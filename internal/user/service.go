package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

type Service interface {
	Register(ctx context.Context, email, password string) (string, User, error)
	Login(email, password string) (string, User, error)
	GetUserByEmail(email string) (User, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Register(ctx context.Context, email, password string) (string, User, error) {
	log := logger.FromCtx(ctx)

	hashed, err := HashPassword(password)
	if err != nil {
		log.Error("failed to hash password", zap.Error(err))
		return "", User{}, err
	}

	u, err := s.repo.Create(ctx, email, hashed, string(RoleUser))
	if err != nil {
		log.Error("failed to create user", zap.String("email", email), zap.Error(err))
		if strings.Contains(err.Error(), "users_email_key") {
			return "", User{}, ErrEmailExists
		}
		return "", User{}, err
	}

	token, err := GenerateJWT(u.ID, string(u.Role), email)
	if err != nil {
		log.Error("failed to generate jwt", zap.String("user_id", fmt.Sprint(u.ID)), zap.Error(err))
		return "", User{}, err
	}

	log.Info("register service completed",
		zap.String("user_id", fmt.Sprint(u.ID)),
		zap.String("email", email),
	)

	return token, u, nil
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
