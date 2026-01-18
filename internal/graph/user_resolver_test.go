package graph

import (
	"context"
	"errors"
	"testing"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(ctx context.Context, email, password string) (string, *user.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(1) == nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*user.User), args.Error(2)
}

func (m *MockUserService) Login(ctx context.Context, email, password string) (string, *user.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(1) == nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*user.User), args.Error(2)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) ForgotPassword(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockUserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

// --- Tests ---

func TestMutationResolver_Register(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.RegisterInput{Email: "test@test.com", Password: "password"}
		expectedUser := &user.User{ID: 1, Email: "test@test.com", Role: "USER"}
		token := "token_123"

		mockSvc.On("Register", ctx, input.Email, input.Password).Return(token, expectedUser, nil)

		res, err := mr.Register(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, token, *res.Token)
		assert.Equal(t, "test@test.com", res.User.Email)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.RegisterInput{Email: "test@test.com", Password: "password"}

		mockSvc.On("Register", ctx, input.Email, input.Password).Return("", nil, errors.New("email exists"))

		_, err := mr.Register(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, "email exists", err.Error())
	})
}

func TestMutationResolver_Login(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.LoginInput{Email: "test@test.com", Password: "password"}
		expectedUser := &user.User{ID: 1, Email: "test@test.com", Role: "USER"}
		token := "token_123"

		mockSvc.On("Login", ctx, input.Email, input.Password).Return(token, expectedUser, nil)

		res, err := mr.Login(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, token, *res.Token)
		assert.Equal(t, "test@test.com", res.User.Email)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.LoginInput{Email: "test@test.com", Password: "password"}

		mockSvc.On("Login", ctx, input.Email, input.Password).Return("", nil, errors.New("invalid credentials"))

		_, err := mr.Login(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	})
}

func TestMutationResolver_ForgotPassword(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.ForgotPasswordInput{Email: "test@example.com"}

		mockSvc.On("ForgotPassword", ctx, input.Email).Return(nil)

		res, err := mr.ForgotPassword(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.Contains(t, *res.Message, "reset link")
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.ForgotPasswordInput{Email: "test@example.com"}

		mockSvc.On("ForgotPassword", ctx, input.Email).Return(errors.New("service error"))

		_, err := mr.ForgotPassword(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, "service error", err.Error())
	})
}

func TestMutationResolver_ResetPassword(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.ResetPasswordInput{
			Token:       "valid-token",
			NewPassword: "new-password",
		}

		mockSvc.On("ResetPassword", ctx, input.Token, input.NewPassword).Return(nil)

		res, err := mr.ResetPassword(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, "Password successfully reset", *res.Message)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.ResetPasswordInput{
			Token:       "invalid-token",
			NewPassword: "new-password",
		}

		mockSvc.On("ResetPassword", ctx, input.Token, input.NewPassword).Return(errors.New("invalid token"))

		_, err := mr.ResetPassword(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})
}
