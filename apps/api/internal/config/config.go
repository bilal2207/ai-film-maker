package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	Environment     string
	StorageDriver   string
	StorageLocalDir string
	StorageBaseURL  string
	AWSRegion       string
	AWSS3Bucket     string
	AWSEndpoint     string
	FFmpegPath      string
	FFprobePath     string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("API_PORT")
	}
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/aifilmmaker?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	storageDriver := os.Getenv("STORAGE_DRIVER")
	if storageDriver == "" {
		if os.Getenv("AWS_S3_BUCKET") != "" {
			storageDriver = "s3"
		} else {
			storageDriver = "local"
		}
	}

	storageLocalDir := os.Getenv("STORAGE_LOCAL_DIR")
	if storageLocalDir == "" {
		storageLocalDir = "./data/storage"
	}

	storageBaseURL := os.Getenv("STORAGE_BASE_URL")
	if storageBaseURL == "" {
		storageBaseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	return &Config{
		Port:            port,
		DatabaseURL:     dbURL,
		RedisURL:        redisURL,
		Environment:     env,
		StorageDriver:   storageDriver,
		StorageLocalDir: storageLocalDir,
		StorageBaseURL:  storageBaseURL,
		AWSRegion:       os.Getenv("AWS_REGION"),
		AWSS3Bucket:     os.Getenv("AWS_S3_BUCKET"),
		AWSEndpoint:     os.Getenv("AWS_ENDPOINT"),
		FFmpegPath:      os.Getenv("FFMPEG_PATH"),
		FFprobePath:     os.Getenv("FFPROBE_PATH"),
	}
}
