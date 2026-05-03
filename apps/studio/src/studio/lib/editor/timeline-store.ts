import { create } from 'zustand'
import type { Timeline, Track, Clip, EditHistoryItem } from './types'

const MAX_HISTORY = 50

export type TimelineStore = {
  // Timeline data
  timeline: Timeline
  currentClip: Clip | null
  playheadTime: number // milliseconds
  selectedTrackId: string | null

  // History
  history: EditHistoryItem[]
  historyIndex: number

  // State setters
  setTimeline: (timeline: Timeline) => void
  setCurrentClip: (clip: Clip | null) => void
  setPlayheadTime: (time: number) => void
  selectTrack: (trackId: string | null) => void

  // Clip operations
  addClip: (clip: Clip) => void
  removeClip: (clipId: string) => void
  moveClip: (clipId: string, newStartTime: number) => void
  trimClip: (clipId: string, newStartTime: number, newDuration: number) => void
  updateClipProperties: (clipId: string, properties: Partial<Clip['properties']>) => void

  // Track operations
  addTrack: (track: Track) => void
  removeTrack: (trackId: string) => void
  toggleTrackVisibility: (trackId: string) => void
  toggleTrackLock: (trackId: string) => void

  // History management
  undo: () => void
  redo: () => void
  canUndo: () => boolean
  canRedo: () => boolean

  // Reset
  reset: () => void
  initializeTimeline: (timeline: Timeline) => void
}

const createEmptyTimeline = (): Timeline => ({
  tracks: [],
  duration: 0,
  fps: 30,
})

const createHistoryItem = (action: EditHistoryItem['action'], before: Timeline, after: Timeline): EditHistoryItem => ({
  action,
  beforeState: JSON.parse(JSON.stringify(before)),
  afterState: JSON.parse(JSON.stringify(after)),
  timestamp: Date.now(),
})

