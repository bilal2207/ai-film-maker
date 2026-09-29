package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ai-filmmaker/api/internal/domain"
)

type PostgresMediaRepository struct {
	db *sql.DB
}

func NewPostgresMediaRepository(db *sql.DB) *PostgresMediaRepository {
	return &PostgresMediaRepository{db: db}
}

func (r *PostgresMediaRepository) Create(ctx context.Context, m *domain.MediaAsset) error {
	query := `
		INSERT INTO media_assets (
			id, project_id, original_object_key, proxy_object_key, thumbnail_object_key,
			original_filename, mime_type, file_size, duration, width, height, fps,
			status, error_message, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		);
	`
	_, err := r.db.ExecContext(
		ctx, query,
		m.ID, m.ProjectID, m.OriginalObjectKey, m.ProxyObjectKey, m.ThumbnailObjectKey,
		m.OriginalFilename, m.MimeType, m.FileSize, m.Duration, m.Width, m.Height, m.FPS,
		string(m.Status), m.ErrorMessage, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert media asset: %w", err)
	}
	return nil
}

func (r *PostgresMediaRepository) GetByID(ctx context.Context, id string) (*domain.MediaAsset, error) {
	query := `
		SELECT
			id, project_id, original_object_key, proxy_object_key, thumbnail_object_key,
			original_filename, mime_type, file_size, duration, width, height, fps,
			status, error_message, created_at, updated_at
		FROM media_assets
		WHERE id = $1;
	`
	var m domain.MediaAsset
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.ProjectID, &m.OriginalObjectKey, &m.ProxyObjectKey, &m.ThumbnailObjectKey,
		&m.OriginalFilename, &m.MimeType, &m.FileSize, &m.Duration, &m.Width, &m.Height, &m.FPS,
		&statusStr, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMediaNotFound
		}
		return nil, fmt.Errorf("failed to query media asset by id: %w", err)
	}
	m.Status = domain.MediaStatus(statusStr)
	return &m, nil
}

func (r *PostgresMediaRepository) ListByProjectID(ctx context.Context, projectID string) ([]*domain.MediaAsset, error) {
	query := `
		SELECT
			id, project_id, original_object_key, proxy_object_key, thumbnail_object_key,
			original_filename, mime_type, file_size, duration, width, height, fps,
			status, error_message, created_at, updated_at
		FROM media_assets
		WHERE project_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query media assets by project id: %w", err)
	}
	defer rows.Close()

	var assets []*domain.MediaAsset
	for rows.Next() {
		var m domain.MediaAsset
		var statusStr string
		err := rows.Scan(
			&m.ID, &m.ProjectID, &m.OriginalObjectKey, &m.ProxyObjectKey, &m.ThumbnailObjectKey,
			&m.OriginalFilename, &m.MimeType, &m.FileSize, &m.Duration, &m.Width, &m.Height, &m.FPS,
			&statusStr, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media asset row: %w", err)
		}
		m.Status = domain.MediaStatus(statusStr)
		assets = append(assets, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if assets == nil {
		assets = []*domain.MediaAsset{}
	}

	return assets, nil
}

func (r *PostgresMediaRepository) ListByStatus(ctx context.Context, status domain.MediaStatus) ([]*domain.MediaAsset, error) {
	query := `
		SELECT
			id, project_id, original_object_key, proxy_object_key, thumbnail_object_key,
			original_filename, mime_type, file_size, duration, width, height, fps,
			status, error_message, created_at, updated_at
		FROM media_assets
		WHERE status = $1
		ORDER BY created_at ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, string(status))
	if err != nil {
		return nil, fmt.Errorf("failed to query media assets by status: %w", err)
	}
	defer rows.Close()

	var assets []*domain.MediaAsset
	for rows.Next() {
		var m domain.MediaAsset
		var statusStr string
		err := rows.Scan(
			&m.ID, &m.ProjectID, &m.OriginalObjectKey, &m.ProxyObjectKey, &m.ThumbnailObjectKey,
			&m.OriginalFilename, &m.MimeType, &m.FileSize, &m.Duration, &m.Width, &m.Height, &m.FPS,
			&statusStr, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media asset row: %w", err)
		}
		m.Status = domain.MediaStatus(statusStr)
		assets = append(assets, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if assets == nil {
		assets = []*domain.MediaAsset{}
	}

	return assets, nil
}

func (r *PostgresMediaRepository) Update(ctx context.Context, m *domain.MediaAsset) error {
	query := `
		UPDATE media_assets
		SET
			proxy_object_key = $1,
			thumbnail_object_key = $2,
			file_size = $3,
			duration = $4,
			width = $5,
			height = $6,
			fps = $7,
			status = $8,
			error_message = $9,
			updated_at = $10
		WHERE id = $11;
	`
	res, err := r.db.ExecContext(
		ctx, query,
		m.ProxyObjectKey, m.ThumbnailObjectKey, m.FileSize, m.Duration,
		m.Width, m.Height, m.FPS, string(m.Status), m.ErrorMessage,
		m.UpdatedAt, m.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update media asset: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrMediaNotFound
	}

	return nil
}

func (r *PostgresMediaRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM media_assets WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete media asset: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrMediaNotFound
	}

	return nil
}
