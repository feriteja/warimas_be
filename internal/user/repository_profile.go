package user

import (
	"context"
	"database/sql"
	"warimas-be/internal/logger"

	"errors"

	"go.uber.org/zap"
)

// GetProfile fetches a user's profile by user ID.
func (r *repository) GetProfile(ctx context.Context, userID uint) (*Profile, error) {
	log := logger.FromCtx(ctx).With(zap.Uint("user_id", userID))

	query := `
		SELECT id, user_id, full_name, bio, avatar_url, phone, date_of_birth, created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`
	row := r.db.QueryRowContext(ctx, query, userID)

	var p Profile
	err := row.Scan(
		&p.ID, &p.UserID, &p.FullName, &p.Bio, &p.AvatarURL, &p.Phone, &p.DateOfBirth, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("profile not found")
			return nil, ErrProfileNotFound
		}
		log.Error("failed to scan profile", zap.Error(err))
		return nil, err
	}

	log.Info("profile fetched successfully")
	return &p, nil
}

// CreateProfile creates a new profile for a user.
func (r *repository) CreateProfile(ctx context.Context, p *Profile) (*Profile, error) {
	query := `
		INSERT INTO profiles (user_id, full_name, bio, avatar_url, phone, date_of_birth)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		p.UserID, p.FullName, p.Bio, p.AvatarURL, p.Phone, p.DateOfBirth,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		log := logger.FromCtx(ctx).With(zap.Uint("user_id", p.UserID))
		log.Error("failed to create profile", zap.Error(err))
		return nil, err
	}
	log := logger.FromCtx(ctx).With(zap.String("profile_id", p.ID.String()), zap.Uint("user_id", p.UserID))
	log.Info("profile created successfully")
	return p, nil
}

// UpdateProfile updates an existing profile.
func (r *repository) UpdateProfile(ctx context.Context, p *Profile) (*Profile, error) {
	// Using COALESCE to keep existing values if input is nil
	query := `
		UPDATE profiles
		SET full_name = COALESCE($2, full_name),
			bio = COALESCE($3, bio),
			avatar_url = COALESCE($4, avatar_url),
			phone = COALESCE($5, phone),
			date_of_birth = COALESCE($6, date_of_birth),
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING id, full_name, bio, avatar_url, phone, date_of_birth, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		p.UserID, p.FullName, p.Bio, p.AvatarURL, p.Phone, p.DateOfBirth,
	).Scan(
		&p.ID, &p.FullName, &p.Bio, &p.AvatarURL, &p.Phone, &p.DateOfBirth, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		log := logger.FromCtx(ctx).With(zap.Uint("user_id", p.UserID))
		log.Error("failed to update profile", zap.Error(err))
		return nil, err
	}

	return p, nil
}
