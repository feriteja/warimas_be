package main

import (
	"database/sql"
	"errors"
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
	err = runMigrationsUp(db, files)
	require.NoError(t, err)

	// 5. Verify
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrationsUp_Skip(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tmpDir := t.TempDir()
	fileName := "20230102_skip.sql"
	filePath := filepath.Join(tmpDir, fileName)
	// We create the file to simulate a real environment, though logic skips reading it if applied.
	err = os.WriteFile(filePath, []byte("-- +migrate Up\nSELECT 1;"), 0644)
	require.NoError(t, err)

	files := []string{filePath}

	mock.ExpectQuery("SELECT EXISTS.*schema_migrations").
		WithArgs(fileName).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	err = runMigrationsUp(db, files)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrationsDown_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tmpDir := t.TempDir()
	fileName := "20230103_down.sql"
	filePath := filepath.Join(tmpDir, fileName)

	content := "-- +migrate Down\nDROP TABLE test;"
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	files := []string{filePath}

	mock.ExpectQuery("SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(fileName))

	mock.ExpectExec("DROP TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("DELETE FROM schema_migrations WHERE version = \\$1").
		WithArgs(fileName).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = runMigrationsDown(db, files)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrationsDown_NoMigrations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1").
		WillReturnError(sql.ErrNoRows)

	err = runMigrationsDown(db, nil)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrationsUp_Errors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	files := []string{"test.sql"}

	t.Run("CheckStatusError", func(t *testing.T) {
		mock.ExpectQuery("SELECT EXISTS").WillReturnError(errors.New("db error"))
		err := runMigrationsUp(db, files)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check migration status")
	})

	t.Run("ExecError", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_exec.sql")
		_ = os.WriteFile(filePath, []byte("-- +migrate Up\nFAIL"), 0644)

		mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectExec("FAIL").WillReturnError(errors.New("exec error"))

		err := runMigrationsUp(db, []string{filePath})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Migration failed")
	})

	t.Run("ReadFileError", func(t *testing.T) {
		// File doesn't exist on disk, simulating a race condition or deletion
		missingFile := "nonexistent.sql"

		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(missingFile).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		err := runMigrationsUp(db, []string{missingFile})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("RecordVersionError", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "record_fail.sql")
		_ = os.WriteFile(filePath, []byte("-- +migrate Up\nSELECT 1;"), 0644)

		mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO schema_migrations").WillReturnError(errors.New("insert error"))

		err := runMigrationsUp(db, []string{filePath})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to record migration version")
	})
}

func TestRunMigrationsDown_Errors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	t.Run("GetVersionError", func(t *testing.T) {
		mock.ExpectQuery("SELECT version").WillReturnError(errors.New("db error"))
		err := runMigrationsDown(db, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get last applied migration")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT version").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("missing.sql"))
		err := runMigrationsDown(db, []string{"other.sql"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "migration file not found")
	})

	t.Run("ExecError", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "down_fail.sql")
		_ = os.WriteFile(filePath, []byte("-- +migrate Down\nFAIL"), 0644)

		mock.ExpectQuery("SELECT version").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("down_fail.sql"))
		mock.ExpectExec("FAIL").WillReturnError(errors.New("exec error"))

		err := runMigrationsDown(db, []string{filePath})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Rollback failed")
	})

	t.Run("RemoveRecordError", func(t *testing.T) {
		tmpDir := t.TempDir()
		fileName := "delete_fail.sql"
		filePath := filepath.Join(tmpDir, fileName)
		_ = os.WriteFile(filePath, []byte("-- +migrate Down\nSELECT 1;"), 0644)

		mock.ExpectQuery("SELECT version").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(fileName))
		mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("DELETE FROM schema_migrations").WillReturnError(errors.New("delete error"))

		err := runMigrationsDown(db, []string{filePath})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove migration record")
	})
}

func TestRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tmpDir := t.TempDir()
	// Create a dummy migration file
	fileName := "20230101_test.sql"
	err = os.WriteFile(filepath.Join(tmpDir, fileName), []byte("-- +migrate Up\nSELECT 1;"), 0644)
	require.NoError(t, err)

	// 1. Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 2. Expect migration check (runMigrationsUp logic)
	mock.ExpectQuery("SELECT EXISTS.*schema_migrations").
		WithArgs(fileName).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// 3. Expect migration execution
	mock.ExpectExec("SELECT 1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 4. Expect recording
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(fileName).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = run(db, "up", tmpDir)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRun_Errors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	t.Run("TableCreationFail", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
			WillReturnError(errors.New("create table error"))
		err := run(db, "up", ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ensure schema_migrations table")
	})

	t.Run("UnknownMode", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").WillReturnResult(sqlmock.NewResult(0, 0))
		err := run(db, "invalid", ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown mode")
	})

	t.Run("GlobError", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").WillReturnResult(sqlmock.NewResult(0, 0))
		// "[" is a syntax error in glob patterns if not closed
		err := run(db, "up", "[")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read migrations")
	})
}
