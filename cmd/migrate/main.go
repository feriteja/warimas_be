package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	_ = godotenv.Load()

	mode := flag.String("mode", "up", "migration mode: up or down")
	flag.Parse()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL not set in environment")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()

	if err := run(db, *mode, "./migrations"); err != nil {
		log.Fatal(err)
	}
}

func run(db *sql.DB, mode, migrationsDir string) error {
	// Ensure schema_migrations table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort for deterministic order
	sortStrings(files)

	switch mode {
	case "up":
		return runMigrationsUp(db, files)
	case "down":
		return runMigrationsDown(db, files)
	default:
		return fmt.Errorf("unknown mode: %s (use 'up' or 'down')", mode)
	}
}

func runMigrationsUp(db *sql.DB, files []string) error {
	for _, file := range files {
		version := filepath.Base(file)

		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}
		if exists {
			fmt.Printf("⏭ Skipping already applied migration: %s\n", version)
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		upSQL := extractMigrationPart(string(content), "Up")
		fmt.Printf("🚀 Applying migration: %s\n", version)

		if _, err := db.Exec(upSQL); err != nil {
			return fmt.Errorf("❌ Migration failed (%s): %w", version, err)
		}

		_, err = db.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version)
		if err != nil {
			return fmt.Errorf("failed to record migration version: %w", err)
		}
	}
	fmt.Println("✅ All new migrations applied successfully.")
	return nil
}

func runMigrationsDown(db *sql.DB, files []string) error {
	// Find the latest applied migration
	var lastVersion string
	err := db.QueryRow(`SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`).Scan(&lastVersion)
	if err == sql.ErrNoRows {
		fmt.Println("⚠️  No migrations to roll back.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get last applied migration: %w", err)
	}

	filePath := ""
	for _, f := range files {
		if filepath.Base(f) == lastVersion {
			filePath = f
			break
		}
	}
	if filePath == "" {
		return fmt.Errorf("migration file not found for version: %s", lastVersion)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	downSQL := extractMigrationPart(string(content), "Down")
	fmt.Printf("🧹 Rolling back migration: %s\n", lastVersion)

	if _, err := db.Exec(downSQL); err != nil {
		return fmt.Errorf("❌ Rollback failed (%s): %w", filePath, err)
	}

	_, err = db.Exec(`DELETE FROM schema_migrations WHERE version = $1`, lastVersion)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	fmt.Println("✅ Rollback successful.")
	return nil
}

func extractMigrationPart(content string, section string) string {
	lines := strings.Split(content, "\n")
	var part strings.Builder
	var inPart bool

	for _, line := range lines {
		if strings.Contains(line, "-- +migrate "+section) {
			inPart = true
			continue
		}
		if inPart && strings.HasPrefix(line, "-- +migrate") {
			break
		}
		if inPart {
			part.WriteString(line + "\n")
		}
	}
	return part.String()
}

func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
