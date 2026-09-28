package service

import (
	"context"
	"testing"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
)

type mockProjectRepo struct {
	projects map[string]*domain.Project
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[string]*domain.Project),
	}
}

func (m *mockProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	m.projects[p.ID] = p
	return nil
}

func (m *mockProjectRepo) List(ctx context.Context) ([]*domain.Project, error) {
	list := make([]*domain.Project, 0, len(m.projects))
	for _, p := range m.projects {
		list = append(list, p)
	}
	return list, nil
}

func (m *mockProjectRepo) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	if _, ok := m.projects[p.ID]; !ok {
		return domain.ErrProjectNotFound
	}
	m.projects[p.ID] = p
	return nil
}

func (m *mockProjectRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.projects[id]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(m.projects, id)
	return nil
}

func TestProjectService_CreateProject(t *testing.T) {
	repo := newMockProjectRepo()
	svc := NewProjectService(repo)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		p, err := svc.CreateProject(ctx, domain.CreateProjectInput{
			Name:        "Sci-Fi Odyssey",
			Description: "A space exploration epic",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.ID == "" {
			t.Errorf("expected non-empty ID")
		}
		if p.Name != "Sci-Fi Odyssey" {
			t.Errorf("expected name 'Sci-Fi Odyssey', got '%s'", p.Name)
		}
		if p.Description != "A space exploration epic" {
			t.Errorf("expected description 'A space exploration epic', got '%s'", p.Description)
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := svc.CreateProject(ctx, domain.CreateProjectInput{
			Name: "   ",
		})
		if err == nil {
			t.Fatalf("expected error for empty name, got nil")
		}
	})
}

func TestProjectService_ListAndGet(t *testing.T) {
	repo := newMockProjectRepo()
	svc := NewProjectService(repo)
	ctx := context.Background()

	p1, _ := svc.CreateProject(ctx, domain.CreateProjectInput{Name: "Project Alpha"})
	p2, _ := svc.CreateProject(ctx, domain.CreateProjectInput{Name: "Project Beta"})

	t.Run("list returns all projects", func(t *testing.T) {
		list, err := svc.ListProjects(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(list))
		}
	})

	t.Run("get by ID success", func(t *testing.T) {
		got, err := svc.GetProject(ctx, p1.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != p1.Name {
			t.Errorf("expected '%s', got '%s'", p1.Name, got.Name)
		}
	})

	t.Run("get by ID not found", func(t *testing.T) {
		_, err := svc.GetProject(ctx, "non-existent-id")
		if err == nil {
			t.Fatalf("expected not found error, got nil")
		}
	})

	t.Run("empty id fails validation", func(t *testing.T) {
		_, err := svc.GetProject(ctx, "   ")
		if err == nil {
			t.Fatalf("expected error for empty id, got nil")
		}
	})

	_ = p2
}

func TestProjectService_Update(t *testing.T) {
	repo := newMockProjectRepo()
	svc := NewProjectService(repo)
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, domain.CreateProjectInput{
		Name:        "Original Name",
		Description: "Original Description",
	})

	t.Run("update name and description", func(t *testing.T) {
		newName := "Updated Name"
		newDesc := "Updated Description"
		updated, err := svc.UpdateProject(ctx, created.ID, domain.UpdateProjectInput{
			Name:        &newName,
			Description: &newDesc,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "Updated Name" {
			t.Errorf("expected 'Updated Name', got '%s'", updated.Name)
		}
		if updated.Description != "Updated Description" {
			t.Errorf("expected 'Updated Description', got '%s'", updated.Description)
		}
	})

	t.Run("update non-existent project fails", func(t *testing.T) {
		newName := "Name"
		_, err := svc.UpdateProject(ctx, "random-id", domain.UpdateProjectInput{
			Name: &newName,
		})
		if err == nil {
			t.Fatalf("expected error for non-existent project")
		}
	})
}

func TestProjectService_Delete(t *testing.T) {
	repo := newMockProjectRepo()
	svc := NewProjectService(repo)
	ctx := context.Background()

	created, _ := svc.CreateProject(ctx, domain.CreateProjectInput{Name: "To Delete"})

	t.Run("delete existing project", func(t *testing.T) {
		err := svc.DeleteProject(ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = svc.GetProject(ctx, created.ID)
		if err == nil {
			t.Fatalf("expected project to be deleted")
		}
	})

	t.Run("delete non-existent project fails", func(t *testing.T) {
		err := svc.DeleteProject(ctx, "non-existent")
		if err == nil {
			t.Fatalf("expected error deleting non-existent project")
		}
	})
}

func TestProjectDomainValidation(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		in := domain.CreateProjectInput{Name: "Short Film"}
		if err := in.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("name over 255 chars", func(t *testing.T) {
		longName := make([]byte, 256)
		for i := range longName {
			longName[i] = 'a'
		}
		in := domain.CreateProjectInput{Name: string(longName)}
		if err := in.Validate(); err == nil {
			t.Errorf("expected error for name > 255 chars")
		}
	})

	t.Run("update with empty name", func(t *testing.T) {
		empty := "   "
		in := domain.UpdateProjectInput{Name: &empty}
		if err := in.Validate(); err == nil {
			t.Errorf("expected error for empty name in update")
		}
	})
}

var _ = time.Now // suppress unused import warning if any
