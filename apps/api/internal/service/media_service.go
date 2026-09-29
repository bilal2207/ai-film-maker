package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/media"
	"github.com/ai-filmmaker/api/internal/repository"
	"github.com/ai-filmmaker/api/internal/storage"
	"github.com/google/uuid"
)

type MediaService struct {
	mediaRepo   repository.MediaRepository
	projectRepo repository.ProjectRepository
	storage     storage.Storage
	processor   media.Processor
	queue       media.Queue
}

func NewMediaService(
	mediaRepo repository.MediaRepository,
	projectRepo repository.ProjectRepository,
	storage storage.Storage,
	processor media.Processor,
	queue media.Queue,
) *MediaService {
	svc := &MediaService{
		mediaRepo:   mediaRepo,
		projectRepo: projectRepo,
		storage:     storage,
		processor:   processor,
		queue:       queue,
	}

	if queue != nil {
		queue.Start(2, svc.handleBackgroundJob)
	}

	// Reconcile orphaned processing records from prior process runs/crashes
	if mediaRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := svc.ReconcileOrphanedProcessing(ctx); err != nil {
			log.Printf("[MediaService] Warning: failed to reconcile orphaned processing assets: %v", err)
		}
	}

	return svc
}

// ReconcileOrphanedProcessing reconciles any media assets left in PROCESSING status across API restarts.
// Because the in-memory queue is not durable across restarts, in-flight jobs are marked as FAILED with
// a clear diagnostic message allowing the user/system to retry.
func (s *MediaService) ReconcileOrphanedProcessing(ctx context.Context) error {
	if s.mediaRepo == nil {
		return nil
	}

	processingAssets, err := s.mediaRepo.ListByStatus(ctx, domain.StatusProcessing)
	if err != nil {
		return fmt.Errorf("failed to list processing assets: %w", err)
	}

	for _, asset := range processingAssets {
		log.Printf("[MediaService] Reconciling orphaned processing asset %s (project %s)...", asset.ID, asset.ProjectID)
		errMsg := "processing interrupted by server restart; retry available"
		asset.Status = domain.StatusFailed
		asset.ErrorMessage = &errMsg
		asset.UpdatedAt = time.Now().UTC()
		if err := s.mediaRepo.Update(ctx, asset); err != nil {
			log.Printf("[MediaService] Failed to update orphaned asset %s: %v", asset.ID, err)
		}
	}

	return nil
}

func (s *MediaService) handleBackgroundJob(ctx context.Context, job media.Job) error {
	return s.ProcessMedia(ctx, job.MediaID)
}

func (s *MediaService) InitUpload(ctx context.Context, projectID string, input domain.InitUploadInput) (*domain.InitUploadOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	// 1. Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	// 2. Validate upload input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	mediaID := uuid.New().String()
	objectKey := fmt.Sprintf("projects/%s/media/%s/original/%s", projectID, mediaID, input.OriginalFilename)

	// 3. Generate presigned upload URL
	uploadURL, err := s.storage.CreateUploadURL(ctx, objectKey, input.MimeType, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	now := time.Now().UTC()
	asset := &domain.MediaAsset{
		ID:                mediaID,
		ProjectID:         projectID,
		OriginalObjectKey: objectKey,
		OriginalFilename:  input.OriginalFilename,
		MimeType:          input.MimeType,
		FileSize:          input.FileSize,
		Status:            domain.StatusUploading,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// 4. Save media asset record
	if err := s.mediaRepo.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("failed to create media asset record: %w", err)
	}

	log.Printf("[MediaService] Media upload initialized: media_id=%s, project_id=%s, filename=%s", mediaID, projectID, input.OriginalFilename)

	return &domain.InitUploadOutput{
		Media:     asset,
		UploadURL: uploadURL,
		ObjectKey: objectKey,
	}, nil
}

func (s *MediaService) CompleteUpload(ctx context.Context, projectID, mediaID string) (*domain.MediaAsset, error) {
	projectID = strings.TrimSpace(projectID)
	mediaID = strings.TrimSpace(mediaID)

	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}
	if mediaID == "" {
		return nil, domain.ErrInvalidMediaID
	}

	// 1. Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	// 2. Fetch media asset
	asset, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	if asset.ProjectID != projectID {
		return nil, domain.ErrMediaNotBelongToProject
	}

	// 3. Validate status transition (supports UPLOADING -> PROCESSING or FAILED -> PROCESSING for retry)
	if !asset.CanTransitionTo(domain.StatusProcessing) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", domain.ErrInvalidStatusTransition, asset.Status, domain.StatusProcessing)
	}

	// 4. Verify object exists in storage
	exists, size, err := s.storage.HeadObject(ctx, asset.OriginalObjectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check storage object: %w", err)
	}
	if !exists {
		return nil, domain.ErrObjectNotFoundInStorage
	}

	if size > 0 {
		asset.FileSize = size
	}

	// 5. Enqueue background processing job before committing PROCESSING status to database
	if s.queue == nil {
		errMsg := "background processing queue is unconfigured"
		asset.Status = domain.StatusFailed
		asset.ErrorMessage = &errMsg
		asset.UpdatedAt = time.Now().UTC()
		_ = s.mediaRepo.Update(ctx, asset)
		return nil, errors.New("background processing queue is unconfigured")
	}

	if err := s.queue.Enqueue(media.Job{MediaID: mediaID, ProjectID: projectID}); err != nil {
		log.Printf("[MediaService] Failed to enqueue media processing job for media_id=%s: %v", mediaID, err)
		errMsg := fmt.Sprintf("failed to enqueue media processing: %v", err)
		asset.Status = domain.StatusFailed
		asset.ErrorMessage = &errMsg
		asset.UpdatedAt = time.Now().UTC()
		_ = s.mediaRepo.Update(ctx, asset)
		return nil, fmt.Errorf("failed to enqueue background job: %w", err)
	}

	// 6. Transition to PROCESSING and persist
	asset.Status = domain.StatusProcessing
	asset.ErrorMessage = nil
	asset.UpdatedAt = time.Now().UTC()

	if err := s.mediaRepo.Update(ctx, asset); err != nil {
		return nil, fmt.Errorf("failed to update media status: %w", err)
	}

	log.Printf("[MediaService] Media upload completed and enqueued: media_id=%s, project_id=%s, size=%d bytes.", mediaID, projectID, asset.FileSize)
	return asset, nil
}

