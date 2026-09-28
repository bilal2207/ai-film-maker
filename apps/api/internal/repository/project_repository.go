package repository

import (
	"context"

	"github.com/ai-filmmaker/api/internal/domain"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	List(ctx context.Context) ([]*domain.Project, error)
	GetByID(ctx context.Context, id string) (*domain.Project, error)
	Update(ctx context.Context, project *domain.Project) error
	Delete(ctx context.Context, id string) error
}
