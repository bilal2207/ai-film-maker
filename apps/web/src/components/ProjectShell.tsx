import { useCallback, useEffect, useRef, useState } from 'react';
import type { Project } from '../types/project';
import type { MediaAsset } from '../types/media';
import {
  initMediaUpload,
  uploadFileToStorage,
  completeMediaUpload,
  listProjectMedia,
  deleteMediaAsset,
} from '../api/media';

interface ProjectShellProps {
  project: Project;
  onBack: () => void;
}

interface UploadProgressState {
  filename: string;
  progressPercent: number;
  stage: 'initializing' | 'uploading' | 'completing' | 'processing';
  error?: string;
}

export function ProjectShell({ project, onBack }: ProjectShellProps) {
  const [mediaList, setMediaList] = useState<MediaAsset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [uploadState, setUploadState] = useState<UploadProgressState | null>(null);
  const [selectedPreview, setSelectedPreview] = useState<MediaAsset | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const loadMedia = useCallback(async () => {
    try {
      const data = await listProjectMedia(project.id);
      setMediaList(data);
      setError(null);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to load media assets';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [project.id]);

  useEffect(() => {
    loadMedia();
  }, [loadMedia]);

  // Poll when any asset is in UPLOADING or PROCESSING status
  useEffect(() => {
    const hasPending = mediaList.some(
      (m) => m.status === 'UPLOADING' || m.status === 'PROCESSING'
    );

    if (hasPending) {
      pollTimerRef.current = setTimeout(() => {
        loadMedia();
      }, 2500);
    }

    return () => {
      if (pollTimerRef.current) {
        clearTimeout(pollTimerRef.current);
      }
    };
  }, [mediaList, loadMedia]);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Reset input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }

    setUploadState({
      filename: file.name,
      progressPercent: 0,
      stage: 'initializing',
    });

    try {
      // 1. Initialize upload
      const initResp = await initMediaUpload(project.id, {
        filename: file.name,
        mimeType: file.type || 'video/mp4',
        fileSize: file.size,
      });

      // 2. Upload to storage
      setUploadState({
        filename: file.name,
        progressPercent: 0,
        stage: 'uploading',
      });

      await uploadFileToStorage(initResp.uploadUrl, file, (percent) => {
        setUploadState((prev) => (prev ? { ...prev, progressPercent: percent } : null));
      });

      // 3. Complete upload
      setUploadState({
        filename: file.name,
        progressPercent: 100,
        stage: 'completing',
      });

      await completeMediaUpload(project.id, initResp.mediaId);

      setUploadState({
        filename: file.name,
        progressPercent: 100,
        stage: 'processing',
      });

      // Refresh list
      await loadMedia();

      // Clear upload progress banner after brief delay
      setTimeout(() => {
        setUploadState(null);
      }, 2000);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Upload failed';
      setUploadState((prev) => (prev ? { ...prev, error: message } : null));
    }
  };

  const handleDeleteMedia = async (mediaId: string, filename: string) => {
    if (!window.confirm(`Are you sure you want to delete "${filename}"?`)) {
      return;
    }

    try {
      await deleteMediaAsset(project.id, mediaId);
      setMediaList((prev) => prev.filter((m) => m.id !== mediaId));
      if (selectedPreview?.id === mediaId) {
        setSelectedPreview(null);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to delete media';
      alert(message);
    }
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds || seconds <= 0) return '--:--';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`;
    }
    if (bytes < 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  return (
    <div className="project-shell">
      <div className="shell-header">
        <button type="button" className="btn-secondary" onClick={onBack}>
          ← Back to Projects
        </button>
        <div className="shell-status-badge">Hop 2 — Media Ingestion</div>
      </div>

      <div className="shell-content">
        <div className="shell-hero">
          <h2 className="shell-title">{project.name}</h2>
          <p className="shell-description">
            {project.description || 'No description provided.'}
          </p>
        </div>

        {/* Media Ingestion Section */}
        <div className="media-section">
          <div className="media-section-header">
            <div>
              <h3 className="section-title">Project Footage & Media</h3>
              <p className="section-subtitle">
                Upload video takes, clips, and raw footage for background proxy generation and metadata extraction.
              </p>
            </div>
            <div>
              <input
                ref={fileInputRef}
                type="file"
                id="media-file-input"
                accept="video/mp4,video/quicktime,video/webm,video/x-matroska,.mp4,.mov,.webm,.mkv"
                style={{ display: 'none' }}
                onChange={handleFileSelect}
              />
              <button
                type="button"
                className="btn-primary"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploadState !== null && !uploadState.error}
              >
                + Ingest Video
              </button>
            </div>
          </div>

          {/* Active Upload Card */}
          {uploadState && (
            <div className={`upload-card ${uploadState.error ? 'upload-error' : ''}`}>
              <div className="upload-card-info">
                <span className="upload-filename">{uploadState.filename}</span>
                <span className="upload-stage">
                  {uploadState.error
                    ? 'Upload Failed'
                    : uploadState.stage === 'initializing'
                    ? 'Initializing presigned URL...'
                    : uploadState.stage === 'uploading'
                    ? `Uploading... ${uploadState.progressPercent}%`
                    : uploadState.stage === 'completing'
                    ? 'Finalizing upload...'
                    : 'Enqueued for background processing...'}
                </span>
              </div>
              {!uploadState.error ? (
                <div className="progress-bar-bg">
                  <div
                    className="progress-bar-fill"
                    style={{ width: `${uploadState.progressPercent}%` }}
                  />
                </div>
              ) : (
                <div className="upload-error-msg">
                  <span>{uploadState.error}</span>
                  <button
                    type="button"
                    className="btn-small"
                    onClick={() => setUploadState(null)}
                  >
                    Dismiss
                  </button>
                </div>
              )}
            </div>
          )}

          {error && (
            <div className="error-banner">
              <span>{error}</span>
              <button type="button" onClick={loadMedia} className="btn-small">
                Retry
              </button>
            </div>
          )}

          {loading ? (
            <div className="loading-state">Loading media assets...</div>
          ) : mediaList.length === 0 ? (
            <div className="empty-media-card">
              <div className="empty-media-icon">🎥</div>
              <h4>No footage uploaded yet</h4>
              <p>Click "Ingest Video" above to upload your first clip.</p>
            </div>
          ) : (
            <div className="media-grid">
              {mediaList.map((media) => (
                <div key={media.id} className="media-card" data-testid={`media-card-${media.id}`}>
                  <div className="media-thumbnail-container">
                    {media.thumbnailUrl ? (
                      <img
                        src={media.thumbnailUrl}
                        alt={media.originalFilename}
                        className="media-thumbnail-img"
                      />
                    ) : (
                      <div className="media-thumbnail-placeholder">
                        {media.status === 'PROCESSING' ? '⏳' : '🎬'}
                      </div>
                    )}
                    <span className={`status-pill status-${media.status.toLowerCase()}`}>
                      {media.status}
                    </span>
                    {media.duration !== undefined && media.duration > 0 && (
                      <span className="duration-badge">{formatDuration(media.duration)}</span>
                    )}
                  </div>

                  <div className="media-card-body">
                    <h4 className="media-card-title" title={media.originalFilename}>
                      {media.originalFilename}
                    </h4>

                    <div className="media-specs">
                      {media.status === 'READY' ? (
                        <>
                          <span>
                            {media.width && media.height ? `${media.width}x${media.height}` : ''}
                          </span>
                          {media.fps ? <span>{media.fps.toFixed(1)} fps</span> : null}
                          <span>{formatFileSize(media.fileSize)}</span>
                        </>
                      ) : media.status === 'PROCESSING' ? (
                        <span className="text-muted">Extracting metadata & rendering proxy...</span>
                      ) : media.status === 'FAILED' ? (
                        <span className="text-danger" title={media.errorMessage}>
                          {media.errorMessage || 'Processing failed'}
                        </span>
                      ) : (
                        <span className="text-muted">Awaiting completion</span>
                      )}
                    </div>

                    <div className="media-card-actions">
                      {media.status === 'READY' && media.proxyUrl && (
                        <button
                          type="button"
                          className="btn-action-preview"
                          onClick={() => setSelectedPreview(media)}
                        >
                          ▶ Preview Proxy
                        </button>
                      )}
                      <button
                        type="button"
                        className="btn-action-delete"
                        onClick={() => handleDeleteMedia(media.id, media.originalFilename)}
                        title="Delete asset"
                      >
                        🗑
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Video Preview Modal */}
        {selectedPreview && (
          <div className="preview-modal-overlay" onClick={() => setSelectedPreview(null)}>
            <div className="preview-modal-content" onClick={(e) => e.stopPropagation()}>
              <div className="preview-modal-header">
                <h3>{selectedPreview.originalFilename} (720p Proxy)</h3>
                <button
                  type="button"
                  className="modal-close-btn"
                  onClick={() => setSelectedPreview(null)}
                >
                  ✕
                </button>
              </div>
              <div className="preview-modal-player">
                {selectedPreview.proxyUrl ? (
                  <video
                    src={selectedPreview.proxyUrl}
                    controls
                    autoPlay
                    className="proxy-video-element"
                  />
                ) : (
                  <div className="no-proxy-msg">Proxy URL unavailable</div>
                )}
              </div>
              <div className="preview-modal-details">
                <span>Duration: {formatDuration(selectedPreview.duration)}</span>
                <span>
                  Resolution: {selectedPreview.width}x{selectedPreview.height}
                </span>
                <span>FPS: {selectedPreview.fps?.toFixed(2)}</span>
                <span>MIME: {selectedPreview.mimeType}</span>
              </div>
            </div>
          </div>
        )}

        {/* Project Meta Card */}
        <div className="shell-meta-card">
          <h3>Project Details</h3>
          <div className="meta-grid">
            <div className="meta-item">
              <span className="meta-label">Project ID</span>
              <span className="meta-value font-mono">{project.id}</span>
            </div>
            <div className="meta-item">
              <span className="meta-label">Created At</span>
              <span className="meta-value">
                {new Date(project.createdAt).toLocaleString()}
              </span>
            </div>
            <div className="meta-item">
              <span className="meta-label">Last Updated</span>
              <span className="meta-value">
                {new Date(project.updatedAt).toLocaleString()}
              </span>
            </div>
            <div className="meta-item">
              <span className="meta-label">Media Assets</span>
              <span className="meta-value text-emerald">{mediaList.length} files</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
