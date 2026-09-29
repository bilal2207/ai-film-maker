import type { MediaAsset } from './media';

export type TrackType = 'VIDEO' | 'AUDIO';

export interface TimelineClip {
  id: string;
  trackId: string;
  mediaAssetId: string;
  timelineStart: number; // Microseconds
  sourceIn: number;      // Microseconds
  sourceOut: number;     // Microseconds
  duration: number;      // Computed microseconds
  timelineEnd: number;   // Computed microseconds
  createdAt: string;
  updatedAt: string;
  media?: MediaAsset;
}

export interface Track {
  id: string;
  timelineId: string;
  type: TrackType;
  name: string;
  trackOrder: number;
  createdAt: string;
  updatedAt: string;
  clips?: TimelineClip[];
}

export interface Timeline {
  id: string;
  projectId: string;
  name: string;
  createdAt: string;
  updatedAt: string;
  tracks?: Track[];
}

export interface CreateTimelineInput {
  name?: string;
}

export interface CreateTrackInput {
  type: TrackType;
  name: string;
  trackOrder?: number;
}

export interface UpdateTrackInput {
  name?: string;
  trackOrder?: number;
}

export interface CreateClipInput {
  mediaAssetId: string;
  timelineStart: number; // Microseconds
  sourceIn: number;      // Microseconds
  sourceOut: number;     // Microseconds
}

export interface UpdateClipInput {
  timelineStart?: number;
  sourceIn?: number;
  sourceOut?: number;
}
