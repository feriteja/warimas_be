package user

import (
	"context"
	"database/sql"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

type Repository interface {
	Create(ctx context.Context, email, password, role string) (User, error)
	FindByEmail(email string) (User, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, email, password, role string) (User, error) {
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

	return u, err
}

func (r *repository) FindByEmail(email string) (User, error) {
	var u User
	err := r.db.QueryRow(
		"SELECT u.id, u.email, u.password, u.role, s.id FROM users u JOIN sellers s ON u.id = s.user_id WHERE u.email=$1",
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.SellerID)

	return u, err
}
