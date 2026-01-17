package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

type Service interface {
	Register(ctx context.Context, email, password string) (string, *User, error)
	Login(ctx context.Context, email, password string) (string, *User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Register(ctx context.Context, email, password string) (string, *User, error) {
	log := logger.FromCtx(ctx)

	hashed, err := HashPassword(password)
	if err != nil {
		log.Error("failed to hash password", zap.Error(err))
		return "", nil, err
	}

	u, err := s.repo.Create(ctx, email, hashed, string(RoleUser))
	if err != nil {
		log.Error("failed to create user", zap.String("email", email), zap.Error(err))
		if strings.Contains(err.Error(), "users_email_key") {
			return "", nil, ErrEmailExists
		}
		return "", nil, err
	}

	token, err := GenerateJWT(u.ID, string(u.Role), email, nil)
	if err != nil {
		log.Error("failed to generate jwt", zap.String("user_id", fmt.Sprint(u.ID)), zap.Error(err))
		return "", nil, err
	}

	log.Info("register service completed",
		zap.String("user_id", fmt.Sprint(u.ID)),
		zap.String("email", email),
	)

	return token, u, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, *User, error) {
	log := logger.FromCtx(ctx)

	log.Info("Login attempt",
		zap.String("email", email),
	)

	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		log.Warn("Login failed: email not found",
			zap.String("email", email),
			zap.Error(err),
		)
		return "", nil, errors.New("invalid credentials")
	}

	// Check password
	if !CheckPasswordHash(password, u.Password) {
		log.Warn("Login failed: incorrect password",
			zap.String("email", email),
			zap.Int("user_id", u.ID),
		)
		return "", nil, errors.New("invalid credentials")
	}

	// Generate token
	token, err := GenerateJWT(u.ID, string(u.Role), email, u.SellerID)
	if err != nil {
		log.Error("JWT generation failed",
			zap.String("email", email),
			zap.Int("user_id", u.ID),
			zap.Error(err),
		)
		return "", nil, errors.New("internal error")
	}

	log.Info("Login successful",
		zap.String("email", email),
		zap.Int("user_id", u.ID),
		zap.String("role", string(u.Role)),
	)

	return token, u, nil
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user, _ := s.repo.FindByEmail(ctx, email)
	return user, nil
}
