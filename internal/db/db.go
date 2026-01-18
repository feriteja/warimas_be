package db

import (
	"database/sql"
	"fmt"
	"log"

	"warimas-be/internal/config"

	_ "github.com/lib/pq"
)

func InitDB(cfg *config.Config) *sql.DB {
	db, err := NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	log.Println("Database connection established")
	return db
}

// NewDatabase creates a new database connection.
// It returns an error instead of exiting, making it testable.
func NewDatabase(cfg *config.Config) (*sql.DB, error) {
	return newDatabaseWithDriver(cfg, "postgres")
}

func newDatabaseWithDriver(cfg *config.Config, driver string) (*sql.DB, error) {
	dsn := buildDSN(cfg)

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return db, nil
}

func buildDSN(cfg *config.Config) string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort,
	)
}
