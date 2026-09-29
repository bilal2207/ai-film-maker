package handlers

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ai-filmmaker/api/internal/storage"
)

type StorageHandler struct {
	store storage.Storage
}

func NewStorageHandler(store storage.Storage) *StorageHandler {
	return &StorageHandler{store: store}
}

// Upload handles PUT /storage/upload/{key...}
func (h *StorageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/storage/upload/")
	if key == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "storage key is required")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.store.PutObject(r.Context(), key, r.Body, r.ContentLength, contentType); err != nil {
		WriteError(w, http.StatusInternalServerError, "STORAGE_ERROR", "failed to write object")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"status": "uploaded",
		"key":    key,
	})
}

// Download handles GET /storage/download/{key...}
func (h *StorageHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/storage/download/")
	if key == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "storage key is required")
		return
	}

	localPath, err := h.store.GetLocalPath(r.Context(), key)
	if err == nil {
		// Serve file with range support for video streaming
		http.ServeFile(w, r, localPath)
		return
	}

	// Fallback streaming via GetObject
	reader, err := h.store.GetObject(r.Context(), key)
	if err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "object not found in storage")
		return
	}
	defer reader.Close()

	ext := filepath.Ext(key)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, reader)
	}
}
