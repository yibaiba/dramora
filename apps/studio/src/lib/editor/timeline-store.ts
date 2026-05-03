/**
 * Video Editor Zustand Store
 * 
 * Manages the global state for the video timeline editor:
 * - Timeline data (tracks, clips)
 * - Playhead position and playback
 * - Selection and UI state
 * - Undo/redo history
 */

import { create } from 'zustand'
import { nanoid } from 'nanoid'
import type { TimelineData, Track, Clip, PlayheadState, HistoryEntry, TrackType, ExportProgress } from './types'

interface EditorStore {
  // Timeline state
  timeline: TimelineData
  playhead: PlayheadState
  exportProgress: ExportProgress

  // History
  history: HistoryEntry[]
  historyIndex: number

  // Actions - Timeline
  initializeTimeline: (data: TimelineData) => void
  addTrack: (type: TrackType, label: string) => void
  removeTrack: (trackId: string) => void
  toggleTrackVisibility: (trackId: string) => void
  toggleTrackLock: (trackId: string) => void

  // Actions - Clips
  addClip: (trackId: string, clip: Omit<Clip, 'id'>) => void
  removeClip: (trackId: string, clipId: string) => void
  updateClip: (trackId: string, clipId: string, updates: Partial<Clip>) => void
  moveClip: (trackId: string, clipId: string, newStartTime: number) => void
  trimClip: (trackId: string, clipId: string, trimStart: number, trimEnd: number) => void

  // Actions - Playhead
  setPlayheadPosition: (position: number) => void
  setIsPlaying: (isPlaying: boolean) => void
  selectClip: (clipId: string | undefined) => void
  setZoomLevel: (level: number) => void

  // Actions - History
  pushHistory: (description: string) => void
  undo: () => void
  redo: () => void

  // Actions - Export
  setExportProgress: (progress: Partial<ExportProgress>) => void
  resetExportProgress: () => void
}

const DEFAULT_EXPORT_PROGRESS: ExportProgress = {
  status: 'idle',
  percentage: 0,
}

