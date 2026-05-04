/**
 * React Hook for WebSocket event handling
 */

import { useEffect, useState, useCallback, useRef } from 'react'
import {
  getWebSocketClient,
  type WebSocketClient,
} from './websocket'
import type { WebSocketMessage } from '../types/websocket'

interface WebSocketEventHandler {
  (message: WebSocketMessage): void
}

interface UseWebSocketEventsReturn {
  isConnected: boolean
  connectionError: string | null
  connect: () => Promise<void>
  disconnect: () => void
  on: (eventType: string, handler: WebSocketEventHandler) => () => void
}

/**
 * Hook for managing WebSocket connections and event handlers
 */
export function useWebSocketEvents(): UseWebSocketEventsReturn {
  const clientRef = useRef<WebSocketClient | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [connectionError, setConnectionError] = useState<string | null>(null)

  // Initialize client
  useEffect(() => {
    clientRef.current = getWebSocketClient()
    return () => {
      // Don't disconnect on unmount - client is a singleton
      // This allows multiple components to share the same connection
    }
  }, [])

  // Connect on mount
  useEffect(() => {
    const client = clientRef.current
    if (!client) return

    if (!client.isConnected()) {
      client
        .connect()
        .then(() => {
          setIsConnected(true)
          setConnectionError(null)
        })
        .catch((err) => {
          console.error('[useWebSocketEvents] Connect failed:', err)
          setIsConnected(false)
          setConnectionError(err instanceof Error ? err.message : 'Connection failed')
        })
    } else {
      setIsConnected(true)
      setConnectionError(null)
    }

    // Update connection status periodically
    const checkInterval = setInterval(() => {
      setIsConnected(client.isConnected())
    }, 1000)

    return () => {
      clearInterval(checkInterval)
    }
  }, [])

  const connect = useCallback(async () => {
    const client = clientRef.current
    if (!client) {
      throw new Error('WebSocket client not initialized')
    }

    try {
      await client.connect()
      setIsConnected(true)
      setConnectionError(null)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Connection failed'
      console.error('[useWebSocketEvents] Connect failed:', err)
      setIsConnected(false)
      setConnectionError(errorMessage)
      throw err
    }
  }, [])

  const disconnect = useCallback(() => {
    const client = clientRef.current
    if (client) {
      client.disconnect()
      setIsConnected(false)
      setConnectionError(null)
    }
  }, [])

  const on = useCallback(
    (eventType: string, handler: WebSocketEventHandler): (() => void) => {
      const client = clientRef.current
      if (!client) {
        console.warn('[useWebSocketEvents] Client not initialized')
        return () => {}
      }

      return client.on(eventType, handler)
    },
    []
  )

  return {
    isConnected,
    connectionError,
    connect,
    disconnect,
    on,
  }
}

/**
 * Hook for listening to a specific WebSocket event type
 */
export function useWebSocketEvent(
  eventType: string,
  onMessage: (message: WebSocketMessage) => void
): void {
  const { on } = useWebSocketEvents()

  useEffect(() => {
    const unsubscribe = on(eventType, onMessage)
    return unsubscribe
  }, [eventType, onMessage, on])
}
