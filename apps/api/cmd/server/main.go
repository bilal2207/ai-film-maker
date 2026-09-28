package main

import (
	"log"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/server"
)

func main() {
	cfg := config.Load()

	srv := server.New(cfg)

	log.Printf("Starting AI Filmmaker API server on port %s (env: %s)...", cfg.Port, cfg.Environment)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