func (s *MediaService) ProcessMedia(ctx context.Context, mediaID string) error {
	asset, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("failed to retrieve media asset for processing: %w", err)
	}

	// Idempotency: skip if already ready
	if asset.Status == domain.StatusReady {
		log.Printf("[MediaService] Media %s is already READY. Skipping processing.", mediaID)
		return nil
	}

	log.Printf("[MediaService] Starting media processing pipeline: media_id=%s, project_id=%s", asset.ID, asset.ProjectID)

	// Create a temporary workspace directory for FFmpeg
	tempDir, err := os.MkdirTemp("", "media_proc_*")
	if err != nil {
		return s.failProcessing(ctx, asset, fmt.Errorf("failed to create temporary working directory: %w", err))
	}
	defer os.RemoveAll(tempDir)

	// Obtain input file path (direct if local storage, or download to temp)
	inputPath, err := s.storage.GetLocalPath(ctx, asset.OriginalObjectKey)
	if err != nil {
		// Download from storage to temp file
		inputPath = filepath.Join(tempDir, "original_"+asset.OriginalFilename)
		reader, err := s.storage.GetObject(ctx, asset.OriginalObjectKey)
		if err != nil {
			return s.failProcessing(ctx, asset, fmt.Errorf("failed to download original media from storage: %w", err))
		}
		defer reader.Close()

		outFile, err := os.Create(inputPath)
		if err != nil {
			return s.failProcessing(ctx, asset, fmt.Errorf("failed to create temp input file: %w", err))
		}
		if _, err := io.Copy(outFile, reader); err != nil {
			outFile.Close()
			return s.failProcessing(ctx, asset, fmt.Errorf("failed to write temp input file: %w", err))
		}
		outFile.Close()
	}

	// 1. Extract metadata with ffprobe
	if s.processor == nil {
		return s.failProcessing(ctx, asset, errors.New("media processor not configured"))
	}

	meta, err := s.processor.InspectMetadata(ctx, inputPath)
	if err != nil {
		return s.failProcessing(ctx, asset, fmt.Errorf("metadata inspection failed: %w", err))
	}

	asset.Duration = meta.Duration
	asset.Width = meta.Width
	asset.Height = meta.Height
	asset.FPS = meta.FPS

	// 2. Generate proxy with FFmpeg
	proxyTempPath := filepath.Join(tempDir, "proxy.mp4")
	if err := s.processor.GenerateProxy(ctx, inputPath, proxyTempPath); err != nil {
		return s.failProcessing(ctx, asset, fmt.Errorf("proxy generation failed: %w", err))
	}

	// Upload proxy to storage
	proxyKey := fmt.Sprintf("projects/%s/media/%s/proxy/proxy.mp4", asset.ProjectID, asset.ID)
	proxyFile, err := os.Open(proxyTempPath)
	if err != nil {
		return s.failProcessing(ctx, asset, fmt.Errorf("failed to open generated proxy: %w", err))
	}
	proxyStat, _ := proxyFile.Stat()
	if err := s.storage.PutObject(ctx, proxyKey, proxyFile, proxyStat.Size(), "video/mp4"); err != nil {
		proxyFile.Close()
		return s.failProcessing(ctx, asset, fmt.Errorf("failed to upload proxy to storage: %w", err))
	}
	proxyFile.Close()
	asset.ProxyObjectKey = &proxyKey

	// 3. Generate thumbnail with FFmpeg
	thumbTempPath := filepath.Join(tempDir, "thumb.jpg")
	thumbTime := math.Min(1.0, math.Max(0, meta.Duration/2))
	if err := s.processor.GenerateThumbnail(ctx, inputPath, thumbTempPath, thumbTime); err != nil {
		log.Printf("[MediaService] Warning: thumbnail generation failed: %v", err)
	} else {
		thumbKey := fmt.Sprintf("projects/%s/media/%s/thumbnails/thumb.jpg", asset.ProjectID, asset.ID)
		thumbFile, err := os.Open(thumbTempPath)
		if err == nil {
			thumbStat, _ := thumbFile.Stat()
			if err := s.storage.PutObject(ctx, thumbKey, thumbFile, thumbStat.Size(), "image/jpeg"); err == nil {
				asset.ThumbnailObjectKey = &thumbKey
			}
			thumbFile.Close()
		}
	}

	// 4. Mark READY
	asset.Status = domain.StatusReady
	asset.ErrorMessage = nil
	asset.UpdatedAt = time.Now().UTC()

	if err := s.mediaRepo.Update(ctx, asset); err != nil {
		return fmt.Errorf("failed to update media record to READY: %w", err)
	}

	log.Printf("[MediaService] Media processing succeeded: media_id=%s, duration=%.2fs, resolution=%dx%d, fps=%.2f", asset.ID, asset.Duration, asset.Width, asset.Height, asset.FPS)
	return nil
}

