package product

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"warimas-be/internal/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetProductsByGroup(t *testing.T) {
	ctx := context.Background()
	opts := ProductQueryOptions{Limit: 10, Page: 1}

	t.Run("Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)

		rows := sqlmock.NewRows([]string{
			"category_id", "category_name", "subcategory_id", "subcategory_name", "total_products",
			"product_id", "product_name", "seller_id", "slug", "status",
			"variant_id", "variant_product_id", "variant_name", "variant_price", "stock", "imageurl", "quantity_type",
			"seller_name",
		}).AddRow(
			"cat1", "Category 1", "sub1", "Sub 1", 5,
			"p1", "Product 1", "s1", "slug-1", "active",
			"v1", "p1", "Var 1", 100.0, 10, "img.jpg", "pcs",
			"Seller A",
		)

		// The query is complex, matching via regex
		mock.ExpectQuery(`(?s)SELECT .* FROM category c .* LIMIT \$1 OFFSET \$2`).
			WithArgs(10, 0).
			WillReturnRows(rows)

		res, err := repo.GetProductsByGroup(ctx, opts)
		assert.NoError(t, err)
		if assert.Len(t, res, 1) {
			assert.Equal(t, "Category 1", res[0].CategoryName)
			assert.Len(t, res[0].Products, 1)
			assert.Equal(t, "Product 1", res[0].Products[0].Name)
		}
	})

	t.Run("QueryError", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)

		mock.ExpectQuery(`(?s)SELECT .*`).WillReturnError(errors.New("db error"))
		_, err = repo.GetProductsByGroup(ctx, opts)
		assert.Error(t, err)
	})

	t.Run("WithFilters", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepository(db)

		search := "phone"
		minP := 100.0
		maxP := 200.0
		inStock := true
		catID := "c1"
		catSlug := "slug"
		sellerName := "seller"
		status := "active"

		optsFiltered := ProductQueryOptions{
			Search:       &search,
			MinPrice:     &minP,
			MaxPrice:     &maxP,
			InStock:      &inStock,
			CategoryID:   &catID,
			CategorySlug: &catSlug,
			SellerName:   &sellerName,
			Status:       &status,
		}

		mock.ExpectQuery(`(?s)SELECT .* FROM category c\s+WHERE c.id = \$6 AND c.slug = \$7`).
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, err = repo.GetProductsByGroup(ctx, optsFiltered)
		assert.NoError(t, err)
	})
}

func TestRepository_GetList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("Success_WithCount", func(t *testing.T) {
		opts := ProductQueryOptions{
			Limit:        10,
			Page:         1,
			IncludeCount: true,
			OnlyActive:   true,
		}

		// Count Query
		mock.ExpectQuery(`SELECT COUNT\(DISTINCT p.id\) FROM products p`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(20))

		// Data Query
		rows := sqlmock.NewRows([]string{
			"id", "name", "seller_id", "seller_name", "status", "category_id", "subcategory_id",
			"slug", "imageurl", "description", "created_at", "updated_at",
			"category_name", "subcategory_name", "variants",
		}).AddRow(
			"p1", "Product 1", "s1", "Seller A", "active", "c1", "sub1",
			"slug-1", "img", "desc", time.Now(), nil,
			"Cat 1", "Sub 1", `[{"id":"v1", "price": 100}]`,
		)

		mock.ExpectQuery(`(?s)SELECT .* FROM products p .* LIMIT \$1 OFFSET \$2`).
			WithArgs(10, 0).
			WillReturnRows(rows)

		products, total, err := repo.GetList(ctx, opts)
		assert.NoError(t, err)
		assert.NotNil(t, total)
		assert.Equal(t, 20, *total)
		assert.Len(t, products, 1)
		assert.Len(t, products[0].Variants, 1)
	})

	t.Run("WithFilters_AndHaving", func(t *testing.T) {
		minP := 10.0
		opts := ProductQueryOptions{
			MinPrice: &minP,
		}

		// Data Query with HAVING
		mock.ExpectQuery(`(?s)SELECT .* HAVING MIN\(v.price\) >= \$1 .*`).
			WithArgs(minP, 20, 0). // Limit/Offset defaults
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, _, err := repo.GetList(ctx, opts)
		assert.NoError(t, err)
	})

	t.Run("JSONUnmarshalError", func(t *testing.T) {
		// Test the branch where variants JSON is invalid
		opts := ProductQueryOptions{Limit: 10, Page: 1}
		rows := sqlmock.NewRows([]string{
			"id", "name", "seller_id", "seller_name", "status", "category_id", "subcategory_id",
			"slug", "imageurl", "description", "created_at", "updated_at",
			"category_name", "subcategory_name", "variants",
		}).AddRow(
			"p1", "Product 1", "s1", "Seller A", "active", "c1", "sub1",
			"slug-1", "img", "desc", time.Now(), nil,
			"Cat 1", "Sub 1", `invalid-json`, // <--- Invalid JSON
		)

		mock.ExpectQuery(`(?s)SELECT .*`).WillReturnRows(rows)

		products, _, err := repo.GetList(ctx, opts)
		assert.NoError(t, err)
		assert.Len(t, products, 1)
		assert.Empty(t, products[0].Variants) // Should default to empty slice
	})
}

func TestRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	sellerID := "s1"
	input := NewProductInput{
		Name:          "Prod 1",
		CategoryID:    "c1",
		SubcategoryID: "sub1",
		ImageURL:      utils.StrPtr("img"),
		Description:   utils.StrPtr("desc"),
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO products`).
			WithArgs(input.CategoryID, sellerID, input.Name, sqlmock.AnyArg(), input.ImageURL, input.SubcategoryID, input.Description).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "imageurl"}).AddRow("p1", "Prod 1", "img"))

		p, err := repo.Create(ctx, input, sellerID)
		assert.NoError(t, err)
		assert.Equal(t, "p1", p.ID)
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Missing SellerID
		_, err := repo.Create(ctx, input, "")
		assert.Error(t, err)
	})

	t.Run("MissingFields", func(t *testing.T) {
		// Missing Name
		_, err := repo.Create(ctx, NewProductInput{CategoryID: "c1", SubcategoryID: "s1"}, sellerID)
		assert.Error(t, err)
		assert.Equal(t, "name is required", err.Error())

		// Missing CategoryID
		_, err = repo.Create(ctx, NewProductInput{Name: "P1", SubcategoryID: "s1"}, sellerID)
		assert.Error(t, err)
		assert.Equal(t, "categoryID is required", err.Error())

		// Missing SubcategoryID
		_, err = repo.Create(ctx, NewProductInput{Name: "P1", CategoryID: "c1"}, sellerID)
		assert.Error(t, err)
		assert.Equal(t, "subcategoryID is required", err.Error())
	})
}

func TestRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	sellerID := "s1"
	name := "New Name"
	input := UpdateProductInput{ID: "p1", Name: &name}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE products SET name = \$1, slug = \$2 WHERE id = \$3 AND seller_id = \$4 RETURNING`).
			WithArgs(name, sqlmock.AnyArg(), input.ID, sellerID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "imageurl", "description", "category_id", "seller_id", "subcategory_id", "status",
			}).AddRow("p1", name, "img", "desc", "c1", "s1", "sub1", "active"))

		p, err := repo.Update(ctx, input, sellerID)
		assert.NoError(t, err)
		assert.Equal(t, name, p.Name)
	})

	t.Run("NoFields", func(t *testing.T) {
		_, err := repo.Update(ctx, UpdateProductInput{ID: "p1"}, sellerID)
		assert.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE products`).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.Update(ctx, input, sellerID)
		assert.Error(t, err)
		assert.Equal(t, "product not found or not owned by seller", err.Error())
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE products`).
			WillReturnError(errors.New("db error"))

		_, err := repo.Update(ctx, input, sellerID)
		assert.Error(t, err)
		assert.NotEqual(t, "product not found or not owned by seller", err.Error())
	})
}

