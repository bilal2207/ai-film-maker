import { useState } from 'react';
import type { Project } from './types/project';
import { ProjectList } from './components/ProjectList';
import { ProjectShell } from './components/ProjectShell';
import './App.css';

export function App() {
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);

  return (
    <div className="app-layout">
      <header className="app-navbar">
        <div className="brand">
          <span className="brand-icon">🎬</span>
          <h1 className="brand-name">AI Filmmaker</h1>
          <span className="version-badge">Hop 2</span>
        </div>
        <div className="nav-meta">
          <span className="backend-indicator">
            <span className="dot"></span> Connected to Go API
          </span>
        </div>
      </header>

      <main className="app-main">
        {selectedProject ? (
          <ProjectShell
            project={selectedProject}
            onBack={() => setSelectedProject(null)}
          />
        ) : (
          <ProjectList onSelectProject={(project) => setSelectedProject(project)} />
        )}
      </main>
    </div>
  );
}

export default App;
