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

	// Ensure schema_migrations table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Fatalf("failed to ensure schema_migrations table: %v", err)
	}

	migrationsDir := "./migrations"
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("failed to read migrations: %v", err)
	}

	// Sort for deterministic order
	sortStrings(files)

	switch *mode {
	case "up":
		runMigrationsUp(db, files)
	case "down":
		runMigrationsDown(db, files)
	default:
		log.Fatalf("unknown mode: %s (use 'up' or 'down')", *mode)
	}
}

func runMigrationsUp(db *sql.DB, files []string) {
	for _, file := range files {
		version := filepath.Base(file)

		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
		if err != nil {
			log.Fatalf("failed to check migration status: %v", err)
		}
		if exists {
			fmt.Printf("⏭ Skipping already applied migration: %s\n", version)
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read %s: %v", file, err)
		}

		upSQL := extractMigrationPart(string(content), "Up")
		fmt.Printf("🚀 Applying migration: %s\n", version)

		if _, err := db.Exec(upSQL); err != nil {
			log.Fatalf("❌ Migration failed (%s): %v", version, err)
		}

		_, err = db.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version)
		if err != nil {
			log.Fatalf("failed to record migration version: %v", err)
		}
	}
	fmt.Println("✅ All new migrations applied successfully.")
}

func runMigrationsDown(db *sql.DB, files []string) {
	// Find the latest applied migration
	var lastVersion string
	err := db.QueryRow(`SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`).Scan(&lastVersion)
	if err == sql.ErrNoRows {
		fmt.Println("⚠️  No migrations to roll back.")
		return
	}
	if err != nil {
		log.Fatalf("failed to get last applied migration: %v", err)
	}

	filePath := ""
	for _, f := range files {
		if filepath.Base(f) == lastVersion {
			filePath = f
			break
		}
	}
	if filePath == "" {
		log.Fatalf("migration file not found for version: %s", lastVersion)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", filePath, err)
	}

	downSQL := extractMigrationPart(string(content), "Down")
	fmt.Printf("🧹 Rolling back migration: %s\n", lastVersion)

	if _, err := db.Exec(downSQL); err != nil {
		log.Fatalf("❌ Rollback failed (%s): %v", filePath, err)
	}

	_, err = db.Exec(`DELETE FROM schema_migrations WHERE version = $1`, lastVersion)
	if err != nil {
		log.Fatalf("failed to remove migration record: %v", err)
	}

	fmt.Println("✅ Rollback successful.")
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
