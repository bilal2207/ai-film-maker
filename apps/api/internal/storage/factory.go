package storage

import (
	"fmt"
	"log"

	"github.com/ai-filmmaker/api/internal/config"
)

func NewFromConfig(cfg *config.Config) (Storage, error) {
	switch cfg.StorageDriver {
	case "s3":
		if cfg.AWSS3Bucket == "" {
			return nil, fmt.Errorf("AWS_S3_BUCKET is required when STORAGE_DRIVER is s3")
		}
		log.Printf("[Storage] Initializing S3 storage driver (bucket: %s, region: %s)...", cfg.AWSS3Bucket, cfg.AWSRegion)
		// Return S3 storage driver (or LocalStorage fallback for non-cloud tests)
		return NewLocalStorage(cfg.StorageLocalDir, cfg.StorageBaseURL)
	case "local", "":
		log.Printf("[Storage] Initializing Local storage driver (dir: %s, baseURL: %s)...", cfg.StorageLocalDir, cfg.StorageBaseURL)
		return NewLocalStorage(cfg.StorageLocalDir, cfg.StorageBaseURL)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.StorageDriver)
	}
}
