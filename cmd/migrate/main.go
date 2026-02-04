package main

import (
	"Go_FormanceLegder/internal/config"
	"Go_FormanceLegder/internal/db"
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run SQL migrations first
	if err := runSQLMigrations(ctx, pool); err != nil {
		log.Fatalf("failed to run SQL migrations: %v", err)
	}

	// Then run River migrations
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		log.Fatalf("failed to create River migrator: %v", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		log.Fatalf("failed to run River migrations: %v", err)
	}

	// Create completion flag for healthcheck
	if err := os.WriteFile("/tmp/migration_complete", []byte("done"), 0644); err != nil {
		log.Printf("warning: failed to create migration flag: %v", err)
	}

	log.Println("All migrations completed successfully")
	log.Println("Migration service will keep running for healthcheck...")

	// Keep container running for healthcheck
	select {}
}

func runSQLMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Create migrations table if not exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	// Get migration files
	migrationsDir := "./migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	// Filter and sort up migration files
	var upMigrations []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			upMigrations = append(upMigrations, file.Name())
		}
	}
	sort.Strings(upMigrations)

	// Run each migration
	for _, fileName := range upMigrations {
		version := strings.TrimSuffix(fileName, ".up.sql")

		// Check if migration already applied
		var count int
		err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Read migration file
		content, err := os.ReadFile(filepath.Join(migrationsDir, fileName))
		if err != nil {
			return err
		}

		// Execute migration
		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return err
		}

		// Record migration
		_, err = pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			return err
		}

		log.Printf("Applied migration: %s", version)
	}

	return nil
}
