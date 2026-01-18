package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, email, password, role string) (*User, error) {
	args := m.Called(ctx, email, password, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

// --- Tests ---

func TestService_Register(t *testing.T) {
	ctx := context.Background()
	email := "john@example.com"
	password := "password123"

	// Set JWT secret for token generation
	t.Setenv("JWT_SECRET", "testsecret")

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		expectedUser := &User{
			ID:    1,
			Email: email,
			Role:  RoleUser,
		}

		// Service hashes password, so we accept any string for the second arg
		mockRepo.On("Create", ctx, email, mock.AnythingOfType("string"), string(RoleUser)).Return(expectedUser, nil)

		token, res, err := svc.Register(ctx, email, password)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, token)
		assert.Equal(t, email, res.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("EmailAlreadyExists", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		// Simulate DB constraint error
		mockRepo.On("Create", ctx, email, mock.AnythingOfType("string"), string(RoleUser)).Return(nil, errors.New("users_email_key"))

		_, _, err := svc.Register(ctx, email, password)
		assert.Error(t, err)
		assert.Equal(t, ErrEmailExists, err)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("Create", ctx, email, mock.AnythingOfType("string"), string(RoleUser)).Return(nil, errors.New("db error"))

		_, _, err := svc.Register(ctx, email, password)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})

	t.Run("JWTError", func(t *testing.T) {
		// Unset JWT_SECRET to force GenerateJWT to fail
		t.Setenv("JWT_SECRET", "")
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("Create", ctx, email, mock.AnythingOfType("string"), string(RoleUser)).Return(&User{ID: 1}, nil)

		_, _, err := svc.Register(ctx, email, password)
		assert.Error(t, err)
	})
}

func TestService_Login(t *testing.T) {
	ctx := context.Background()
	email := "john@example.com"
	password := "password123"

	t.Setenv("JWT_SECRET", "testsecret")

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		// Generate a real hash so CheckPasswordHash succeeds
		hashedPassword, _ := HashPassword(password)
		user := &User{ID: 1, Email: email, Password: hashedPassword, Role: RoleUser}

		mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

		token, _, err := svc.Login(ctx, email, password)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		hashedPassword, _ := HashPassword("correct-password")
		user := &User{ID: 1, Email: email, Password: hashedPassword}

		mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

		_, _, err := svc.Login(ctx, email, "wrong-password")
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	})

	t.Run("UserNotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("FindByEmail", ctx, email).Return(nil, errors.New("not found"))

		_, _, err := svc.Login(ctx, email, password)
		assert.Error(t, err)
	})
}

func TestService_GetUserByEmail(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := &User{ID: 1, Email: email}

		mockRepo.On("FindByEmail", ctx, email).Return(expected, nil)

		res, err := svc.GetUserByEmail(ctx, email)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		// Service swallows error and returns the user object (likely empty/partial)
		mockRepo.On("FindByEmail", ctx, email).Return(nil, errors.New("db error"))

		res, err := svc.GetUserByEmail(ctx, email)
		assert.NoError(t, err)
		assert.Nil(t, res)
	})
}
