package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/service"
)

type inMemoryRepo struct {
	data map[string]*domain.Project
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{data: make(map[string]*domain.Project)}
}

func (m *inMemoryRepo) Create(ctx context.Context, p *domain.Project) error {
	m.data[p.ID] = p
	return nil
}

func (m *inMemoryRepo) List(ctx context.Context) ([]*domain.Project, error) {
	var list []*domain.Project
	for _, p := range m.data {
		list = append(list, p)
	}
	return list, nil
}

func (m *inMemoryRepo) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	p, ok := m.data[id]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	return p, nil
}

func (m *inMemoryRepo) Update(ctx context.Context, p *domain.Project) error {
	if _, ok := m.data[p.ID]; !ok {
		return domain.ErrProjectNotFound
	}
	m.data[p.ID] = p
	return nil
}

func (m *inMemoryRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.data[id]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(m.data, id)
	return nil
}

func setupTestServer() (http.Handler, *service.ProjectService) {
	repo := newInMemoryRepo()
	svc := service.NewProjectService(repo)
	h := NewProjectHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/projects", h.ProjectDispatcher)
	mux.HandleFunc("/projects/", h.ProjectDispatcher)

	return mux, svc
}

func TestProjectHandler_Create(t *testing.T) {
	mux, _ := setupTestServer()

	t.Run("create project success 201", func(t *testing.T) {
		body := `{"name":"The Great AI Movie","description":"A masterpiece"}`
		req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var created domain.Project
		if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if created.Name != "The Great AI Movie" {
			t.Errorf("expected 'The Great AI Movie', got '%s'", created.Name)
		}
		if created.ID == "" {
			t.Errorf("expected non-empty ID")
		}
	})

	t.Run("create project empty name 400", func(t *testing.T) {
		body := `{"name":"","description":"No name"}`
		req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}

		var errResp ErrorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error.Code != "INVALID_INPUT" {
			t.Errorf("expected INVALID_INPUT code, got '%s'", errResp.Error.Code)
		}
	})
}

func TestProjectHandler_ListAndGet(t *testing.T) {
	mux, svc := setupTestServer()

	p1, _ := svc.CreateProject(context.Background(), domain.CreateProjectInput{Name: "Film 1"})
	p2, _ := svc.CreateProject(context.Background(), domain.CreateProjectInput{Name: "Film 2"})

	t.Run("list projects 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}

		var list []domain.Project
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if len(list) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(list))
		}
	})

	t.Run("get project by id 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+p1.ID, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}

		var got domain.Project
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if got.ID != p1.ID || got.Name != p1.Name {
			t.Errorf("mismatched project data: %+v", got)
		}
	})

	t.Run("get non-existent project 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/missing-id", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", rr.Code)
		}

		var errResp ErrorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error.Code != "NOT_FOUND" {
			t.Errorf("expected NOT_FOUND code, got '%s'", errResp.Error.Code)
		}
	})

	_ = p2
}

func TestProjectHandler_UpdateAndDelete(t *testing.T) {
	mux, svc := setupTestServer()

	p, _ := svc.CreateProject(context.Background(), domain.CreateProjectInput{
		Name:        "Initial Name",
		Description: "Initial Description",
	})

	t.Run("patch update project 200", func(t *testing.T) {
		body := `{"name":"Patched Name"}`
		req := httptest.NewRequest(http.MethodPatch, "/projects/"+p.ID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var updated domain.Project
		if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if updated.Name != "Patched Name" {
			t.Errorf("expected 'Patched Name', got '%s'", updated.Name)
		}
	})

	t.Run("delete project 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/projects/"+p.ID, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d", rr.Code)
		}

		// Verify deletion
		reqGet := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID, nil)
		rrGet := httptest.NewRecorder()
		mux.ServeHTTP(rrGet, reqGet)
		if rrGet.Code != http.StatusNotFound {
			t.Errorf("expected 404 after deletion, got %d", rrGet.Code)
		}
	})
}
