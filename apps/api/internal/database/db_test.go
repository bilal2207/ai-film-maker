package database

import (
	"testing"
	"time"
)

func TestConnect_InvalidURL(t *testing.T) {
	t.Run("empty URL returns error", func(t *testing.T) {
		_, err := Connect(Config{URL: ""})
		if err == nil {
			t.Fatalf("expected error for empty database URL, got nil")
		}
	})

	t.Run("unreachable database returns ping error", func(t *testing.T) {
		cfg := Config{
			URL:             "postgres://invalid_user:invalid_pass@127.0.0.1:59999/non_existent_db?sslmode=disable",
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 1 * time.Second,
		}
		_, err := Connect(cfg)
		if err == nil {
			t.Fatalf("expected connection error for unreachable database, got nil")
		}
	})
}
