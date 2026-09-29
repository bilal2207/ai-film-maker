package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/service"
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

	mediaSvc := service.NewMediaService(mediaRepo, projectRepo, &mockStorageAdapter{mediaRepo}, nil, nil)
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
