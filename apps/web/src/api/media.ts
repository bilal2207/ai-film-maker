import type { InitUploadInput, InitUploadOutput, MediaAsset } from '../types/media';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export async function initMediaUpload(
  projectId: string,
  input: InitUploadInput
): Promise<InitUploadOutput> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/media/upload`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to initialize media upload (status ${response.status})`);
  }

  return response.json();
}

export async function uploadFileToStorage(
  uploadUrl: string,
  file: File,
  onProgress?: (progressPercent: number) => void
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', uploadUrl, true);
    xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream');

    if (xhr.upload && onProgress) {
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable) {
          const percent = Math.round((event.loaded / event.total) * 100);
          onProgress(percent);
        }
      };
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else {
        reject(new Error(`Storage upload failed with status ${xhr.status}`));
      }
    };

    xhr.onerror = () => {
      reject(new Error('Network error during file upload to storage'));
    };

    xhr.send(file);
  });
}

export async function completeMediaUpload(
  projectId: string,
  mediaId: string
): Promise<MediaAsset> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/media/${mediaId}/complete`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to complete media upload (status ${response.status})`);
  }

  return response.json();
}

export async function listProjectMedia(projectId: string): Promise<MediaAsset[]> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/media`, {
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to fetch media assets (status ${response.status})`);
  }

  return response.json();
}

export async function getMediaAsset(projectId: string, mediaId: string): Promise<MediaAsset> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/media/${mediaId}`, {
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to fetch media asset (status ${response.status})`);
  }

  return response.json();
}

export async function deleteMediaAsset(projectId: string, mediaId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/projects/${projectId}/media/${mediaId}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(errorData?.error?.message || `Failed to delete media asset (status ${response.status})`);
  }
}
