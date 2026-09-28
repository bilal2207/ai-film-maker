import React, { useState, useEffect } from 'react';
import type { Project } from '../types/project';
import { fetchProjects, createProject, deleteProject } from '../api/projects';

interface ProjectListProps {
  onSelectProject: (project: Project) => void;
}

export function ProjectList({ onSelectProject }: ProjectListProps) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Creation form states
  const [showCreateModal, setShowCreateModal] = useState<boolean>(false);
  const [projectName, setProjectName] = useState<string>('');
  const [projectDesc, setProjectDesc] = useState<string>('');
  const [creating, setCreating] = useState<boolean>(false);
  const [formError, setFormError] = useState<string | null>(null);

  const loadProjects = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchProjects();
      setProjects(data);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to load projects');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProjects();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedName = projectName.trim();
    if (!trimmedName) {
      setFormError('Project name is required');
      return;
    }

    try {
      setCreating(true);
      setFormError(null);
      const created = await createProject({
        name: trimmedName,
        description: projectDesc.trim(),
      });
      setProjects((prev) => [created, ...prev]);
      setProjectName('');
      setProjectDesc('');
      setShowCreateModal(false);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setFormError(err.message);
      } else {
        setFormError('Failed to create project');
      }
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    if (!window.confirm('Are you sure you want to delete this project?')) {
      return;
    }

    try {
      await deleteProject(id);
      setProjects((prev) => prev.filter((p) => p.id !== id));
    } catch (err: unknown) {
      if (err instanceof Error) {
        alert(err.message);
      } else {
        alert('Failed to delete project');
      }
    }
  };

  return (
    <div className="project-list-container">
      <div className="list-header">
        <div>
          <h2 className="section-title">Projects</h2>
          <p className="section-subtitle">Manage and launch your film productions</p>
        </div>
        <button
          type="button"
          className="btn-primary"
          onClick={() => setShowCreateModal(true)}
        >
          + New Project
        </button>
      </div>

      {loading && (
        <div className="state-box">
          <div className="spinner"></div>
          <p>Loading projects...</p>
        </div>
      )}

      {error && !loading && (
        <div className="state-box error-state">
          <p className="error-text">{error}</p>
          <button type="button" className="btn-secondary" onClick={loadProjects}>
            Retry
          </button>
        </div>
      )}

      {!loading && !error && projects.length === 0 && (
        <div className="state-box empty-state">
          <div className="empty-icon">🎬</div>
          <h3>No projects yet</h3>
          <p>Get started by creating your first AI filmmaking project.</p>
          <button
            type="button"
            className="btn-primary"
            onClick={() => setShowCreateModal(true)}
          >
            Create First Project
          </button>
        </div>
      )}

      {!loading && !error && projects.length > 0 && (
        <div className="projects-grid">
          {projects.map((p) => (
            <div
              key={p.id}
              className="project-card"
              onClick={() => onSelectProject(p)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  onSelectProject(p);
                }
              }}
            >
              <div className="card-top">
                <h3 className="project-name">{p.name}</h3>
                <button
                  type="button"
                  className="btn-delete"
                  title="Delete Project"
                  onClick={(e) => handleDelete(e, p.id)}
                >
                  ✕
                </button>
              </div>
              <p className="project-card-desc">
                {p.description || 'No description'}
              </p>
              <div className="card-footer">
                <span className="project-date">
                  {new Date(p.createdAt).toLocaleDateString()}
                </span>
                <span className="open-link">Open Shell →</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create Project Modal */}
      {showCreateModal && (
        <div className="modal-backdrop">
          <div className="modal-dialog">
            <div className="modal-header">
              <h3>Create New Project</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => {
                  setShowCreateModal(false);
                  setFormError(null);
                }}
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleCreate}>
              {formError && <div className="form-error-banner">{formError}</div>}

              <div className="form-group">
                <label htmlFor="projectName">Project Name *</label>
                <input
                  id="projectName"
                  type="text"
                  placeholder="e.g. Neon Horizon"
                  value={projectName}
                  onChange={(e) => setProjectName(e.target.value)}
                  maxLength={255}
                  required
                  autoFocus
                />
              </div>

              <div className="form-group">
                <label htmlFor="projectDesc">Description</label>
                <textarea
                  id="projectDesc"
                  placeholder="Brief synopsis or project logline..."
                  value={projectDesc}
                  onChange={(e) => setProjectDesc(e.target.value)}
                  rows={3}
                />
              </div>

              <div className="modal-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => {
                    setShowCreateModal(false);
                    setFormError(null);
                  }}
                  disabled={creating}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={creating}
                >
                  {creating ? 'Creating...' : 'Create Project'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
