package storage

import (
	"fmt"
	"log"

	"github.com/ai-filmmaker/api/internal/config"
)

func NewFromConfig(cfg *config.Config) (Storage, error) {
	switch cfg.StorageDriver {
	case "local", "":
		log.Printf("[Storage] Initializing Local storage driver (dir: %s, baseURL: %s)...", cfg.StorageLocalDir, cfg.StorageBaseURL)
		return NewLocalStorage(cfg.StorageLocalDir, cfg.StorageBaseURL)
	case "s3":
		return nil, fmt.Errorf("storage driver %q is unsupported in Hop 2; only %q is supported", cfg.StorageDriver, "local")
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.StorageDriver)
	}
}