export const useTimelineStore = create<TimelineStore>((set, get) => ({
  timeline: createEmptyTimeline(),
  currentClip: null,
  playheadTime: 0,
  selectedTrackId: null,
  history: [],
  historyIndex: -1,

  setTimeline: (timeline) => set({ timeline }),

  setCurrentClip: (clip) => set({ currentClip: clip }),

  setPlayheadTime: (time) => {
    const { timeline } = get()
    const clamped = Math.max(0, Math.min(time, timeline.duration))
    set({ playheadTime: clamped })
  },

  selectTrack: (trackId) => set({ selectedTrackId: trackId }),

  addClip: (clip) => {
    const { timeline, history, historyIndex } = get()

    const trackIndex = timeline.tracks.findIndex((t) => t.id === clip.trackId)
    if (trackIndex === -1) return

    const beforeState = JSON.parse(JSON.stringify(timeline))
    const newTracks = JSON.parse(JSON.stringify(timeline.tracks))
    newTracks[trackIndex].clips.push(clip)
    const newDuration = Math.max(
      timeline.duration,
      Math.max(...newTracks.flatMap((t: Track) => t.clips.map((c: Clip) => c.startTime + c.duration))),
    )

    const afterState: Timeline = {
      ...timeline,
      tracks: newTracks,
      duration: newDuration,
    }

    const newHistory = history.slice(0, historyIndex + 1)
    newHistory.push(createHistoryItem('add_clip', beforeState, afterState))

    set({
      timeline: afterState,
      history: newHistory.slice(-MAX_HISTORY),
      historyIndex: newHistory.length - 1,
    })
  },

  removeClip: (clipId) => {
    const { timeline, history, historyIndex } = get()

    const beforeState = JSON.parse(JSON.stringify(timeline))
    const newTracks = JSON.parse(JSON.stringify(timeline.tracks))

    for (const track of newTracks) {
      track.clips = track.clips.filter((c: Clip) => c.id !== clipId)
    }

    const afterState: Timeline = {
      ...timeline,
      tracks: newTracks,
    }

    const newHistory = history.slice(0, historyIndex + 1)
    newHistory.push(createHistoryItem('remove_clip', beforeState, afterState))

    set({
      timeline: afterState,
      history: newHistory.slice(-MAX_HISTORY),
      historyIndex: newHistory.length - 1,
      currentClip: null,
    })
  },

  moveClip: (clipId, newStartTime) => {
    const { timeline, history, historyIndex } = get()

    const beforeState = JSON.parse(JSON.stringify(timeline))
    const newTracks = JSON.parse(JSON.stringify(timeline.tracks))

    let found = false
    for (const track of newTracks) {
      const clip = track.clips.find((c: Clip) => c.id === clipId)
      if (clip) {
        clip.startTime = Math.max(0, newStartTime)
        found = true
        break
      }
    }

    if (!found) return

    const newDuration = Math.max(
      timeline.duration,
      Math.max(...newTracks.flatMap((t: Track) => t.clips.map((c: Clip) => c.startTime + c.duration))),
    )

    const afterState: Timeline = {
      ...timeline,
      tracks: newTracks,
      duration: newDuration,
    }

    const newHistory = history.slice(0, historyIndex + 1)
    newHistory.push(createHistoryItem('move_clip', beforeState, afterState))

    set({
      timeline: afterState,
      history: newHistory.slice(-MAX_HISTORY),
      historyIndex: newHistory.length - 1,
    })
  },

  trimClip: (clipId, newStartTime, newDuration) => {
    const { timeline, history, historyIndex } = get()

    const beforeState = JSON.parse(JSON.stringify(timeline))
    const newTracks = JSON.parse(JSON.stringify(timeline.tracks))

    let found = false
    for (const track of newTracks) {
      const clip = track.clips.find((c: Clip) => c.id === clipId)
      if (clip) {
        clip.startTime = Math.max(0, newStartTime)
        clip.duration = Math.max(0, newDuration)
        found = true
        break
      }
    }

    if (!found) return

    const newDuration_ = Math.max(
      timeline.duration,
      Math.max(...newTracks.flatMap((t: Track) => t.clips.map((c: Clip) => c.startTime + c.duration))),
    )

    const afterState: Timeline = {
      ...timeline,
      tracks: newTracks,
      duration: newDuration_,
    }

    const newHistory = history.slice(0, historyIndex + 1)
    newHistory.push(createHistoryItem('trim_clip', beforeState, afterState))

    set({
      timeline: afterState,
      history: newHistory.slice(-MAX_HISTORY),
      historyIndex: newHistory.length - 1,
    })
  },

  updateClipProperties: (clipId, properties) => {
    const { timeline } = get()
    const newTracks = JSON.parse(JSON.stringify(timeline.tracks))

    for (const track of newTracks) {
      const clip = track.clips.find((c: Clip) => c.id === clipId)
      if (clip) {
        clip.properties = { ...clip.properties, ...properties }
        break
      }
    }

    set({ timeline: { ...timeline, tracks: newTracks } })
  },

  addTrack: (track) => {
    const { timeline, history, historyIndex } = get()

    const beforeState = JSON.parse(JSON.stringify(timeline))
    const newTracks = [...timeline.tracks, track]

    const afterState: Timeline = {
      ...timeline,
      tracks: newTracks,
    }

    const newHistory = history.slice(0, historyIndex + 1)
    newHistory.push(createHistoryItem('other', beforeState, afterState))

    set({
      timeline: afterState,
      history: newHistory.slice(-MAX_HISTORY),
      historyIndex: newHistory.length - 1,
    })
  },

  removeTrack: (trackId) => {
    const { timeline, history, historyIndex } = get()

    const beforeState = JSON.parse(JSON.stringify(timeline))
    const newTracks = timeline.tracks.filter((t) => t.id !== trackId)

    const afterState: Timeline = {
      ...timeline,
      tracks: newTracks,
    }

    const newHistory = history.slice(0, historyIndex + 1)
    newHistory.push(createHistoryItem('other', beforeState, afterState))

    set({
      timeline: afterState,
      history: newHistory.slice(-MAX_HISTORY),
      historyIndex: newHistory.length - 1,
      selectedTrackId: timeline.tracks.find((t) => t.id === trackId) ? null : get().selectedTrackId,
    })
  },

  toggleTrackVisibility: (trackId) => {
    const { timeline } = get()
    const newTracks = timeline.tracks.map((t) => (t.id === trackId ? { ...t, visible: !t.visible } : t))
    set({ timeline: { ...timeline, tracks: newTracks } })
  },

  toggleTrackLock: (trackId) => {
    const { timeline } = get()
    const newTracks = timeline.tracks.map((t) => (t.id === trackId ? { ...t, locked: !t.locked } : t))
    set({ timeline: { ...timeline, tracks: newTracks } })
  },

  undo: () => {
    const { history, historyIndex } = get()
    if (historyIndex <= 0) return

    const newIndex = historyIndex - 1
    const item = history[newIndex]
    set({
      timeline: JSON.parse(JSON.stringify(item.beforeState)),
      historyIndex: newIndex,
      currentClip: null,
    })
  },

  redo: () => {
    const { history, historyIndex } = get()
    if (historyIndex >= history.length - 1) return

    const newIndex = historyIndex + 1
    const item = history[newIndex]
    set({
      timeline: JSON.parse(JSON.stringify(item.afterState)),
      historyIndex: newIndex,
      currentClip: null,
    })
  },

  canUndo: () => get().historyIndex > 0,
  canRedo: () => get().historyIndex < get().history.length - 1,

  reset: () => {
    set({
      timeline: createEmptyTimeline(),
      currentClip: null,
      playheadTime: 0,
      selectedTrackId: null,
      history: [],
      historyIndex: -1,
    })
  },

  initializeTimeline: (timeline) => {
    set({
      timeline,
      currentClip: null,
      playheadTime: 0,
      selectedTrackId: null,
      history: [createHistoryItem('other', createEmptyTimeline(), timeline)],
      historyIndex: 0,
    })
  },
}))