func TestRepository_BulkCreateVariants(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	sellerID := "s1"
	input := []*NewVariantInput{
		{ProductID: "p1", Name: "V1", Price: 100},
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO variants`).
			WithArgs(input[0].ProductID, input[0].Name, input[0].QuantityType, input[0].Price, input[0].Stock, input[0].ImageURL, input[0].Description).
			WillReturnRows(sqlmock.NewRows([]string{"id", "product_id", "name", "quantity_type", "price", "stock", "imageurl", "created_at"}).
				AddRow("v1", "p1", "V1", "pcs", 100.0, 10, "img", time.Now()))

		vars, err := repo.BulkCreateVariants(ctx, input, sellerID)
		assert.NoError(t, err)
		assert.Len(t, vars, 1)
	})

	t.Run("TooMany", func(t *testing.T) {
		many := make([]*NewVariantInput, 101)
		_, err := repo.BulkCreateVariants(ctx, many, sellerID)
		assert.Error(t, err)
	})
}

func TestRepository_BulkUpdateVariants(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	sellerID := "s1"
	name := "V1 New"
	input := []*UpdateVariantInput{
		{ID: "v1", ProductID: "p1", Name: &name},
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`UPDATE variants SET name = \$1 WHERE id = \$2 AND product_id = \$3 AND product_id IN`).
			WithArgs(name, input[0].ID, input[0].ProductID, sellerID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "product_id", "name", "price", "stock", "imageurl", "description"}).
				AddRow("v1", "p1", name, 100.0, 10, "img", "desc"))
		mock.ExpectCommit()

		vars, err := repo.BulkUpdateVariants(ctx, input, sellerID)
		assert.NoError(t, err)
		assert.Len(t, vars, 1)
	})

	t.Run("TxBeginError", func(t *testing.T) {
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))
		_, err := repo.BulkUpdateVariants(ctx, input, sellerID)
		assert.Error(t, err)
	})

	t.Run("SkipEmptyUpdate", func(t *testing.T) {
		// Input with no fields to update
		emptyInput := []*UpdateVariantInput{{ID: "v1", ProductID: "p1"}}

		mock.ExpectBegin()
		mock.ExpectCommit() // Should commit immediately without queries

		vars, err := repo.BulkUpdateVariants(ctx, emptyInput, sellerID)
		assert.NoError(t, err)
		assert.Len(t, vars, 0)
	})
}

func TestRepository_GetPackages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "image_url", "user_id",
			"item_id", "variant_id", "item_name", "item_image_url", "variant_price", "quantity", "created_at", "updated_at",
		}).AddRow(
			"pkg1", "Package 1", "img", "u1",
			"item1", "v1", "Item 1", "img", 100.0, 1, time.Now(), time.Now(),
		)

		mock.ExpectQuery(`(?s)SELECT .* FROM packages p .*`).
			WillReturnRows(rows)

		pkgs, err := repo.GetPackages(ctx, nil, nil, 10, 1, false)
		assert.NoError(t, err)
		assert.Len(t, pkgs, 1)
		assert.Len(t, pkgs[0].Items, 1)
	})
}

func TestRepository_GetProductByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	pID := "p1"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "seller_id", "category_id", "subcategory_id", "slug", "imageurl", "description", "created_at",
			"category_name", "subcategory_name", "seller_name", "variants",
		}).AddRow(
			pID, "Prod 1", "s1", "c1", "sub1", "slug", "img", "desc", time.Now(),
			"Cat 1", "Sub 1", "Seller A", `[]`,
		)

		mock.ExpectQuery(`(?s)SELECT .* FROM products p .* WHERE p.id = \$1`).
			WithArgs(pID).
			WillReturnRows(rows)

		p, err := repo.GetProductByID(ctx, GetProductOptions{ProductID: pID})
		assert.NoError(t, err)
		assert.Equal(t, pID, p.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT .* FROM products p .* WHERE p.id = \$1`).
			WithArgs(pID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetProductByID(ctx, GetProductOptions{ProductID: pID})
		assert.Error(t, err)
		assert.Equal(t, ErrProductNotFound, err)
	})
}

func TestRepository_GetProductVariantByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	vID := "v1"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "product_id", "quantity_type", "price", "stock", "imageurl", "category_id", "seller_id", "created_at", "description",
		}).AddRow(
			vID, "V1", "p1", "pcs", 100.0, 10, "img", "c1", "s1", time.Now(), "desc",
		)

		mock.ExpectQuery(`(?s)SELECT .* FROM variants v .* WHERE v.id = \$1`).
			WithArgs(vID).
			WillReturnRows(rows)

		v, err := repo.GetProductVariantByID(ctx, GetVariantOptions{VariantID: vID})
		assert.NoError(t, err)
		assert.Equal(t, vID, v.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT .* FROM variants v`).
			WithArgs(vID).
			WillReturnError(sql.ErrNoRows)

		v, err := repo.GetProductVariantByID(ctx, GetVariantOptions{VariantID: vID})
		assert.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT .* FROM variants v`).
			WithArgs(vID).
			WillReturnError(errors.New("db error"))

		_, err := repo.GetProductVariantByID(ctx, GetVariantOptions{VariantID: vID})
		assert.Error(t, err)
	})
}
