package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
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

func (m *MockRepository) UpdatePassword(ctx context.Context, email, password string) error {
	args := m.Called(ctx, email, password)
	return args.Error(0)
}

func (m *MockRepository) GetProfile(ctx context.Context, userID uint) (*Profile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Profile), args.Error(1)
}

func (m *MockRepository) CreateProfile(ctx context.Context, p *Profile) (*Profile, error) {
	args := m.Called(ctx, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Profile), args.Error(1)
}

func (m *MockRepository) UpdateProfile(ctx context.Context, p *Profile) (*Profile, error) {
	args := m.Called(ctx, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Profile), args.Error(1)
}

func TestService_Register(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		expectedUser := &User{
			ID:       1,
			Email:    email,
			Password: "hashed_password",
			Role:     RoleUser,
		}

		// We match role as string because repository expects string
		mockRepo.On("Create", ctx, email, mock.AnythingOfType("string"), string(RoleUser)).Return(expectedUser, nil)

		token, user, err := svc.Register(ctx, email, password)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, expectedUser, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("EmailExists", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("Create", ctx, email, mock.Anything, string(RoleUser)).Return(nil, errors.New("duplicate key value violates unique constraint \"users_email_key\""))

		_, _, err := svc.Register(ctx, email, password)

		assert.Error(t, err)
		assert.Equal(t, ErrEmailExists, err)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("Create", ctx, email, mock.Anything, string(RoleUser)).Return(nil, errors.New("db error"))

		_, _, err := svc.Register(ctx, email, password)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestService_Login(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"

	hashedPassword, _ := HashPassword(password)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		user := &User{
			ID:       1,
			Email:    email,
			Password: hashedPassword,
			Role:     RoleUser,
		}

		mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

		token, u, err := svc.Login(ctx, email, password)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, user, u)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("FindByEmail", ctx, email).Return(nil, errors.New("not found"))

		_, _, err := svc.Login(ctx, email, password)

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		user := &User{
			ID:       1,
			Email:    email,
			Password: hashedPassword,
			Role:     RoleUser,
		}

		mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

		_, _, err := svc.Login(ctx, email, "wrongpassword")

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
	})
}

func TestService_GetUserByEmail(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expectedUser := &User{ID: 1, Email: email}

		mockRepo.On("FindByEmail", ctx, email).Return(expectedUser, nil)

		user, err := svc.GetUserByEmail(ctx, email)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("FindByEmail", ctx, email).Return(nil, errors.New("db error"))

		_, err := svc.GetUserByEmail(ctx, email)
		assert.Error(t, err)
	})
}

func TestService_ForgotPassword(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")
	ctx := context.Background()
	email := "test@example.com"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		user := &User{ID: 1, Email: email, Role: RoleUser}

		mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

		err := svc.ForgotPassword(ctx, email)
		assert.NoError(t, err)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("FindByEmail", ctx, email).Return(nil, errors.New("not found"))

		err := svc.ForgotPassword(ctx, email)
		assert.NoError(t, err) // Should return nil to prevent enumeration
	})
}

func TestService_ResetPassword(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")
	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newpassword"

	// Generate a valid token
	token, _ := GenerateJWT(1, "USER", email, nil)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("UpdatePassword", ctx, email, mock.Anything).Return(nil)

		err := svc.ResetPassword(ctx, token, newPassword)
		assert.NoError(t, err)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		err := svc.ResetPassword(ctx, "invalid-token", newPassword)
		assert.Error(t, err)
		assert.Equal(t, "invalid or expired token", err.Error())
	})

	t.Run("UpdateError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("UpdatePassword", ctx, email, mock.Anything).Return(errors.New("db error"))

		err := svc.ResetPassword(ctx, token, newPassword)
		assert.Error(t, err)
	})
}

