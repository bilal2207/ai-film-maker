package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/media"
)

type mockMediaRepo struct {
	assets map[string]*domain.MediaAsset
}

func newMockMediaRepo() *mockMediaRepo {
	return &mockMediaRepo{assets: make(map[string]*domain.MediaAsset)}
}

func (m *mockMediaRepo) Create(ctx context.Context, a *domain.MediaAsset) error {
	m.assets[a.ID] = a
	return nil
}

func (m *mockMediaRepo) GetByID(ctx context.Context, id string) (*domain.MediaAsset, error) {
	a, ok := m.assets[id]
	if !ok {
		return nil, domain.ErrMediaNotFound
	}
	return a, nil
}

func (m *mockMediaRepo) ListByProjectID(ctx context.Context, projectID string) ([]*domain.MediaAsset, error) {
	var list []*domain.MediaAsset
	for _, a := range m.assets {
		if a.ProjectID == projectID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *mockMediaRepo) ListByStatus(ctx context.Context, status domain.MediaStatus) ([]*domain.MediaAsset, error) {
	var list []*domain.MediaAsset
	for _, a := range m.assets {
		if a.Status == status {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *mockMediaRepo) Update(ctx context.Context, a *domain.MediaAsset) error {
	if _, ok := m.assets[a.ID]; !ok {
		return domain.ErrMediaNotFound
	}
	m.assets[a.ID] = a
	return nil
}

func (m *mockMediaRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.assets[id]; !ok {
		return domain.ErrMediaNotFound
	}
	delete(m.assets, id)
	return nil
}

type mockStorage struct {
	objects map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{objects: make(map[string][]byte)}
}

func (s *mockStorage) CreateUploadURL(ctx context.Context, key string, mimeType string, expiry time.Duration) (string, error) {
	return "http://localhost:8080/storage/upload/" + key, nil
}

func (s *mockStorage) CreateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "http://localhost:8080/storage/download/" + key, nil
}

func (s *mockStorage) HeadObject(ctx context.Context, key string) (bool, int64, error) {
	data, ok := s.objects[key]
	if !ok {
		return false, 0, nil
	}
	return true, int64(len(data)), nil
}

func (s *mockStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *mockStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.objects[key] = data
	return nil
}

func (s *mockStorage) DeleteObject(ctx context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *mockStorage) GetLocalPath(ctx context.Context, key string) (string, error) {
	return "", errors.New("local path not supported in mock")
}

type mockProcessor struct {
	failInspect bool
}

func (p *mockProcessor) InspectMetadata(ctx context.Context, inputPath string) (*domain.MediaMetadata, error) {
	if p.failInspect {
		return nil, errors.New("ffprobe mock failure")
	}
	return &domain.MediaMetadata{
		Duration: 12.5,
		Width:    1920,
		Height:   1080,
		FPS:      24.0,
		Codec:    "h264",
	}, nil
}

func (p *mockProcessor) GenerateProxy(ctx context.Context, inputPath, outputPath string) error {
	return os.WriteFile(outputPath, []byte("dummy-proxy-mp4-data"), 0644)
}

func (p *mockProcessor) GenerateThumbnail(ctx context.Context, inputPath, outputPath string, atSeconds float64) error {
	return os.WriteFile(outputPath, []byte("dummy-thumbnail-jpg-data"), 0644)
}

type failingQueue struct {
	err error
}

func (q *failingQueue) Enqueue(job media.Job) error {
	return q.err
}

func (q *failingQueue) Start(workers int, handler media.JobHandler) {}
func (q *failingQueue) Stop()                                       {}

func setupTestMediaService() (*MediaService, *mockMediaRepo, *mockProjectRepo, *mockStorage, *mockProcessor) {
	mediaRepo := newMockMediaRepo()
	projectRepo := newMockProjectRepo()
	store := newMockStorage()
	proc := &mockProcessor{}
	queue := media.NewMemoryQueue(10)

	svc := NewMediaService(mediaRepo, projectRepo, store, proc, queue)
	return svc, mediaRepo, projectRepo, store, proc
}

func TestMediaService_UploadAndComplete(t *testing.T) {
	svc, _, projectRepo, store, _ := setupTestMediaService()
	ctx := context.Background()

	// 1. Setup project
	_ = projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Film",
	})

	t.Run("init upload success", func(t *testing.T) {
		out, err := svc.InitUpload(ctx, "proj-1", domain.InitUploadInput{
			OriginalFilename: "take_01.mp4",
			MimeType:         "video/mp4",
			FileSize:         1024 * 1024 * 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Media.Status != domain.StatusUploading {
			t.Errorf("expected status UPLOADING, got %s", out.Media.Status)
		}
		if out.UploadURL == "" || out.ObjectKey == "" {
			t.Errorf("expected upload url and object key")
		}

		// 2. Complete upload without file in storage -> fails
		_, err = svc.CompleteUpload(ctx, "proj-1", out.Media.ID)
		if !errors.Is(err, domain.ErrObjectNotFoundInStorage) {
			t.Errorf("expected ErrObjectNotFoundInStorage, got %v", err)
		}

		// 3. Place object in storage and complete upload -> transitions to PROCESSING
		store.objects[out.ObjectKey] = []byte("mock video bytes")
		completed, err := svc.CompleteUpload(ctx, "proj-1", out.Media.ID)
		if err != nil {
			t.Fatalf("unexpected error completing upload: %v", err)
		}
		if completed.Status != domain.StatusProcessing {
			t.Errorf("expected status PROCESSING, got %s", completed.Status)
		}
	})

	t.Run("init upload for non-existent project fails", func(t *testing.T) {
		_, err := svc.InitUpload(ctx, "missing-project", domain.InitUploadInput{
			OriginalFilename: "clip.mp4",
			MimeType:         "video/mp4",
			FileSize:         5000,
		})
		if !errors.Is(err, domain.ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})
}

func TestMediaService_QueueFailure(t *testing.T) {
	mediaRepo := newMockMediaRepo()
	projectRepo := newMockProjectRepo()
	store := newMockStorage()
	proc := &mockProcessor{}
	queueErr := errors.New("queue capacity exceeded")
	fQueue := &failingQueue{err: queueErr}

	svc := NewMediaService(mediaRepo, projectRepo, store, proc, fQueue)
	ctx := context.Background()

	_ = projectRepo.Create(ctx, &domain.Project{ID: "proj-q", Name: "Queue Test"})
	objectKey := "projects/proj-q/media/m-q/original/clip.mp4"
	store.objects[objectKey] = []byte("video bytes")

	_ = mediaRepo.Create(ctx, &domain.MediaAsset{
		ID:                "m-q",
		ProjectID:         "proj-q",
		OriginalObjectKey: objectKey,
		OriginalFilename:  "clip.mp4",
		Status:            domain.StatusUploading,
	})

	// When enqueue fails, CompleteUpload must return error AND media must transition to FAILED
	completed, err := svc.CompleteUpload(ctx, "proj-q", "m-q")
	if err == nil {
		t.Fatalf("expected error from CompleteUpload when queue fails, got %v", completed)
	}

	asset, getErr := mediaRepo.GetByID(ctx, "m-q")
	if getErr != nil {
		t.Fatalf("failed to get asset: %v", getErr)
	}
	if asset.Status != domain.StatusFailed {
		t.Errorf("expected status FAILED after enqueue failure, got %s", asset.Status)
	}
	if asset.ErrorMessage == nil || *asset.ErrorMessage == "" {
		t.Errorf("expected error message to be set when enqueue fails")
	}
}

func TestMediaService_StartupReconciliation(t *testing.T) {
	mediaRepo := newMockMediaRepo()
	projectRepo := newMockProjectRepo()
	store := newMockStorage()
	proc := &mockProcessor{}

	ctx := context.Background()
	_ = projectRepo.Create(ctx, &domain.Project{ID: "p-rec", Name: "Recon Test"})

	// Create an orphaned PROCESSING asset
	_ = mediaRepo.Create(ctx, &domain.MediaAsset{
		ID:                "m-orphaned",
		ProjectID:         "p-rec",
		OriginalObjectKey: "projects/p-rec/media/m-orphaned/original/take.mp4",
		OriginalFilename:  "take.mp4",
		Status:            domain.StatusProcessing,
	})

	// Create a READY asset that shouldn't be changed
	_ = mediaRepo.Create(ctx, &domain.MediaAsset{
		ID:                "m-ready",
		ProjectID:         "p-rec",
		OriginalObjectKey: "projects/p-rec/media/m-ready/original/take2.mp4",
		OriginalFilename:  "take2.mp4",
		Status:            domain.StatusReady,
	})

	// Initialize service (triggers ReconcileOrphanedProcessing)
	svc := NewMediaService(mediaRepo, projectRepo, store, proc, nil)
	_ = svc

	orphaned, _ := mediaRepo.GetByID(ctx, "m-orphaned")
	if orphaned.Status != domain.StatusFailed {
		t.Errorf("expected orphaned asset to be reconciled to FAILED, got %s", orphaned.Status)
	}
	if orphaned.ErrorMessage == nil || *orphaned.ErrorMessage != "processing interrupted by server restart; retry available" {
		t.Errorf("unexpected error message: %v", orphaned.ErrorMessage)
	}

	ready, _ := mediaRepo.GetByID(ctx, "m-ready")
	if ready.Status != domain.StatusReady {
		t.Errorf("expected ready asset to remain READY, got %s", ready.Status)
	}
}

func TestMediaService_ProcessingPipeline(t *testing.T) {
	svc, mediaRepo, _, store, proc := setupTestMediaService()
	ctx := context.Background()

	mediaID := "media-100"
	objectKey := "projects/p1/media/media-100/original/raw.mp4"
	store.objects[objectKey] = []byte("video file bytes")

	asset := &domain.MediaAsset{
		ID:                mediaID,
		ProjectID:         "p1",
		OriginalObjectKey: objectKey,
		OriginalFilename:  "raw.mp4",
		MimeType:          "video/mp4",
		Status:            domain.StatusProcessing,
	}
	_ = mediaRepo.Create(ctx, asset)

	t.Run("successful processing transitions to READY", func(t *testing.T) {
		err := svc.ProcessMedia(ctx, mediaID)
		if err != nil {
			t.Fatalf("unexpected error processing media: %v", err)
		}

		updated, _ := mediaRepo.GetByID(ctx, mediaID)
		if updated.Status != domain.StatusReady {
			t.Errorf("expected READY status, got %s", updated.Status)
		}
		if updated.Duration != 12.5 || updated.Width != 1920 || updated.Height != 1080 {
			t.Errorf("unexpected metadata: duration=%.2f, res=%dx%d", updated.Duration, updated.Width, updated.Height)
		}
		if updated.ProxyObjectKey == nil {
			t.Errorf("expected proxy object key to be populated")
		}
	})

	t.Run("repeated processing of READY asset is idempotent", func(t *testing.T) {
		err := svc.ProcessMedia(ctx, mediaID)
		if err != nil {
			t.Fatalf("expected nil on repeated processing of READY asset, got %v", err)
		}
	})

	t.Run("failed processing transitions to FAILED with error message", func(t *testing.T) {
		mediaID2 := "media-200"
		key2 := "projects/p1/media/media-200/original/corrupt.mp4"
		store.objects[key2] = []byte("corrupt")
		_ = mediaRepo.Create(ctx, &domain.MediaAsset{
			ID:                mediaID2,
			ProjectID:         "p1",
			OriginalObjectKey: key2,
			OriginalFilename:  "corrupt.mp4",
			Status:            domain.StatusProcessing,
		})

		proc.failInspect = true
		err := svc.ProcessMedia(ctx, mediaID2)
		if err == nil {
			t.Fatalf("expected error from failed inspect")
		}

		updated, _ := mediaRepo.GetByID(ctx, mediaID2)
		if updated.Status != domain.StatusFailed {
			t.Errorf("expected FAILED status, got %s", updated.Status)
		}
		if updated.ErrorMessage == nil || *updated.ErrorMessage == "" {
			t.Errorf("expected error message to be set")
		}
	})
}
