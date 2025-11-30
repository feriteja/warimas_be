package graph

import (
	"context"
	"fmt"

	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

func (r *mutationResolver) Register(ctx context.Context, input model.RegisterInput) (*model.AuthResponse, error) {
	log := logger.FromCtx(ctx)

	token, u, err := r.UserSvc.Register(ctx, input.Email, input.Password)
	if err != nil {
		log.Warn("register failed", zap.String("email", input.Email), zap.Error(err))
		return nil, err
	}

	log.Info("user registered successfully",
		zap.String("user_id", fmt.Sprint(u.ID)),
		zap.String("email", u.Email),
	)

	return &model.AuthResponse{
		Token: token,
		User: &model.User{
			ID:    fmt.Sprint(u.ID),
			Email: u.Email,
			Role:  model.Role(u.Role),
		},
	}, nil
}

func (r *mutationResolver) Login(ctx context.Context, input model.LoginInput) (*model.AuthResponse, error) {
	token, u, err := r.UserSvc.Login(input.Email, input.Password)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Token: token,
		User: &model.User{
			ID:    fmt.Sprint(u.ID),
			Email: u.Email,
			Role:  model.Role(u.Role),
		},
	}, nil
}
