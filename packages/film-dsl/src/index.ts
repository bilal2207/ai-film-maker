/**
 * Film DSL Core Type Definitions (Foundation)
 * Defines the contract between AI decision outputs and Editor Core inputs.
 */

export interface FilmProject {
  id: string;
  title: string;
  createdAt: string;
  updatedAt: string;
}

export interface Scene {
  id: string;
  projectId: string;
  sequenceNumber: number;
  heading: string;
  summary?: string;
}

export interface Shot {
  id: string;
  sceneId: string;
  shotNumber: number;
  description: string;
  durationSeconds: number;
  cameraPrompt?: string;
}
