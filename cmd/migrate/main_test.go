package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMigrationPart(t *testing.T) {
	content := `
-- +migrate Up
CREATE TABLE users (id int);
ALTER TABLE users ADD COLUMN name text;

-- +migrate Down
DROP TABLE users;
`
	t.Run("Extract Up", func(t *testing.T) {
		up := extractMigrationPart(content, "Up")
		assert.Contains(t, up, "CREATE TABLE users")
		assert.Contains(t, up, "ALTER TABLE users")
		assert.NotContains(t, up, "DROP TABLE users")
		assert.NotContains(t, up, "-- +migrate Up") // Should not contain the marker itself
	})

	t.Run("Extract Down", func(t *testing.T) {
		down := extractMigrationPart(content, "Down")
		assert.Contains(t, down, "DROP TABLE users")
		assert.NotContains(t, down, "CREATE TABLE users")
	})
}

func TestSortStrings(t *testing.T) {
	// Test that it sorts filenames correctly
	files := []string{"20230201_b.sql", "20230101_a.sql", "20230301_c.sql"}
	sortStrings(files)

	expected := []string{"20230101_a.sql", "20230201_b.sql", "20230301_c.sql"}
	assert.Equal(t, expected, files)
}

func TestRunMigrationsUp(t *testing.T) {
	// 1. Mock Database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 2. Create a temporary migration file
	tmpDir := t.TempDir()
	fileName := "20230101_init.sql"
	filePath := filepath.Join(tmpDir, fileName)

	content := "-- +migrate Up\nCREATE TABLE test (id int);"
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	files := []string{filePath}

	// 3. Define Expectations
	// Check if migration exists (return false so it runs)
	mock.ExpectQuery("SELECT EXISTS.*schema_migrations").
		WithArgs(fileName).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Execute the SQL from the file
	mock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Record the migration version
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(fileName).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. Run the function
	runMigrationsUp(db, files)

	// 5. Verify
	require.NoError(t, mock.ExpectationsWereMet())
}
