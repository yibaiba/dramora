/**
 * Video editor data types and structures
 */

export type TrackType = 'video' | 'audio' | 'subtitle'

export type Clip = {
  id: string
  trackId: string
  startTime: number // milliseconds
  duration: number // milliseconds
  sourceUrl: string
  properties: {
    speed: number // 1.0 = normal speed
    opacity: number // 0-1
  }
}

export type Track = {
  id: string
  type: TrackType
  name: string
  clips: Clip[]
  visible: boolean
  locked: boolean
  height: number // pixel height for rendering
}

export type Timeline = {
  tracks: Track[]
  duration: number // total length in milliseconds
  fps: number // 30 or 24
}

export type EditHistoryItem = {
  action: 'add_clip' | 'remove_clip' | 'move_clip' | 'trim_clip' | 'other'
  beforeState: Timeline
  afterState: Timeline
  timestamp: number
}

export type EditorMode = 'modal' | 'fullscreen'

export type EditorState = {
  isOpen: boolean
  mode: EditorMode
  videoId: string | null
  videoUrl: string | null
  videoTitle: string | null
}
