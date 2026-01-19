package user

import (
	"context"
	"database/sql"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

type Repository interface {
	Create(ctx context.Context, email, password, role string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	UpdatePassword(ctx context.Context, email, password string) error
	GetProfile(ctx context.Context, userID uint) (*Profile, error)
	CreateProfile(ctx context.Context, p *Profile) (*Profile, error)
	UpdateProfile(ctx context.Context, p *Profile) (*Profile, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, email, password, role string) (*User, error) {
	log := logger.FromCtx(ctx)

	var u User
	err := r.db.QueryRowContext(ctx,
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, email, password, role",
		email, password,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role)

	if err != nil {
		log.Error("db: failed to insert user",
			zap.String("email", email),
			zap.Error(err),
		)
	}

	return &u, err
}

func (r *repository) UpdatePassword(ctx context.Context, email, password string) error {
	log := logger.FromCtx(ctx).With(zap.String("email", email))

	result, err := r.db.ExecContext(ctx, "UPDATE users SET password = $1 WHERE email = $2", password, email)
	if err != nil {
		log.Error("db: failed to update password", zap.Error(err))
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Error("db: failed to get rows affected", zap.Error(err))
		return err
	}
	if rows == 0 {
		log.Warn("db: no user found to update password")
		return sql.ErrNoRows
	}

	log.Info("db: password updated successfully")
	return nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	log := logger.FromCtx(ctx).With(zap.String("email", email))

	var u User
	err := r.db.QueryRowContext(ctx,
		"SELECT u.id, u.email, u.password, u.role, s.id FROM users u LEFT JOIN sellers s ON u.id = s.user_id WHERE u.email=$1",
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.SellerID)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("db: user not found")
		} else {
			log.Error("db: failed to find user", zap.Error(err))
		}
	}

	return &u, err
}
