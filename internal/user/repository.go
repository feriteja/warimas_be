package user

import (
	"database/sql"
)

type Repository interface {
	Create(email, password, role string) (User, error)
	FindByEmail(email string) (User, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(email, password, role string) (User, error) {
	var u User
	err := r.db.QueryRow(
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, email, password",
		email, password,
	).Scan(&u.ID, &u.Email, &u.Password)
	return u, err
}

func (r *repository) FindByEmail(email string) (User, error) {
	var u User
	err := r.db.QueryRow(
		"SELECT id, email, password, role FROM users WHERE email=$1",
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role)

	return u, err
}
