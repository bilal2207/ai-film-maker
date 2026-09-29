package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/handlers"
)

type mockRepo struct {
	projects map[string]*domain.Project
}

func newMockRepo() *mockRepo {
	return &mockRepo{projects: make(map[string]*domain.Project)}
}

func (m *mockRepo) Create(ctx context.Context, p *domain.Project) error {
	m.projects[p.ID] = p
	return nil
}

func (m *mockRepo) List(ctx context.Context) ([]*domain.Project, error) {
	list := make([]*domain.Project, 0, len(m.projects))
	for _, p := range m.projects {
		list = append(list, p)
	}
	return list, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	return p, nil
}

func (m *mockRepo) Update(ctx context.Context, p *domain.Project) error {
	if _, ok := m.projects[p.ID]; !ok {
		return domain.ErrProjectNotFound
	}
	m.projects[p.ID] = p
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.projects[id]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(m.projects, id)
	return nil
}

func TestServerRoutes(t *testing.T) {
	cfg := &config.Config{
		Port: "8080",
	}
	repo := newMockRepo()
	s := NewWithRepository(cfg, repo)

	t.Run("health check route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		s.Router().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var resp handlers.HealthResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "ok" || resp.Service != "api" {
			t.Errorf("unexpected health response: %+v", resp)
		}
	})

	t.Run("projects route", func(t *testing.T) {
		body := `{"name":"Test Production"}`
		req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.Router().ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("cors preflight options", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/projects", nil)
		rr := httptest.NewRecorder()

		s.Router().ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content for OPTIONS, got %d", rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("expected CORS allow origin header")
		}
	})
}
