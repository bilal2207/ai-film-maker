import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { ProjectList } from './ProjectList';
import * as projectApi from '../api/projects';
import type { Project } from '../types/project';

vi.mock('../api/projects');

const mockProjects: Project[] = [
  {
    id: 'proj-1',
    name: 'Cyberpunk 2099',
    description: 'Neon futuristic film',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'proj-2',
    name: 'Desert Mirage',
    description: 'Post-apocalyptic journey',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

describe('ProjectList Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state initially and then lists projects', async () => {
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce(mockProjects);
    const onSelect = vi.fn();

    render(<ProjectList onSelectProject={onSelect} />);

    expect(screen.getByText(/loading projects/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Cyberpunk 2099')).toBeInTheDocument();
      expect(screen.getByText('Desert Mirage')).toBeInTheDocument();
    });
  });

  it('renders empty state when no projects exist', async () => {
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce([]);
    const onSelect = vi.fn();

    render(<ProjectList onSelectProject={onSelect} />);

    await waitFor(() => {
      expect(screen.getByText('No projects yet')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /create first project/i })).toBeInTheDocument();
    });
  });

  it('opens modal and creates a new project', async () => {
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce([]);
    const newProject: Project = {
      id: 'proj-3',
      name: 'New Solaris',
      description: 'Cosmic mystery',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    vi.mocked(projectApi.createProject).mockResolvedValueOnce(newProject);
    const onSelect = vi.fn();

    render(<ProjectList onSelectProject={onSelect} />);

    await waitFor(() => {
      expect(screen.getByText('No projects yet')).toBeInTheDocument();
    });

    // Open modal
    fireEvent.click(screen.getByRole('button', { name: /\+ new project/i }));

    expect(screen.getByText('Create New Project')).toBeInTheDocument();

    // Fill form
    fireEvent.change(screen.getByLabelText(/project name/i), {
      target: { value: 'New Solaris' },
    });
    fireEvent.change(screen.getByLabelText(/description/i), {
      target: { value: 'Cosmic mystery' },
    });

    // Submit
    fireEvent.click(screen.getByRole('button', { name: /^create project$/i }));

    await waitFor(() => {
      expect(projectApi.createProject).toHaveBeenCalledWith({
        name: 'New Solaris',
        description: 'Cosmic mystery',
      });
      expect(screen.getByText('New Solaris')).toBeInTheDocument();
    });
  });

  it('calls onSelectProject when clicking a project card', async () => {
    vi.mocked(projectApi.fetchProjects).mockResolvedValueOnce(mockProjects);
    const onSelect = vi.fn();

    render(<ProjectList onSelectProject={onSelect} />);

    await waitFor(() => {
      expect(screen.getByText('Cyberpunk 2099')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('Cyberpunk 2099'));
    expect(onSelect).toHaveBeenCalledWith(mockProjects[0]);
  });
});
