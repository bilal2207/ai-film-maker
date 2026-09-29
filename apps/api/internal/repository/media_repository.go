package repository

import (
	"context"

	"github.com/ai-filmmaker/api/internal/domain"
)

type MediaRepository interface {
	Create(ctx context.Context, media *domain.MediaAsset) error
	GetByID(ctx context.Context, id string) (*domain.MediaAsset, error)
	ListByProjectID(ctx context.Context, projectID string) ([]*domain.MediaAsset, error)
	Update(ctx context.Context, media *domain.MediaAsset) error
	Delete(ctx context.Context, id string) error
}