func TestService_GetOrCreateProfile(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)

	t.Run("ProfileExists", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		expectedProfile := &Profile{
			ID:     uuid.New(),
			UserID: userID,
		}

		mockRepo.On("GetProfile", ctx, userID).Return(expectedProfile, nil)

		result, err := svc.GetOrCreateProfile(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedProfile, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ProfileNotFound_CreateNew", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		// GetProfile returns ErrProfileNotFound
		mockRepo.On("GetProfile", ctx, userID).Return(nil, ErrProfileNotFound)

		// Expect CreateProfile to be called
		createdProfile := &Profile{
			ID:     uuid.New(),
			UserID: userID,
		}

		// Use MatchedBy to verify the input to CreateProfile
		mockRepo.On("CreateProfile", ctx, mock.MatchedBy(func(p *Profile) bool {
			return p.UserID == userID
		})).Return(createdProfile, nil)

		result, err := svc.GetOrCreateProfile(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, createdProfile, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetProfile_DBError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("GetProfile", ctx, userID).Return(nil, errors.New("db error"))

		result, err := svc.GetOrCreateProfile(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertNotCalled(t, "CreateProfile")
	})

	t.Run("CreateProfile_DBError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("GetProfile", ctx, userID).Return(nil, ErrProfileNotFound)
		mockRepo.On("CreateProfile", ctx, mock.Anything).Return(nil, errors.New("create error"))

		result, err := svc.GetOrCreateProfile(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "create error", err.Error())
	})
}

func TestService_UpdateProfile(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	fullName := "John Doe"
	bio := "Hello World"
	dob := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

	params := UpdateProfileParams{
		UserID:      userID,
		FullName:    &fullName,
		Bio:         &bio,
		DateOfBirth: &dob,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		updatedProfile := &Profile{
			ID:          uuid.New(),
			UserID:      userID,
			FullName:    &fullName,
			Bio:         &bio,
			DateOfBirth: &dob,
		}

		mockRepo.On("UpdateProfile", ctx, mock.MatchedBy(func(p *Profile) bool {
			return p.UserID == userID &&
				*p.FullName == fullName &&
				*p.Bio == bio &&
				p.DateOfBirth.Equal(dob)
		})).Return(updatedProfile, nil)

		result, err := svc.UpdateProfile(ctx, params)

		assert.NoError(t, err)
		assert.Equal(t, updatedProfile, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DBError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("UpdateProfile", ctx, mock.Anything).Return(nil, errors.New("update error"))

		result, err := svc.UpdateProfile(ctx, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "update error", err.Error())
	})
}

func TestService_Register_JWTError(t *testing.T) {
	t.Setenv("JWT_SECRET", "") // Force error
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"

	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)

	expectedUser := &User{ID: 1, Email: email, Role: RoleUser}
	mockRepo.On("Create", ctx, email, mock.Anything, string(RoleUser)).Return(expectedUser, nil)

	_, _, err := svc.Register(ctx, email, password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET is not set")
}

func TestService_Register_HashError(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")
	ctx := context.Background()
	email := "test@example.com"
	// Bcrypt max password length is 72 bytes. Sending 73+ bytes triggers error.
	longPassword := string(make([]byte, 73))

	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)

	_, _, err := svc.Register(ctx, email, longPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password length exceeds")
}

func TestService_Login_JWTError(t *testing.T) {
	t.Setenv("JWT_SECRET", "") // Force error
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	hashed, _ := HashPassword(password)

	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)

	user := &User{ID: 1, Email: email, Password: hashed, Role: RoleUser}
	mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

	_, _, err := svc.Login(ctx, email, password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error") // Service wraps it
}

func TestService_ForgotPassword_JWTError(t *testing.T) {
	t.Setenv("JWT_SECRET", "") // Force error
	ctx := context.Background()
	email := "test@example.com"

	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)

	user := &User{ID: 1, Email: email, Role: RoleUser}
	mockRepo.On("FindByEmail", ctx, email).Return(user, nil)

	err := svc.ForgotPassword(ctx, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET is not set")
}

func TestService_ResetPassword_HashError(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")
	ctx := context.Background()
	token, _ := GenerateJWT(1, "USER", "test@example.com", nil)

	// Bcrypt max password length is 72 bytes. Sending 73+ bytes triggers error.
	longPassword := string(make([]byte, 73))

	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)

	err := svc.ResetPassword(ctx, token, longPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password length exceeds")
}
