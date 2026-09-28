package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/repository"
	"github.com/google/uuid"
)

type ProjectService struct {
	repo repository.ProjectRepository
}

func NewProjectService(repo repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) CreateProject(ctx context.Context, input domain.CreateProjectInput) (*domain.Project, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	project := &domain.Project{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("service failed to create project: %w", err)
	}

	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	return s.repo.List(ctx)
}

func (s *ProjectService) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidProjectID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) UpdateProject(ctx context.Context, id string, input domain.UpdateProjectInput) (*domain.Project, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidProjectID
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Description != nil {
		existing.Description = strings.TrimSpace(*input.Description)
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("service failed to update project: %w", err)
	}

	return existing, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ErrInvalidProjectID
	}
	return s.repo.Delete(ctx, id)
}
