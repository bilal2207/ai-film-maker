package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/database"
	"github.com/ai-filmmaker/api/internal/database/migrations"
	"github.com/ai-filmmaker/api/internal/server"
)

func main() {
	cfg := config.Load()

	var db *sql.DB
	var err error

	dbConfig := database.Config{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
	}

	log.Printf("Connecting to PostgreSQL at %s...", cfg.DatabaseURL)
	db, err = database.Connect(dbConfig)
	if err != nil {
		log.Printf("Warning: Database connection could not be established: %v", err)
		log.Println("Starting server without active database connection (health endpoint available).")
	} else {
		defer db.Close()
		log.Println("PostgreSQL connection established.")

		// Run database migrations
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := migrations.Run(ctx, db); err != nil {
			log.Fatalf("Failed to execute database migrations: %v", err)
		}
		log.Println("Database migrations up to date.")
	}

	srv := server.New(cfg, db)

	log.Printf("Starting AI Filmmaker API server on port %s (env: %s)...", cfg.Port, cfg.Environment)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
