package storage

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrInvalidStorageKey = errors.New("invalid or unsafe storage key")
	ErrStorageTraversal  = errors.New("storage key escapes base directory")
)

var validKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-\./]+$`)

// ValidateStorageKey checks that a key does not contain traversal, absolute path markers,
// Windows drive letters, or invalid characters.
func ValidateStorageKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrInvalidStorageKey
	}

	// Reject null bytes, backslashes, control characters
	if strings.Contains(key, "\x00") || strings.Contains(key, "\\") {
		return ErrInvalidStorageKey
	}

	// Reject absolute paths and Windows drive letters (e.g. C:, D:)
	if strings.HasPrefix(key, "/") || (len(key) >= 2 && key[1] == ':') {
		return ErrInvalidStorageKey
	}

	// Reject any encoded traversal tricks or patterns outside whitelist
	if !validKeyPattern.MatchString(key) {
		return ErrInvalidStorageKey
	}

	// Check path segments
	parts := strings.Split(key, "/")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || trimmed == "." || trimmed == ".." {
			return ErrInvalidStorageKey
		}
	}

	// Double check with filepath.Clean
	clean := filepath.ToSlash(filepath.Clean(key))
	if clean != key || strings.HasPrefix(clean, "../") || clean == ".." {
		return ErrInvalidStorageKey
	}

	return nil
}

// ValidateAIMediaKey verifies the key strictly belongs to the AI Filmmaker storage namespace:
// projects/{projectId}/media/{...}
func ValidateAIMediaKey(key string) error {
	if err := ValidateStorageKey(key); err != nil {
		return err
	}
	parts := strings.Split(key, "/")
	if len(parts) < 4 || parts[0] != "projects" || parts[2] != "media" {
		return ErrInvalidStorageKey
	}
	return nil
}
