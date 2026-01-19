package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

var ErrProfileNotFound = errors.New("profile not found")

type Service interface {
	Register(ctx context.Context, email, password string) (string, *User, error)
	Login(ctx context.Context, email, password string) (string, *User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	GetOrCreateProfile(ctx context.Context, userID uint) (*Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*Profile, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Register(ctx context.Context, email, password string) (string, *User, error) {
	log := logger.FromCtx(ctx).With(zap.String("email", email))

	log.Info("register service starting")
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

	token, err := GenerateJWT(u.ID, string(u.Role), email, u.SellerID)
	if err != nil {
		log.Error("failed to generate jwt", zap.Error(err))
		return "", nil, err
	}

	log.Info("register service completed",
		zap.String("user_id", fmt.Sprint(u.ID)),
		zap.String("email", email),
	)

	return token, u, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, *User, error) {
	log := logger.FromCtx(ctx).With(zap.String("email", email))

	log.Info("Login attempt",
		zap.String("email", email),
	)

	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		log.Warn("email not found",
			zap.String("email", email),
			zap.Error(err),
		)
		return "", nil, errors.New("invalid credentials")
	}

	// Check password
	if !CheckPasswordHash(password, u.Password) {
		log.Warn("incorrect password")
		return "", nil, errors.New("invalid credentials")
	}

	// Generate token
	token, err := GenerateJWT(u.ID, string(u.Role), email, u.SellerID)
	if err != nil {
		log.Error("failed to generate jwt",
			zap.Error(err),
		)
		return "", nil, errors.New("internal error")
	}

	log.Info("Login successful",
		zap.Int("user_id", u.ID),
		zap.String("role", string(u.Role)),
	)

	return token, u, nil
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	log := logger.FromCtx(ctx).With(zap.String("email", email))

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		log.Error("failed to get user by email", zap.Error(err))
		return nil, err
	}
	return user, nil
}

func (s *service) ForgotPassword(ctx context.Context, email string) error {
	log := logger.FromCtx(ctx).With(zap.String("email", email))

	// 1. Check if user exists
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// We return nil even if user is not found to prevent email enumeration
		log.Warn("email not found")
		return nil
	}

	// 2. Generate reset token (using existing JWT logic for simplicity)
	// In a real scenario, you might want a shorter expiration or a specific "reset" claim
	token, err := GenerateJWT(u.ID, string(u.Role), u.Email, u.SellerID)
	if err != nil {
		log.Error("failed to generate reset token", zap.Error(err))
		return err
	}

	// 3. Send Email (Mocked)
	// In production, call an email service here.
	log.Info("==================================================")
	log.Info("PASSWORD RESET LINK SENT", zap.String("email", email))
	log.Info("TOKEN: " + token)
	log.Info("==================================================")

	return nil
}

func (s *service) ResetPassword(ctx context.Context, token, newPassword string) error {
	log := logger.FromCtx(ctx).With(zap.String("token", token))

	claims, err := ParseJWT(token)
	if err != nil {
		log.Warn("reset password: invalid token", zap.Error(err))
		return errors.New("invalid or expired token")
	}

	log = log.With(zap.String("email", claims.Email))
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		log.Error("failed to hash password", zap.Error(err))
		return err
	}

	if err := s.repo.UpdatePassword(ctx, claims.Email, hashedPassword); err != nil {
		log.Error("reset password: failed to update password", zap.Error(err))
		return err
	}

	log.Info("password reset")
	return nil
}

func (s *service) GetOrCreateProfile(ctx context.Context, userID uint) (*Profile, error) {
	log := logger.FromCtx(ctx).With(zap.Uint("user_id", userID))

	// 1. Try to get existing profile
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		// If error is anything other than "not found", return it
		// Assuming repository returns a specific error or we check the error string/type
		// For now, we'll assume a specific error string or nil profile means not found if err is nil
		if !errors.Is(err, ErrProfileNotFound) {
			log.Error("failed to get profile", zap.Error(err))
			return nil, err
		}
		// If "profile not found", proceed to create
	}

	if profile != nil {
		return profile, nil
	}

	// 2. Create new profile
	log.Info("profile not found, creating profile")
	newProfile := &Profile{UserID: userID}

	createdProfile, err := s.repo.CreateProfile(ctx, newProfile)
	if err != nil {
		log.Error("failed to create profile", zap.Error(err))
		return nil, err
	}

	return createdProfile, nil
}

func (s *service) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*Profile, error) {
	log := logger.FromCtx(ctx).With(zap.Uint("user_id", params.UserID))

	// Construct profile object with fields to update
	p := &Profile{
		UserID:      params.UserID,
		FullName:    params.FullName,
		Bio:         params.Bio,
		AvatarURL:   params.AvatarURL,
		Phone:       params.Phone,
		DateOfBirth: params.DateOfBirth,
	}

	updatedProfile, err := s.repo.UpdateProfile(ctx, p)
	if err != nil {
		log.Error("failed to update profile", zap.Error(err))
		return nil, err
	}

	log.Info("profile updated successfully")
	return updatedProfile, nil
}
