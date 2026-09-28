/**
 * Editor Core Abstractions (Foundation)
 * Provides interfaces for future NLE / Timeline / WebCodecs / WebGPU pipeline.
 */

export interface TimelineTrack {
  id: string;
  type: 'video' | 'audio' | 'overlay';
  name: string;
}

export interface TimelineState {
  currentTime: number;
  duration: number;
  isPlaying: boolean;
  tracks: TimelineTrack[];
}

export function createInitialTimelineState(): TimelineState {
  return {
    currentTime: 0,
    duration: 0,
    isPlaying: false,
    tracks: [],
  };
}
