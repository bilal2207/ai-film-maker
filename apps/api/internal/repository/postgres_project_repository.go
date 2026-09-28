package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ai-filmmaker/api/internal/domain"
)

type PostgresProjectRepository struct {
	db *sql.DB
}

func NewPostgresProjectRepository(db *sql.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

func (r *PostgresProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	query := `
		INSERT INTO projects (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5);
	`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.Name, p.Description, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}
	return nil
}

func (r *PostgresProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project row: %w", err)
		}
		projects = append(projects, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if projects == nil {
		projects = []*domain.Project{}
	}

	return projects, nil
}

func (r *PostgresProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1;
	`
	var p domain.Project
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project by id: %w", err)
	}

	return &p, nil
}

func (r *PostgresProjectRepository) Update(ctx context.Context, p *domain.Project) error {
	query := `
		UPDATE projects
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4;
	`
	result, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrProjectNotFound
	}

	return nil
}

func (r *PostgresProjectRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM projects
		WHERE id = $1;
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrProjectNotFound
	}

	return nil
}
