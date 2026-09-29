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
// Enforces path confinement within baseDir to prevent traversal attacks.
type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error) {
	if baseDir == "" {
		baseDir = "./data/storage"
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local storage path: %w", err)
	}
	if err := os.MkdirAll(absBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local storage directory: %w", err)
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &LocalStorage{
		baseDir: absBase,
		baseURL: baseURL,
	}, nil
}

func (s *LocalStorage) getFilePath(key string) (string, error) {
	if err := ValidateStorageKey(key); err != nil {
		return "", err
	}

	absBase, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve storage base dir: %w", err)
	}

	target := filepath.Join(absBase, filepath.FromSlash(key))
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path: %w", err)
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") || (rel == "." && key != "") {
		return "", ErrStorageTraversal
	}

	return absTarget, nil
}

func (s *LocalStorage) CreateUploadURL(ctx context.Context, key string, mimeType string, expiry time.Duration) (string, error) {
	if err := ValidateStorageKey(key); err != nil {
		return "", err
	}
	encodedKey := strings.ReplaceAll(key, "\\", "/")
	return fmt.Sprintf("%s/storage/upload/%s", s.baseURL, encodedKey), nil
}

func (s *LocalStorage) CreateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if err := ValidateStorageKey(key); err != nil {
		return "", err
	}
	encodedKey := strings.ReplaceAll(key, "\\", "/")
	return fmt.Sprintf("%s/storage/download/%s", s.baseURL, encodedKey), nil
}

func (s *LocalStorage) HeadObject(ctx context.Context, key string) (bool, int64, error) {
	filePath, err := s.getFilePath(key)
	if err != nil {
		return false, 0, err
	}
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
	filePath, err := s.getFilePath(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage file: %w", err)
	}
	return file, nil
}

func (s *LocalStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	filePath, err := s.getFilePath(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	_, copyErr := io.Copy(dst, body)
	dst.Close()

	if copyErr != nil {
		// Clean up partially written file on error (e.g. if MaxBytesReader aborted stream)
		_ = os.Remove(filePath)
		return fmt.Errorf("failed to write object content: %w", copyErr)
	}

	return nil
}

func (s *LocalStorage) DeleteObject(ctx context.Context, key string) error {
	filePath, err := s.getFilePath(key)
	if err != nil {
		return err
	}
	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete storage file: %w", err)
	}
	return nil
}

func (s *LocalStorage) GetLocalPath(ctx context.Context, key string) (string, error) {
	filePath, err := s.getFilePath(key)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}
