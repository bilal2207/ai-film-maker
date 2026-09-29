import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { ProjectShell } from './ProjectShell';
import * as mediaApi from '../api/media';
import type { Project } from '../types/project';
import type { MediaAsset } from '../types/media';

vi.mock('../api/media');

const mockProject: Project = {
  id: 'proj-123',
  name: 'Quantum Horizon',
  description: 'Deep space anomaly',
  createdAt: '2026-09-29T00:00:00.000Z',
  updatedAt: '2026-09-29T00:00:00.000Z',
};

const mockReadyMedia: MediaAsset = {
  id: 'media-1',
  projectId: 'proj-123',
  originalObjectKey: 'projects/proj-123/media/raw/take_01.mp4',
  proxyObjectKey: 'projects/proj-123/media/proxies/proxy_take_01.mp4',
  thumbnailObjectKey: 'projects/proj-123/media/thumbnails/thumb_take_01.jpg',
  originalFilename: 'take_01.mp4',
  mimeType: 'video/mp4',
  fileSize: 10485760, // 10 MB
  duration: 45.2,
  width: 1920,
  height: 1080,
  fps: 24.0,
  status: 'READY',
  createdAt: '2026-09-29T00:00:00.000Z',
  updatedAt: '2026-09-29T00:00:00.000Z',
  proxyUrl: 'http://localhost:8080/storage/download/projects/proj-123/media/proxies/proxy_take_01.mp4',
  thumbnailUrl: 'http://localhost:8080/storage/download/projects/proj-123/media/thumbnails/thumb_take_01.jpg',
};

const mockProcessingMedia: MediaAsset = {
  id: 'media-2',
  projectId: 'proj-123',
  originalObjectKey: 'projects/proj-123/media/raw/take_02.mp4',
  originalFilename: 'take_02.mp4',
  mimeType: 'video/mp4',
  fileSize: 20971520,
  status: 'PROCESSING',
  createdAt: '2026-09-29T00:00:00.000Z',
  updatedAt: '2026-09-29T00:00:00.000Z',
};

const mockFailedMedia: MediaAsset = {
  id: 'media-3',
  projectId: 'proj-123',
  originalObjectKey: 'projects/proj-123/media/raw/corrupt.mp4',
  originalFilename: 'corrupt.mp4',
  mimeType: 'video/mp4',
  fileSize: 1000,
  status: 'FAILED',
  errorMessage: 'Corrupt video stream detected',
  createdAt: '2026-09-29T00:00:00.000Z',
  updatedAt: '2026-09-29T00:00:00.000Z',
};

describe('ProjectShell Component', () => {
  const onBackMock = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders project details and empty media state when no assets exist', async () => {
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([]);

    render(<ProjectShell project={mockProject} onBack={onBackMock} />);

    expect(screen.getByText('Quantum Horizon')).toBeInTheDocument();
    expect(screen.getByText('Deep space anomaly')).toBeInTheDocument();
    expect(screen.getByText('Hop 2 — Media Ingestion')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('No footage uploaded yet')).toBeInTheDocument();
    });
  });

  it('renders media cards with READY, PROCESSING, and FAILED states', async () => {
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([
      mockReadyMedia,
      mockProcessingMedia,
      mockFailedMedia,
    ]);

    render(<ProjectShell project={mockProject} onBack={onBackMock} />);

    await waitFor(() => {
      expect(screen.getByText('take_01.mp4')).toBeInTheDocument();
      expect(screen.getByText('take_02.mp4')).toBeInTheDocument();
      expect(screen.getByText('corrupt.mp4')).toBeInTheDocument();
    });

    // Check statuses
    expect(screen.getByText('READY')).toBeInTheDocument();
    expect(screen.getByText('PROCESSING')).toBeInTheDocument();
    expect(screen.getByText('FAILED')).toBeInTheDocument();
    expect(screen.getByText('Corrupt video stream detected')).toBeInTheDocument();
  });

  it('opens and closes proxy preview modal', async () => {
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([mockReadyMedia]);

    render(<ProjectShell project={mockProject} onBack={onBackMock} />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /preview proxy/i })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /preview proxy/i }));

    expect(screen.getByText('take_01.mp4 (720p Proxy)')).toBeInTheDocument();
    expect(screen.getByText('Resolution: 1920x1080')).toBeInTheDocument();

    // Close modal
    fireEvent.click(screen.getByRole('button', { name: '✕' }));
    expect(screen.queryByText('take_01.mp4 (720p Proxy)')).not.toBeInTheDocument();
  });

  it('handles video ingestion upload flow', async () => {
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([]);
    vi.mocked(mediaApi.initMediaUpload).mockResolvedValueOnce({
      mediaId: 'new-media-id',
      uploadUrl: 'http://localhost:8080/storage/upload/raw/take_03.mp4',
      objectKey: 'raw/take_03.mp4',
      expiresAt: '2026-09-29T01:00:00Z',
    });
    vi.mocked(mediaApi.uploadFileToStorage).mockResolvedValueOnce(undefined);
    vi.mocked(mediaApi.completeMediaUpload).mockResolvedValueOnce({
      ...mockProcessingMedia,
      id: 'new-media-id',
      originalFilename: 'take_03.mp4',
    });
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([
      { ...mockProcessingMedia, id: 'new-media-id', originalFilename: 'take_03.mp4' },
    ]);

    render(<ProjectShell project={mockProject} onBack={onBackMock} />);

    await waitFor(() => {
      expect(screen.getByText('No footage uploaded yet')).toBeInTheDocument();
    });

    const fileInput = document.getElementById('media-file-input') as HTMLInputElement;
    const testFile = new File(['dummy video content'], 'take_03.mp4', { type: 'video/mp4' });

    fireEvent.change(fileInput, { target: { files: [testFile] } });

    await waitFor(() => {
      expect(mediaApi.initMediaUpload).toHaveBeenCalledWith('proj-123', {
        filename: 'take_03.mp4',
        mimeType: 'video/mp4',
        fileSize: testFile.size,
      });
      expect(mediaApi.uploadFileToStorage).toHaveBeenCalled();
      expect(mediaApi.completeMediaUpload).toHaveBeenCalledWith('proj-123', 'new-media-id');
    });
  });

  it('deletes a media asset after user confirmation', async () => {
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([mockReadyMedia]);
    vi.mocked(mediaApi.deleteMediaAsset).mockResolvedValueOnce(undefined);
    vi.spyOn(window, 'confirm').mockReturnValue(true);

    render(<ProjectShell project={mockProject} onBack={onBackMock} />);

    await waitFor(() => {
      expect(screen.getByText('take_01.mp4')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTitle('Delete asset'));

    await waitFor(() => {
      expect(mediaApi.deleteMediaAsset).toHaveBeenCalledWith('proj-123', 'media-1');
      expect(screen.queryByText('take_01.mp4')).not.toBeInTheDocument();
    });
  });
});
