export type MediaStatus = 'UPLOADING' | 'PROCESSING' | 'READY' | 'FAILED';

export interface MediaAsset {
  id: string;
  projectId: string;
  originalObjectKey: string;
  proxyObjectKey?: string;
  thumbnailObjectKey?: string;
  originalFilename: string;
  mimeType: string;
  fileSize: number;
  duration?: number;
  width?: number;
  height?: number;
  fps?: number;
  status: MediaStatus;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  proxyUrl?: string;
  thumbnailUrl?: string;
}

export interface InitUploadInput {
  filename: string;
  mimeType: string;
  fileSize: number;
}

export interface InitUploadOutput {
  mediaId: string;
  uploadUrl: string;
  objectKey: string;
  expiresAt: string;
}
