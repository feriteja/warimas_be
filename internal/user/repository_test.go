package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
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

	t.Run("ScanError", func(t *testing.T) {
		// Return fewer columns than expected to trigger Scan error
		mock.ExpectQuery(`INSERT INTO users`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

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

	t.Run("ScanError", func(t *testing.T) {
		// Return fewer columns than expected
		mock.ExpectQuery(`SELECT .* FROM users`).
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

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

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET password = \\$1 WHERE email = \\$2").
			WithArgs(newPassword, email).
			WillReturnError(errors.New("exec error"))

		err := repo.UpdatePassword(ctx, email, newPassword)
		assert.Error(t, err)
		assert.Equal(t, "exec error", err.Error())
	})
}

func TestRepository_GetProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "full_name", "bio", "avatar_url", "phone", "date_of_birth", "created_at", "updated_at", "email",
		}).AddRow(
			uuid.New(), userID, "John Doe", "Bio", "http://avatar", "123456", time.Now(), time.Now(), time.Now(), "test@example.com",
		)

		mock.ExpectQuery(`SELECT p.id, p.user_id, p.full_name, p.bio, p.avatar_url, p.phone, p.date_of_birth, p.created_at, p.updated_at, u.email FROM profiles p INNER JOIN users u ON p.user_id = u.id WHERE p.user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(rows)

		p, err := repo.GetProfile(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, userID, p.UserID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM profiles`).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetProfile(ctx, userID)
		assert.Error(t, err)
		assert.Equal(t, ErrProfileNotFound, err)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM profiles`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		_, err := repo.GetProfile(ctx, userID)
		assert.Error(t, err)
	})
}

func TestRepository_CreateProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uint(1)
	profile := &Profile{UserID: userID}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO profiles \(user_id, full_name, bio, avatar_url, phone, date_of_birth\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) RETURNING id, created_at, updated_at`).
			WithArgs(userID, nil, nil, nil, nil, nil).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))

		p, err := repo.CreateProfile(ctx, profile)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO profiles`).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateProfile(ctx, profile)
		assert.Error(t, err)
	})
}

func TestRepository_UpdateProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	userID := uint(1)
	name := "Updated Name"
	profile := &Profile{UserID: userID, FullName: &name}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE profiles SET full_name = COALESCE\(\$2, full_name\), .* WHERE user_id = \$1 RETURNING id, full_name, bio, avatar_url, phone, date_of_birth, created_at, updated_at`).
			WithArgs(userID, &name, nil, nil, nil, nil).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "full_name", "bio", "avatar_url", "phone", "date_of_birth", "created_at", "updated_at",
			}).AddRow(
				uuid.New(), name, nil, nil, nil, nil, time.Now(), time.Now(),
			))

		p, err := repo.UpdateProfile(ctx, profile)
		assert.NoError(t, err)
		assert.Equal(t, name, *p.FullName)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE profiles`).
			WillReturnError(errors.New("db error"))

		_, err := repo.UpdateProfile(ctx, profile)
		assert.Error(t, err)
	})
}
