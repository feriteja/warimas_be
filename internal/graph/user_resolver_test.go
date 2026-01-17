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

		jwtToken := "jwt-token"
		ctx := context.Background()
		input := model.RegisterInput{Email: "test@test.com", Password: "password"}

		// Mock service return
		serviceUser := &user.User{ID: 1, Email: "test@test.com", Role: "USER"}

		mockSvc.On("Register", ctx, input.Email, input.Password).Return(jwtToken, serviceUser, nil)

		res, err := mr.Register(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, jwtToken, *res.Token)
		assert.Equal(t, "test@test.com", res.User.Email)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		input := model.RegisterInput{Email: "test@test.com"}
		mockSvc.On("Register", context.Background(), input.Email, input.Password).Return("", nil, errors.New("email exists"))

		_, err := mr.Register(context.Background(), input)
		assert.Error(t, err)
	})
}

func TestMutationResolver_Login(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockUserService)
		resolver := &Resolver{UserSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.LoginInput{Email: "test@test.com", Password: "password"}
		jwtToken := "jwt-token"
		serviceUser := &user.User{ID: 1, Email: "test@test.com", Role: "USER"}

		mockSvc.On("Login", ctx, input.Email, input.Password).Return(jwtToken, serviceUser, nil)

		res, err := mr.Login(ctx, input)
		assert.NoError(t, err)
		assert.Equal(t, jwtToken, *res.Token)
	})
}
