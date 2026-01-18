package address

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "phone", "address_line1", "address_line2",
			"city", "province", "postal_code", "country", "is_default", "is_active", "receiver_name",
		}).AddRow(
			uuid.New(), userID, "Home", "123", "Street 1", nil,
			"City", "Prov", "12345", "ID", true, true, "John",
		)

		mock.ExpectQuery("SELECT .* FROM addresses WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(rows)

		res, err := repo.GetByUserID(context.Background(), userID)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Home", res[0].Name)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery("SELECT .* FROM addresses").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		res, err := repo.GetByUserID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "phone", "address_line1", "address_line2",
			"city", "province", "postal_code", "country", "is_default", "is_active", "receiver_name",
		}).AddRow(
			id, 1, "Home", "123", "Street 1", nil,
			"City", "Prov", "12345", "ID", true, true, "John",
		)

		mock.ExpectQuery("SELECT .* FROM addresses WHERE id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		res, err := repo.GetByID(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, id, res.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT .* FROM addresses WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		res, err := repo.GetByID(context.Background(), id)
		assert.Error(t, err)
		assert.Equal(t, "address not found", err.Error())
		assert.Nil(t, res)
	})
}

func TestRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	addr := &Address{
		ID:           uuid.New(),
		UserID:       1,
		Name:         "Office",
		Phone:        "08123",
		Address1:     "Jalan 1",
		City:         "Jakarta",
		Province:     "DKI",
		Postal:       "10110",
		Country:      "ID",
		IsDefault:    false,
		IsActive:     true,
		ReceiverName: "Doe",
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO addresses").
			WithArgs(
				addr.ID, addr.UserID, addr.Name, addr.Phone,
				addr.Address1, addr.Address2, addr.City, addr.Province,
				addr.Postal, addr.Country, addr.IsDefault, addr.IsActive, addr.ReceiverName,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), addr)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO addresses").
			WillReturnError(errors.New("insert failed"))

		err := repo.Create(context.Background(), addr)
		assert.Error(t, err)
	})
}

func TestRepository_Deactivate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("UPDATE addresses SET is_active = false").
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Deactivate(context.Background(), id)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec("UPDATE addresses SET is_active = false").
			WithArgs(id).
			WillReturnError(errors.New("db error"))
		err := repo.Deactivate(context.Background(), id)
		assert.Error(t, err)
	})
}

func TestRepository_ClearDefault(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("UPDATE addresses SET is_default = false").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.ClearDefault(context.Background(), userID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec("UPDATE addresses SET is_default = false").
			WithArgs(userID).
			WillReturnError(errors.New("db error"))
		err := repo.ClearDefault(context.Background(), userID)
		assert.Error(t, err)
	})
}

func TestRepository_SetDefault(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)
	addrID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("UPDATE addresses SET is_default = true").
			WithArgs(userID, addrID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.SetDefault(context.Background(), userID, addrID)
		assert.NoError(t, err)
	})

	t.Run("UpdateError", func(t *testing.T) {
		mock.ExpectExec("UPDATE addresses SET is_default = true").
			WithArgs(userID, addrID).
			WillReturnError(errors.New("db error"))

		err := repo.SetDefault(context.Background(), userID, addrID)
		assert.Error(t, err)
	})

	t.Run("AddressInactive", func(t *testing.T) {
		// 1. Update returns 0 rows
		mock.ExpectExec("UPDATE addresses SET is_default = true").
			WithArgs(userID, addrID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// 2. Check active returns false
		mock.ExpectQuery("SELECT is_active FROM addresses").
			WithArgs(userID, addrID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(false))

		err := repo.SetDefault(context.Background(), userID, addrID)
		assert.Error(t, err)
		assert.Equal(t, "cannot set default address: address is inactive", err.Error())
	})

	t.Run("AddressNotFound", func(t *testing.T) {
		// 1. Update returns 0 rows
		mock.ExpectExec("UPDATE addresses SET is_default = true").
			WithArgs(userID, addrID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// 2. Check active returns NoRows
		mock.ExpectQuery("SELECT is_active FROM addresses").
			WithArgs(userID, addrID).
			WillReturnError(sql.ErrNoRows)

		err := repo.SetDefault(context.Background(), userID, addrID)
		assert.Error(t, err)
		assert.Equal(t, "address not found", err.Error())
	})

	t.Run("GenericFailure", func(t *testing.T) {
		// 1. Update returns 0 rows
		mock.ExpectExec("UPDATE addresses SET is_default = true").
			WithArgs(userID, addrID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// 2. Check active returns true (so it should have updated, but didn't for some reason)
		mock.ExpectQuery("SELECT is_active FROM addresses").
			WithArgs(userID, addrID).
			WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

		err := repo.SetDefault(context.Background(), userID, addrID)
		assert.Error(t, err)
		assert.Equal(t, "failed to set default address", err.Error())
	})
}

func TestRepository_GetByIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ids := []uuid.UUID{uuid.New(), uuid.New()}

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "phone", "address_line1", "address_line2",
			"city", "province", "postal_code", "country", "is_default", "is_active", "receiver_name",
		}).AddRow(
			ids[0], 1, "Home", "123", "Street 1", nil,
			"City", "Prov", "12345", "ID", true, true, "John",
		).AddRow(
			ids[1], 1, "Work", "456", "Street 2", nil,
			"City", "Prov", "67890", "ID", false, true, "Doe",
		)

		// Expect ANY($1)
		mock.ExpectQuery("SELECT .* FROM addresses WHERE id = ANY").
			WithArgs(sqlmock.AnyArg()). // pq.Array is hard to match exactly with sqlmock default matcher
			WillReturnRows(rows)

		res, err := repo.GetByIDs(context.Background(), ids)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("EmptyIDs", func(t *testing.T) {
		res, err := repo.GetByIDs(context.Background(), []uuid.UUID{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery("SELECT .* FROM addresses WHERE id = ANY").
			WillReturnError(errors.New("db error"))

		res, err := repo.GetByIDs(context.Background(), ids)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
