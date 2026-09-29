package main

import (
	"context"
	"log"
	"time"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/database"
	"github.com/ai-filmmaker/api/internal/database/migrations"
	"github.com/ai-filmmaker/api/internal/server"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Connect to PostgreSQL (Required Dependency)
	dbConfig := database.Config{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
	}

	log.Printf("Connecting to PostgreSQL at %s...", cfg.DatabaseURL)
	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()
	log.Println("PostgreSQL connection established.")

	// 3. Run database migrations (Required Dependency)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrations.Run(ctx, db); err != nil {
		log.Fatalf("Fatal: Failed to execute database migrations: %v", err)
	}
	log.Println("Database migrations up to date.")

	// 4. Start HTTP Server only after successful DB connection & migrations
	srv := server.New(cfg, db)

	log.Printf("Starting AI Filmmaker API server on port %s (env: %s)...", cfg.Port, cfg.Environment)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
