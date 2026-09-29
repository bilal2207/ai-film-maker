package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Storage interface {
	CreateUploadURL(ctx context.Context, key string, mimeType string, expiry time.Duration) (string, error)
	CreateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	HeadObject(ctx context.Context, key string) (bool, int64, error)
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error
	DeleteObject(ctx context.Context, key string) error
	GetLocalPath(ctx context.Context, key string) (string, error)
}

// LocalStorage implements Storage interface using the local filesystem.
// Ideal for local development, CI testing, and offline workflows without AWS S3 credentials.
type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error) {
	if baseDir == "" {
		baseDir = "./data/storage"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local storage directory: %w", err)
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &LocalStorage{
		baseDir: baseDir,
		baseURL: baseURL,
	}, nil
}

func (s *LocalStorage) getFilePath(key string) string {
	cleanKey := filepath.Clean(key)
	return filepath.Join(s.baseDir, cleanKey)
}

func (s *LocalStorage) CreateUploadURL(ctx context.Context, key string, mimeType string, expiry time.Duration) (string, error) {
	// Returns direct endpoint served by Go API for local upload
	encodedKey := strings.ReplaceAll(key, "\\", "/")
	return fmt.Sprintf("%s/storage/upload/%s", s.baseURL, encodedKey), nil
}

func (s *LocalStorage) CreateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	encodedKey := strings.ReplaceAll(key, "\\", "/")
	return fmt.Sprintf("%s/storage/download/%s", s.baseURL, encodedKey), nil
}

func (s *LocalStorage) HeadObject(ctx context.Context, key string) (bool, int64, error) {
	filePath := s.getFilePath(key)
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, info.Size(), nil
}

func (s *LocalStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	filePath := s.getFilePath(key)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage file: %w", err)
	}
	return file, nil
}

func (s *LocalStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	filePath := s.getFilePath(key)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, body); err != nil {
		return fmt.Errorf("failed to write object content: %w", err)
	}
	return nil
}

func (s *LocalStorage) DeleteObject(ctx context.Context, key string) error {
	filePath := s.getFilePath(key)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete storage file: %w", err)
	}
	return nil
}

func (s *LocalStorage) GetLocalPath(ctx context.Context, key string) (string, error) {
	filePath := s.getFilePath(key)
	if _, err := os.Stat(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}
