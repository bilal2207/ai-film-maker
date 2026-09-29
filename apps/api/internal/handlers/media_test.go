package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/media"
	"github.com/ai-filmmaker/api/internal/service"
	"github.com/ai-filmmaker/api/internal/storage"
)

type mockMediaRepoForHandler struct {
	assets map[string]*domain.MediaAsset
}

func newMockMediaRepoForHandler() *mockMediaRepoForHandler {
	return &mockMediaRepoForHandler{assets: make(map[string]*domain.MediaAsset)}
}

func (m *mockMediaRepoForHandler) Create(ctx context.Context, a *domain.MediaAsset) error {
	m.assets[a.ID] = a
	return nil
}

func (m *mockMediaRepoForHandler) GetByID(ctx context.Context, id string) (*domain.MediaAsset, error) {
	a, ok := m.assets[id]
	if !ok {
		return nil, domain.ErrMediaNotFound
	}
	return a, nil
}

func (m *mockMediaRepoForHandler) ListByProjectID(ctx context.Context, projectID string) ([]*domain.MediaAsset, error) {
	var list []*domain.MediaAsset
	for _, a := range m.assets {
		if a.ProjectID == projectID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *mockMediaRepoForHandler) ListByStatus(ctx context.Context, status domain.MediaStatus) ([]*domain.MediaAsset, error) {
	var list []*domain.MediaAsset
	for _, a := range m.assets {
		if a.Status == status {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *mockMediaRepoForHandler) Update(ctx context.Context, a *domain.MediaAsset) error {
	if _, ok := m.assets[a.ID]; !ok {
		return domain.ErrMediaNotFound
	}
	m.assets[a.ID] = a
	return nil
}

func (m *mockMediaRepoForHandler) Delete(ctx context.Context, id string) error {
	if _, ok := m.assets[id]; !ok {
		return domain.ErrMediaNotFound
	}
	delete(m.assets, id)
	return nil
}

type mockStorageAdapter struct {
	repo *mockMediaRepoForHandler
}

func (s *mockStorageAdapter) CreateUploadURL(ctx context.Context, key string, mimeType string, expiry time.Duration) (string, error) {
	return "http://localhost:8080/storage/upload/" + key, nil
}

func (s *mockStorageAdapter) CreateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "http://localhost:8080/storage/download/" + key, nil
}

func (s *mockStorageAdapter) HeadObject(ctx context.Context, key string) (bool, int64, error) {
	return true, 2048, nil
}

func (s *mockStorageAdapter) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("mock"))), nil
}

func (s *mockStorageAdapter) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	return nil
}

func (s *mockStorageAdapter) DeleteObject(ctx context.Context, key string) error {
	return nil
}

func (s *mockStorageAdapter) GetLocalPath(ctx context.Context, key string) (string, error) {
	return "", nil
}

func setupTestMediaHandler() (http.Handler, *mockMediaRepoForHandler, *inMemoryRepo) {
	mediaRepo := newMockMediaRepoForHandler()
	projectRepo := newInMemoryRepo()

	// Seed a project
	_ = projectRepo.Create(context.Background(), &domain.Project{
		ID:   "proj-abc",
		Name: "Sci-Fi Film",
	})

	q := media.NewMemoryQueue(10)
	mediaSvc := service.NewMediaService(mediaRepo, projectRepo, &mockStorageAdapter{mediaRepo}, nil, q)
	h := NewMediaHandler(mediaSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/projects/", func(w http.ResponseWriter, r *http.Request) {
		h.MediaDispatcher(w, r)
	})

	return mux, mediaRepo, projectRepo
}

func TestMediaHandler_HTTPFlow(t *testing.T) {
	mux, mediaRepo, _ := setupTestMediaHandler()

	var createdMediaID string

	t.Run("init upload 201", func(t *testing.T) {
		body := `{"originalFilename":"take_01.mp4","mimeType":"video/mp4","fileSize":102400}`
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-abc/media/upload", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var out domain.InitUploadOutput
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if out.Media.OriginalFilename != "take_01.mp4" {
			t.Errorf("expected filename 'take_01.mp4', got '%s'", out.Media.OriginalFilename)
		}
		if out.UploadURL == "" {
			t.Errorf("expected upload url")
		}
		createdMediaID = out.Media.ID
	})

	t.Run("complete upload 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-abc/media/"+createdMediaID+"/complete", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var asset domain.MediaAsset
		if err := json.Unmarshal(rr.Body.Bytes(), &asset); err != nil {
			t.Fatalf("failed to decode asset: %v", err)
		}
		if asset.Status != domain.StatusProcessing {
			t.Errorf("expected PROCESSING status, got %s", asset.Status)
		}
	})

	t.Run("list media 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/proj-abc/media", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}

		var list []domain.MediaAsset
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("failed to decode list: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 media asset, got %d", len(list))
		}
	})

	t.Run("get media by id 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/proj-abc/media/"+createdMediaID, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	})

	t.Run("delete media 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/projects/proj-abc/media/"+createdMediaID, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d", rr.Code)
		}

		if len(mediaRepo.assets) != 0 {
			t.Errorf("expected media to be deleted from repo")
		}
	})
}

func TestStorageHandler_SecurityAndLimits(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_storage_sec_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := storage.NewLocalStorage(tempDir, "http://localhost:8080")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	handler := NewStorageHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("/storage/upload/", handler.Upload)
	mux.HandleFunc("/storage/download/", handler.Download)

	t.Run("upload with traversal key rejected 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/storage/upload/projects/p1/media/m1/../../../outside.txt", bytes.NewBufferString("attack"))
		rr := httptest.NewRecorder()
		handler.Upload(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for traversal upload, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("upload outside AI media namespace rejected 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/storage/upload/some_random_file.txt", bytes.NewBufferString("data"))
		rr := httptest.NewRecorder()
		handler.Upload(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for non-AI namespace upload, got %d", rr.Code)
		}
	})

	t.Run("upload with oversized ContentLength rejected 413", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/storage/upload/projects/p1/media/m1/original/take.mp4", bytes.NewBufferString("small body"))
		req.ContentLength = 6 * 1024 * 1024 * 1024 // 6 GB
		rr := httptest.NewRecorder()
		handler.Upload(rr, req)

		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected 413 Payload Too Large, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("download with traversal key rejected 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/storage/download/projects/p1/media/m1/../../../etc/passwd", nil)
		rr := httptest.NewRecorder()
		handler.Download(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for traversal download, got %d", rr.Code)
		}
	})

	t.Run("valid upload and download succeeds 200", func(t *testing.T) {
		key := "projects/p1/media/m1/original/take.mp4"
		content := []byte("valid video payload")
		req := httptest.NewRequest(http.MethodPut, "/storage/upload/"+key, bytes.NewReader(content))
		req.Header.Set("Content-Type", "video/mp4")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for valid upload, got %d: %s", rr.Code, rr.Body.String())
		}

		downReq := httptest.NewRequest(http.MethodGet, "/storage/download/"+key, nil)
		downRR := httptest.NewRecorder()
		mux.ServeHTTP(downRR, downReq)

		if downRR.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for valid download, got %d: %s", downRR.Code, downRR.Body.String())
		}
		if !bytes.Equal(downRR.Body.Bytes(), content) {
			t.Errorf("download content does not match uploaded content")
		}
	})
}
