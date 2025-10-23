package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()

	migrationsDir := "./migrations"

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("failed to read migrations: %v", err)
	}

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read %s: %v", file, err)
		}

		sqlStatements := parseMigration(string(content))
		fmt.Printf("Running migration: %s\n", filepath.Base(file))

		if _, err := db.Exec(sqlStatements); err != nil {
			log.Fatalf("migration failed (%s): %v", file, err)
		}
	}

	fmt.Println("✅ All migrations applied successfully.")
}

func parseMigration(content string) string {
	lines := strings.Split(content, "\n")
	var upPart strings.Builder
	var inUp bool

	for _, line := range lines {
		if strings.Contains(line, "-- +migrate Up") {
			inUp = true
			continue
		}
		if strings.Contains(line, "-- +migrate Down") {
			inUp = false
			break
		}
		if inUp {
			upPart.WriteString(line + "\n")
		}
	}
	return upPart.String()
}
