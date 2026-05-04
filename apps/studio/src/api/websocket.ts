/**
 * WebSocket client with auto-reconnect and event handling
 */

import type { WebSocketConfig, WebSocketMessage } from '../types/websocket'

type EventHandler = (message: WebSocketMessage) => void

export class WebSocketClient {
  private ws: WebSocket | null = null
  private url: string
  private reconnectInterval: number
  private maxReconnectInterval: number
  private maxReconnectAttempts: number
  private reconnectAttempts = 0
  private handlers: Map<string, Set<EventHandler>> = new Map()
  private isIntentionallyClosed = false
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private connectPromise: Promise<void> | null = null

  constructor(config: WebSocketConfig = {}) {
    const defaultUrl =
      import.meta.env.DEV
        ? `ws://localhost:${import.meta.env.VITE_API_PORT || 8080}/ws`
        : `wss://${window.location.host}/ws`

    this.url = config.url || defaultUrl
    this.reconnectInterval = config.reconnectInterval || 1000
    this.maxReconnectInterval = config.maxReconnectInterval || 30000
    this.maxReconnectAttempts = config.maxReconnectAttempts || 10
  }

  /**
   * Connect to WebSocket server
   */
  connect(): Promise<void> {
    if (this.connectPromise) {
      return this.connectPromise
    }

    this.connectPromise = new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url)

        this.ws.onopen = () => {
          console.log('[WebSocket] Connected')
          this.reconnectAttempts = 0
          this.isIntentionallyClosed = false
          this.startHeartbeat()
          this.connectPromise = null
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data)
            this.dispatchMessage(message)
          } catch (err) {
            console.error('[WebSocket] Failed to parse message:', err)
          }
        }

        this.ws.onerror = (error) => {
          console.error('[WebSocket] Error:', error)
          this.connectPromise = null
          reject(error)
        }

        this.ws.onclose = () => {
          console.log('[WebSocket] Disconnected')
          this.stopHeartbeat()
          this.connectPromise = null

          if (!this.isIntentionallyClosed) {
            this.attemptReconnect()
          }
        }
      } catch (err) {
        this.connectPromise = null
        reject(err)
      }
    })

    return this.connectPromise
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.isIntentionallyClosed = true
    this.stopHeartbeat()

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  /**
   * Register event handler
   */
  on(eventType: string, handler: EventHandler): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set())
    }
    this.handlers.get(eventType)!.add(handler)

    // Return unsubscribe function
    return () => {
      this.off(eventType, handler)
    }
  }

  /**
   * Unregister event handler
   */
  off(eventType: string, handler: EventHandler): void {
    const handlers = this.handlers.get(eventType)
    if (handlers) {
      handlers.delete(handler)
      if (handlers.size === 0) {
        this.handlers.delete(eventType)
      }
    }
  }

  /**
   * Dispatch message to registered handlers
   */
  private dispatchMessage(message: WebSocketMessage): void {
    const handlers = this.handlers.get(message.type)
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler(message)
        } catch (err) {
          console.error(
            `[WebSocket] Error in handler for ${message.type}:`,
            err
          )
        }
      })
    }
  }

  /**
   * Attempt to reconnect with exponential backoff
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error(
        '[WebSocket] Max reconnect attempts reached, giving up'
      )
      return
    }

    this.reconnectAttempts++

    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
    const delay = Math.min(
      this.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1),
      this.maxReconnectInterval
    )

    console.log(
      `[WebSocket] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    )

    setTimeout(() => {
      if (!this.isIntentionallyClosed) {
        this.connect().catch((err) => {
          console.error('[WebSocket] Reconnect failed:', err)
        })
      }
    }, delay)
  }

  /**
   * Start heartbeat ping
   */
  private startHeartbeat(): void {
    this.stopHeartbeat()

    // Send ping every 30 seconds
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        try {
          this.ws?.send(JSON.stringify({ type: 'ping' }))
        } catch (err) {
          console.error('[WebSocket] Failed to send ping:', err)
        }
      }
    }, 30000)
  }

  /**
   * Stop heartbeat ping
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }
}

// Singleton instance
let wsClientInstance: WebSocketClient | null = null

/**
 * Get or create the WebSocket client singleton
 */
export function getWebSocketClient(
  config?: WebSocketConfig
): WebSocketClient {
  if (!wsClientInstance) {
    wsClientInstance = new WebSocketClient(config)
  }
  return wsClientInstance
}