export const useEditorStore = create<EditorStore>((set) => ({
  timeline: {
    id: nanoid(),
    name: 'Untitled Timeline',
    duration: 0,
    resolution: { width: 1920, height: 1080 },
    frameRate: 30,
    speed: 1,
    tracks: [],
  },
  playhead: {
    position: 0,
    isPlaying: false,
    zoomLevel: 50, // pixels per second
  },
  exportProgress: { ...DEFAULT_EXPORT_PROGRESS },
  history: [],
  historyIndex: -1,

  // Timeline actions
  initializeTimeline: (data) =>
    set(() => {
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: 'Initialize timeline',
        state: data,
      }
      return {
        timeline: data,
        history: [entry],
        historyIndex: 0,
      }
    }),

  addTrack: () =>
    set((state) => {
      const newTrack: Track = {
        id: nanoid(),
        type: 'video',
        label: `Track ${state.timeline.tracks.length + 1}`,
        visible: true,
        locked: false,
        clips: [],
        height: 100,
      }
      const newTimeline = {
        ...state.timeline,
        tracks: [...state.timeline.tracks, newTrack],
      }
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: `Add ${newTrack.type} track`,
        state: newTimeline,
      }
      return {
        timeline: newTimeline,
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
      }
    }),

  removeTrack: (trackId) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.filter((t) => t.id !== trackId),
      }
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: 'Remove track',
        state: newTimeline,
      }
      return {
        timeline: newTimeline,
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
      }
    }),

  toggleTrackVisibility: (trackId) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) =>
          t.id === trackId ? { ...t, visible: !t.visible } : t,
        ),
      }
      return { timeline: newTimeline }
    }),

  toggleTrackLock: (trackId) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) =>
          t.id === trackId ? { ...t, locked: !t.locked } : t,
        ),
      }
      return { timeline: newTimeline }
    }),

  // Clip actions
  addClip: (trackId, clip) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) => {
          if (t.id !== trackId) return t
          const newClip: Clip = { ...clip, id: nanoid() }
          const newClips = [...t.clips, newClip].sort((a, b) => a.startTime - b.startTime)
          return { ...t, clips: newClips }
        }),
      }
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: 'Add clip',
        state: newTimeline,
      }
      return {
        timeline: newTimeline,
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
      }
    }),

  removeClip: (trackId, clipId) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) => {
          if (t.id !== trackId) return t
          return { ...t, clips: t.clips.filter((c) => c.id !== clipId) }
        }),
      }
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: 'Remove clip',
        state: newTimeline,
      }
      return {
        timeline: newTimeline,
        selectedClipId: state.playhead.selectedClipId === clipId ? undefined : state.playhead.selectedClipId,
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
        playhead: {
          ...state.playhead,
          selectedClipId: state.playhead.selectedClipId === clipId ? undefined : state.playhead.selectedClipId,
        },
      }
    }),

  updateClip: (trackId, clipId, updates) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) => {
          if (t.id !== trackId) return t
          return {
            ...t,
            clips: t.clips.map((c) => (c.id === clipId ? { ...c, ...updates } : c)),
          }
        }),
      }
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: 'Update clip',
        state: newTimeline,
      }
      return {
        timeline: newTimeline,
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
      }
    }),

  moveClip: (trackId, clipId, newStartTime) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) => {
          if (t.id !== trackId) return t
          const clip = t.clips.find((c) => c.id === clipId)
          if (!clip) return t
          const updatedClips = t.clips.map((c) =>
            c.id === clipId ? { ...c, startTime: newStartTime } : c,
          )
          return { ...t, clips: updatedClips.sort((a, b) => a.startTime - b.startTime) }
        }),
      }
      return { timeline: newTimeline }
    }),

  trimClip: (trackId, clipId, trimStart, trimEnd) =>
    set((state) => {
      const newTimeline = {
        ...state.timeline,
        tracks: state.timeline.tracks.map((t) => {
          if (t.id !== trackId) return t
          return {
            ...t,
            clips: t.clips.map((c) =>
              c.id === clipId
                ? { ...c, trimStart, trimEnd, duration: trimEnd - trimStart }
                : c,
            ),
          }
        }),
      }
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description: 'Trim clip',
        state: newTimeline,
      }
      return {
        timeline: newTimeline,
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
      }
    }),

  // Playhead actions
  setPlayheadPosition: (position) =>
    set((state) => ({
      playhead: { ...state.playhead, position },
    })),

  setIsPlaying: (isPlaying) =>
    set((state) => ({
      playhead: { ...state.playhead, isPlaying },
    })),

  selectClip: (clipId) =>
    set((state) => ({
      playhead: { ...state.playhead, selectedClipId: clipId },
    })),

  setZoomLevel: (level) =>
    set((state) => ({
      playhead: { ...state.playhead, zoomLevel: Math.max(10, Math.min(500, level)) },
    })),

  // History actions
  pushHistory: (description) =>
    set((state) => {
      const entry: HistoryEntry = {
        id: nanoid(),
        timestamp: Date.now(),
        description,
        state: state.timeline,
      }
      return {
        history: [...state.history.slice(0, state.historyIndex + 1), entry],
        historyIndex: state.historyIndex + 1,
      }
    }),

  undo: () =>
    set((state) => {
      if (state.historyIndex > 0) {
        const newIndex = state.historyIndex - 1
        return {
          timeline: state.history[newIndex].state,
          historyIndex: newIndex,
        }
      }
      return state
    }),

  redo: () =>
    set((state) => {
      if (state.historyIndex < state.history.length - 1) {
        const newIndex = state.historyIndex + 1
        return {
          timeline: state.history[newIndex].state,
          historyIndex: newIndex,
        }
      }
      return state
    }),

  // Export actions
  setExportProgress: (progress) =>
    set((state) => ({
      exportProgress: { ...state.exportProgress, ...progress },
    })),

  resetExportProgress: () =>
    set(() => ({
      exportProgress: { ...DEFAULT_EXPORT_PROGRESS },
    })),
}))
