import type {
  CreateClipInput,
  CreateTimelineInput,
  CreateTrackInput,
  Timeline,
  TimelineClip,
  Track,
  UpdateClipInput,
  UpdateTrackInput,
} from '../types/timeline';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export async function fetchTimeline(projectId: string): Promise<Timeline> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/timeline`, {
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to fetch timeline (status ${response.status})`);
  }

  return response.json();
}

export async function createTimeline(
  projectId: string,
  input: CreateTimelineInput
): Promise<Timeline> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/timeline`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to create timeline (status ${response.status})`);
  }

  return response.json();
}

export async function createTrack(
  projectId: string,
  input: CreateTrackInput
): Promise<Track> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/timeline/tracks`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to create track (status ${response.status})`);
  }

  return response.json();
}

export async function updateTrack(
  projectId: string,
  trackId: string,
  input: UpdateTrackInput
): Promise<Track> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/timeline/tracks/${trackId}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to update track (status ${response.status})`);
  }

  return response.json();
}

export async function deleteTrack(projectId: string, trackId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/timeline/tracks/${trackId}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to delete track (status ${response.status})`);
  }
}

export async function createClip(
  projectId: string,
  trackId: string,
  input: CreateClipInput
): Promise<TimelineClip> {
  const response = await fetch(
    `${API_BASE_URL}/projects/${projectId}/timeline/tracks/${trackId}/clips`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify(input),
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to create clip (status ${response.status})`);
  }

  return response.json();
}

export async function updateClip(
  projectId: string,
  trackId: string,
  clipId: string,
  input: UpdateClipInput
): Promise<TimelineClip> {
  const response = await fetch(
    `${API_BASE_URL}/projects/${projectId}/timeline/tracks/${trackId}/clips/${clipId}`,
    {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify(input),
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to update clip (status ${response.status})`);
  }

  return response.json();
}

export async function deleteClip(
  projectId: string,
  trackId: string,
  clipId: string
): Promise<void> {
  const response = await fetch(
    `${API_BASE_URL}/projects/${projectId}/timeline/tracks/${trackId}/clips/${clipId}`,
    {
      method: 'DELETE',
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to delete clip (status ${response.status})`);
  }
}
