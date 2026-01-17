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
