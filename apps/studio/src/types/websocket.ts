/**
 * WebSocket event types and message formats
 */

export const WebSocketEventType = {
  GENERATION_PROGRESS: 'generation_progress',
  GENERATION_COMPLETED: 'generation_completed',
  GENERATION_FAILED: 'generation_failed',
} as const

export type WebSocketEventType = (typeof WebSocketEventType)[keyof typeof WebSocketEventType]

export interface WebSocketMessage {
  type: string
  data: Record<string, unknown>
}

export interface GenerationProgressEvent {
  videoID: string
  progress?: number
  status?: string
}

export interface GenerationCompletedEvent {
  videoID: string
  videoUrl?: string
}

export interface GenerationFailedEvent {
  videoID: string
  error: string
}

export type WebSocketEventData =
  | GenerationProgressEvent
  | GenerationCompletedEvent
  | GenerationFailedEvent
  | Record<string, unknown>

export interface WebSocketConfig {
  url?: string
  reconnectInterval?: number
  maxReconnectInterval?: number
  maxReconnectAttempts?: number
}
