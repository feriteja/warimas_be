package packages

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetPackages(t *testing.T) {
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()
		viewerID := uint(1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT p.id) FROM packages p WHERE p.deleted_at IS NULL AND p.is_active = TRUE AND (p.type != 'personal' OR p.user_id = $1)")).
			WithArgs(viewerID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{
			"p.id", "p.name", "p.image_url", "p.user_id", "p.type", "p.created_at", "p.updated_at",
			"pi.id", "pi.variant_id", "pi.name", "pi.image_url", "pi.quantity", "pi.created_at", "pi.updated_at", "pi.price",
		}).AddRow(
			"pkg1", "Package 1", "img", 1, "personal", now, now,
			"item1", "v1", "Item 1", "img", 1, now, now, 100.0,
		)
		mock.ExpectQuery(`SELECT .* FROM packages p`).
			WillReturnRows(rows)

		pkgs, total, err := repo.GetPackages(ctx, nil, nil, 10, 1, false, &viewerID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, pkgs, 1)
		assert.Len(t, pkgs[0].Items, 1)
		assert.Equal(t, "pkg1", pkgs[0].ID)
		assert.Equal(t, "Package 1", pkgs[0].Name)
		assert.Equal(t, "img", *pkgs[0].ImageURL)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NoItems", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()
		viewerID := uint(1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT p.id) FROM packages p WHERE p.deleted_at IS NULL AND p.is_active = TRUE AND (p.type != 'personal' OR p.user_id = $1)")).
			WithArgs(viewerID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{
			"p.id", "p.name", "p.image_url", "p.user_id", "p.type", "p.created_at", "p.updated_at",
			"pi.id", "pi.variant_id", "pi.name", "pi.image_url", "pi.quantity", "pi.created_at", "pi.updated_at",
			"v.price",
		}).AddRow(
			"pkg1", "Package 1", "img", 1, "personal", now, now,
			sql.NullString{}, sql.NullString{}, sql.NullString{},
			sql.NullString{}, sql.NullInt32{}, sql.NullTime{}, sql.NullTime{}, sql.NullFloat64{},
		)

		mock.ExpectQuery(`SELECT .* FROM packages p`).
			WillReturnRows(rows)

		pkgs, total, err := repo.GetPackages(ctx, nil, nil, 10, 1, false, &viewerID)
		assert.NoError(t, err)
		assert.Len(t, pkgs, 1)
		assert.Len(t, pkgs[0].Items, 0)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "pkg1", pkgs[0].ID)
		assert.Equal(t, "Package 1", pkgs[0].Name)
		assert.Equal(t, "img", *pkgs[0].ImageURL)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DBErrorOnCount", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()
		viewerID := uint(1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT p.id) FROM packages p")).
			WillReturnError(errors.New("db error"))

		pkgs, total, err := repo.GetPackages(ctx, nil, nil, 10, 1, false, &viewerID)
		assert.Error(t, err)
		assert.Nil(t, pkgs)
		assert.Equal(t, int64(0), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DBErrorOnQuery", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()
		viewerID := uint(1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT p.id) FROM packages p")).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery(`SELECT .* FROM packages p`).
			WillReturnError(errors.New("db error"))

		pkgs, total, err := repo.GetPackages(ctx, nil, nil, 10, 1, false, &viewerID)
		assert.Error(t, err)
		assert.Nil(t, pkgs)
		assert.Equal(t, int64(0), total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_CreatePackage(t *testing.T) {
	userID := uint(1)
	input := CreatePackageInput{
		Name: "Test Package",
		Type: "personal",
		Items: []CreatePackageItemInput{
			{VariantID: "v1", Quantity: 2},
		},
	}

	t.Run("Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO packages").
			WithArgs(sqlmock.AnyArg(), input.Name, input.Type, userID, true, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery("SELECT name, imageurl, price FROM variants").
			WithArgs("v1").
			WillReturnRows(sqlmock.NewRows([]string{"name", "imageurl", "price"}).AddRow("Variant 1", "img.jpg", 150.0))

		mock.ExpectExec("INSERT INTO package_items").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "v1", "Variant 1", "img.jpg", int32(2), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		pkg, err := repo.CreatePackage(ctx, input, userID)
		assert.NoError(t, err)
		require.NotNil(t, pkg)
		assert.Equal(t, "Test Package", pkg.Name)
		assert.Len(t, pkg.Items, 1)
		assert.Equal(t, "v1", pkg.Items[0].VariantID)
		assert.Equal(t, float64(150.0), pkg.Items[0].Price)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("VariantNotFound", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO packages").
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery("SELECT name, imageurl, price FROM variants").
			WithArgs("v1").
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()

		_, err = repo.CreatePackage(ctx, input, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "variant not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CommitError", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO packages").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery("SELECT name, imageurl, price FROM variants").
			WillReturnRows(sqlmock.NewRows([]string{"name", "imageurl", "price"}).AddRow("Variant 1", "img.jpg", 150.0))
		mock.ExpectExec("INSERT INTO package_items").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		_, err = repo.CreatePackage(ctx, input, userID)
		assert.Error(t, err)
		assert.Equal(t, "commit failed", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
