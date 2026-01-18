package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	email := "john@example.com"
	password := "hashed_password"
	role := "USER"

	t.Run("Success", func(t *testing.T) {
		// Matches: INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, email, password, role
		mock.ExpectQuery(`INSERT INTO users \(email, password\) VALUES \(\$1, \$2\) RETURNING id, email, password, role`).
			WithArgs(email, password).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).
				AddRow(1, email, password, role))

		u, err := repo.Create(ctx, email, password, role)
		assert.NoError(t, err)
		assert.Equal(t, 1, u.ID)
		assert.Equal(t, email, u.Email)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO users`).
			WillReturnError(errors.New("db error"))

		_, err := repo.Create(ctx, email, password, role)
		assert.Error(t, err)
	})
}

func TestRepository_FindByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	email := "john@example.com"

	t.Run("Success", func(t *testing.T) {
		// Matches: SELECT u.id, u.email, u.password, u.role, s.id FROM users u LEFT JOIN sellers s ...
		rows := sqlmock.NewRows([]string{"id", "email", "password", "role", "seller_id"}).
			AddRow(1, email, "hashed", "USER", nil)

		mock.ExpectQuery(`SELECT u.id, u.email, u.password, u.role, s.id FROM users u LEFT JOIN sellers s ON u.id = s.user_id WHERE u.email=\$1`).
			WithArgs(email).
			WillReturnRows(rows)

		u, err := repo.FindByEmail(ctx, email)
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, email, u.Email)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM users`).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		u, err := repo.FindByEmail(ctx, email)
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.NotNil(t, u) // Implementation returns &u even on error
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM users`).
			WithArgs(email).
			WillReturnError(errors.New("connection refused"))

		_, err := repo.FindByEmail(ctx, email)
		assert.Error(t, err)
	})
}

func TestRepository_UpdatePassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	email := "john@example.com"
	newPassword := "new_hashed_password"

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET password = \\$1 WHERE email = \\$2").
			WithArgs(newPassword, email).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdatePassword(ctx, email, newPassword)
		assert.NoError(t, err)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET password = \\$1 WHERE email = \\$2").
			WithArgs(newPassword, email).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.UpdatePassword(ctx, email, newPassword)
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)
	})
}
