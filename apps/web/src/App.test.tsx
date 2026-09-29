import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import App from './App';
import * as projectApi from './api/projects';
import * as mediaApi from './api/media';
import type { Project } from './types/project';

vi.mock('./api/projects');
vi.mock('./api/media');

const mockProject: Project = {
  id: 'proj-123',
  name: 'Quantum Horizon',
  description: 'Deep space anomaly',
  createdAt: '2026-09-29T00:00:00.000Z',
  updatedAt: '2026-09-29T00:00:00.000Z',
};

describe('App Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders navbar and loads project list', async () => {
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce([mockProject]);

    render(<App />);

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('AI Filmmaker');
    expect(screen.getByText('Connected to Go API')).toBeInTheDocument();
    expect(screen.getByText('Hop 2')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Quantum Horizon')).toBeInTheDocument();
    });
  });

  it('transitions to ProjectShell when selecting a project and navigates back', async () => {
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce([mockProject]);
    vi.mocked(mediaApi.listProjectMedia).mockResolvedValueOnce([]);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText('Quantum Horizon')).toBeInTheDocument();
    });

    // Open project
    fireEvent.click(screen.getByText('Quantum Horizon'));

    // ProjectShell view is active
    expect(screen.getByRole('button', { name: /← back to projects/i })).toBeInTheDocument();
    expect(screen.getByText('Hop 2 — Media Ingestion')).toBeInTheDocument();
    expect(screen.getByText('proj-123')).toBeInTheDocument();

    // Navigate back
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce([mockProject]);
    fireEvent.click(screen.getByRole('button', { name: /← back to projects/i }));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /\+ new project/i })).toBeInTheDocument();
    });
  });
});
