package packages

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
