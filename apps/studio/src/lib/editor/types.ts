/**
 * Web Video Editor Types
 * 
 * Core data structures for the timeline-based video editor.
 * Follows freecut architecture: Track[] → Clip[] with playhead tracking.
 */

export type TrackType = 'video' | 'audio' | 'subtitle'

export interface Clip {
  id: string
  trackId: string
  /** Clip start time in milliseconds */
  startTime: number
  /** Clip duration in milliseconds */
  duration: number
  /** Source media URL or data */
  source: string
  /** Trim start in source media (ms) */
  trimStart: number
  /** Trim end in source media (ms) */
  trimEnd: number
  /** Playback speed multiplier (0.25x to 2x) */
  speed: number
  /** Opacity (0-1) for fade effects */
  opacity: number
  /** Display name for UI */
  label: string
}

export interface Track {
  id: string
  type: TrackType
  label: string
  /** Track is visible/audible */
  visible: boolean
  /** Track is locked (no edits allowed) */
  locked: boolean
  /** Clips on this track, sorted by startTime */
  clips: Clip[]
  /** Height in editor (px) - for video/audio visualization */
  height: number
}

export interface TimelineData {
  /** Unique timeline ID */
  id: string
  /** Display name */
  name: string
  /** Total duration (ms) */
  duration: number
  /** Video resolution */
  resolution: {
    width: number
    height: number
  }
  /** Frame rate (fps) */
  frameRate: number
  /** Playback speed profile */
  speed: number
  /** All tracks */
  tracks: Track[]
}

/**
 * Playhead position and selection state
 */
export interface PlayheadState {
  /** Current playhead position (ms) */
  position: number
  /** Whether currently playing */
  isPlaying: boolean
  /** Selected clip ID for editing */
  selectedClipId?: string
  /** Zoom level (pixels per second) */
  zoomLevel: number
}

/**
 * Edit history entry for undo/redo
 */
export interface HistoryEntry {
  id: string
  timestamp: number
  description: string
  state: TimelineData
}

/**
 * Export options for multiple formats
 */
export type ExportFormat = 'mp4' | 'fcpxml' | 'premiere' | 'davinci'

export interface ExportOptions {
  format: ExportFormat
  quality: 'low' | 'medium' | 'high'
  codec?: string
  bitrate?: number
  preset?: string
}

/**
 * Export progress tracking
 */
export interface ExportProgress {
  status: 'idle' | 'processing' | 'complete' | 'error'
  percentage: number
  currentFrame?: number
  totalFrames?: number
  estimatedSecondsRemaining?: number
  error?: string
}
