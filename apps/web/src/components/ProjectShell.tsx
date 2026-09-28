import type { Project } from '../types/project';

interface ProjectShellProps {
  project: Project;
  onBack: () => void;
}

export function ProjectShell({ project, onBack }: ProjectShellProps) {
  return (
    <div className="project-shell">
      <div className="shell-header">
        <button type="button" className="btn-secondary" onClick={onBack}>
          ← Back to Projects
        </button>
        <div className="shell-status-badge">Hop 1 Shell</div>
      </div>

      <div className="shell-content">
        <div className="shell-hero">
          <h2 className="shell-title">{project.name}</h2>
          <p className="shell-description">
            {project.description || 'No description provided.'}
          </p>
        </div>

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
              <span className="meta-label">Status</span>
              <span className="meta-value text-emerald">Initialized</span>
            </div>
          </div>
        </div>

        <div className="shell-placeholder-note">
          <p>
            🎬 <strong>Film Pipeline Ready:</strong> Scripts, Storyboards, and NLE Editor modules will attach to this shell in upcoming hops.
          </p>
        </div>
      </div>
    </div>
  );
}
