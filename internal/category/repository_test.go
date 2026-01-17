package category

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateCategory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	name := "Electronics"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name"}).AddRow("cat-1", name)

		// Expect INSERT returning ID and Name
		mock.ExpectQuery("INSERT INTO category").
			WithArgs(name).
			WillReturnRows(rows)

		res, err := repo.AddCategory(context.Background(), name)
		assert.NoError(t, err)
		assert.Equal(t, "cat-1", res.ID)
		assert.Equal(t, name, res.Name)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO category").WillReturnError(errors.New("db error"))
		_, err := repo.AddCategory(context.Background(), name)
		assert.Error(t, err)
	})
}

func TestRepository_CreateSubcategory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	catID := "cat-1"
	name := "Laptops"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "category_id", "name"}).AddRow("sub-1", catID, name)

		mock.ExpectQuery("INSERT INTO subcategories").
			WithArgs(catID, name).
			WillReturnRows(rows)

		res, err := repo.AddSubcategory(context.Background(), catID, name)
		assert.NoError(t, err)
		assert.Equal(t, "sub-1", res.ID)
		assert.Equal(t, catID, res.CategoryID)
	})
}

func TestRepository_GetCategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	t.Run("Success_NoFilter", func(t *testing.T) {
		limit := int32(10)
		page := int32(1)

		// 1. Count Query
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM category c").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		// 2. Data Query
		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow("cat-1", "A").
			AddRow("cat-2", "B")

		mock.ExpectQuery("SELECT .* FROM category c ORDER BY c.name ASC LIMIT \\$1 OFFSET \\$2").
			WithArgs(limit, 0). // Limit, Offset (page 1 = offset 0)
			WillReturnRows(rows)

		res, total, err := repo.GetCategories(context.Background(), nil, &limit, &page)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, int64(2), total)
	})

	t.Run("Success_WithFilter", func(t *testing.T) {
		filter := "elec"
		limit := int32(10)
		page := int32(1)

		// 1. Count Query
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM category c WHERE c.name ILIKE \\$1").WithArgs("%elec%").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// 2. Data Query
		rows := sqlmock.NewRows([]string{"id", "name"}).AddRow("cat-1", "Electronics")

		mock.ExpectQuery("SELECT .* FROM category c WHERE c.name ILIKE \\$1 ORDER BY c.name ASC LIMIT \\$2 OFFSET \\$3").
			WithArgs("%elec%", limit, 0).
			WillReturnRows(rows)

		res, total, err := repo.GetCategories(context.Background(), &filter, &limit, &page)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, int64(1), total)
	})
}

func TestRepository_GetSubcategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	catID := "cat-1"
	limit := int32(10)
	page := int32(1)

	t.Run("Success", func(t *testing.T) {
		// 1. Count
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM subcategories s WHERE s.category_id = \\$1").WithArgs(catID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// 2. Data
		rows := sqlmock.NewRows([]string{"id", "category_id", "name"}).
			AddRow("sub-1", catID, "Sub A")

		mock.ExpectQuery("SELECT s.id, s.category_id, s.name FROM subcategories s WHERE s.category_id = \\$1 ORDER BY s.name ASC LIMIT \\$2 OFFSET \\$3").
			WithArgs(catID, limit, 0).
			WillReturnRows(rows)

		res, total, err := repo.GetSubcategories(context.Background(), catID, nil, &limit, &page)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, int64(1), total)
	})

	t.Run("Success_WithFilter", func(t *testing.T) {
		filter := "test"

		// 1. Count
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM subcategories s WHERE s.category_id = \\$1 AND s.name ILIKE \\$2").WithArgs(catID, "%test%").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// 2. Data
		rows := sqlmock.NewRows([]string{"id", "category_id", "name"})

		mock.ExpectQuery("SELECT s.id, s.category_id, s.name FROM subcategories s WHERE s.category_id = \\$1 AND s.name ILIKE \\$2 ORDER BY s.name ASC LIMIT \\$3 OFFSET \\$4").
			WithArgs(catID, "%test%", limit, 0).
			WillReturnRows(rows)

		res, total, err := repo.GetSubcategories(context.Background(), catID, &filter, &limit, &page)
		assert.NoError(t, err)
		assert.Empty(t, res)
		assert.Equal(t, int64(0), total)
	})
}

func TestRepository_GetSubcategoriesByIds(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	catIDs := []string{"cat-1", "cat-2"}

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "category_id", "name"}).
			AddRow("sub-1", "cat-1", "Sub 1").
			AddRow("sub-2", "cat-1", "Sub 2").
			AddRow("sub-3", "cat-2", "Sub 3")

		// The query uses IN ($1, $2)
		mock.ExpectQuery("SELECT id, category_id, name FROM subcategories WHERE category_id IN \\(\\$1,\\$2\\)").
			WithArgs("cat-1", "cat-2").
			WillReturnRows(rows)

		res, err := repo.GetSubcategoriesByIds(context.Background(), catIDs)
		assert.NoError(t, err)
		assert.Len(t, res, 2) // 2 keys (cat-1, cat-2)
		assert.Len(t, res["cat-1"], 2)
		assert.Len(t, res["cat-2"], 1)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		res, err := repo.GetSubcategoriesByIds(context.Background(), []string{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
