import type { CreateProjectInput, Project } from '../types/project';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export async function fetchProjects(): Promise<Project[]> {
  const response = await fetch(`${API_BASE_URL}/projects`, {
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to fetch projects (status ${response.status})`);
  }

  return response.json();
}

export async function fetchProject(id: string): Promise<Project> {
  const response = await fetch(`${API_BASE_URL}/projects/${id}`, {
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to fetch project (status ${response.status})`);
  }

  return response.json();
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
  const response = await fetch(`${API_BASE_URL}/projects`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to create project (status ${response.status})`);
  }

  return response.json();
}

export async function deleteProject(id: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/projects/${id}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to delete project (status ${response.status})`);
  }
}
