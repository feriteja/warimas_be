package graph

import (
	"context"
	"fmt"

	"warimas-be/internal/graph/model"
)

func (r *mutationResolver) Register(ctx context.Context, input model.RegisterInput) (*model.AuthResponse, error) {
	token, u, err := r.UserSvc.Register(input.Email, input.Password)
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