func (s *MediaService) failProcessing(ctx context.Context, asset *domain.MediaAsset, originalErr error) error {
	log.Printf("[MediaService] Media processing failed for media_id=%s: %v", asset.ID, originalErr)

	errMsg := "media processing failed"
	if originalErr != nil {
		errMsg = originalErr.Error()
	}

	asset.Status = domain.StatusFailed
	asset.ErrorMessage = &errMsg
	asset.UpdatedAt = time.Now().UTC()

	_ = s.mediaRepo.Update(ctx, asset)
	return originalErr
}

func (s *MediaService) ListMedia(ctx context.Context, projectID string) ([]*domain.MediaAsset, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	assets, err := s.mediaRepo.ListByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	for _, a := range assets {
		s.populateURLs(ctx, a)
	}

	return assets, nil
}

func (s *MediaService) GetMedia(ctx context.Context, projectID, mediaID string) (*domain.MediaAsset, error) {
	projectID = strings.TrimSpace(projectID)
	mediaID = strings.TrimSpace(mediaID)

	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}
	if mediaID == "" {
		return nil, domain.ErrInvalidMediaID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	asset, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	if asset.ProjectID != projectID {
		return nil, domain.ErrMediaNotBelongToProject
	}

	s.populateURLs(ctx, asset)
	return asset, nil
}

func (s *MediaService) DeleteMedia(ctx context.Context, projectID, mediaID string) error {
	projectID = strings.TrimSpace(projectID)
	mediaID = strings.TrimSpace(mediaID)

	if projectID == "" {
		return domain.ErrInvalidProjectID
	}
	if mediaID == "" {
		return domain.ErrInvalidMediaID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return err
	}

	asset, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}

	if asset.ProjectID != projectID {
		return domain.ErrMediaNotBelongToProject
	}

	// Delete storage objects
	if asset.OriginalObjectKey != "" {
		_ = s.storage.DeleteObject(ctx, asset.OriginalObjectKey)
	}
	if asset.ProxyObjectKey != nil && *asset.ProxyObjectKey != "" {
		_ = s.storage.DeleteObject(ctx, *asset.ProxyObjectKey)
	}
	if asset.ThumbnailObjectKey != nil && *asset.ThumbnailObjectKey != "" {
		_ = s.storage.DeleteObject(ctx, *asset.ThumbnailObjectKey)
	}

	return s.mediaRepo.Delete(ctx, mediaID)
}

func (s *MediaService) populateURLs(ctx context.Context, a *domain.MediaAsset) {
	if a.OriginalObjectKey != "" {
		if u, err := s.storage.CreateDownloadURL(ctx, a.OriginalObjectKey, 1*time.Hour); err == nil {
			a.DownloadURL = &u
		}
	}
	if a.ProxyObjectKey != nil && *a.ProxyObjectKey != "" {
		if u, err := s.storage.CreateDownloadURL(ctx, *a.ProxyObjectKey, 1*time.Hour); err == nil {
			a.ProxyURL = &u
		}
	}
	if a.ThumbnailObjectKey != nil && *a.ThumbnailObjectKey != "" {
		if u, err := s.storage.CreateDownloadURL(ctx, *a.ThumbnailObjectKey, 1*time.Hour); err == nil {
			a.ThumbnailURL = &u
		}
	}
}
