// Lightweight SSE consumer for /api/v1/agents/stream.
// Uses fetch + ReadableStream so we can carry the bearer token (EventSource cannot).
import type { AuthSession } from './types'

const API_BASE_URL = import.meta.env.VITE_MANMU_API_BASE_URL ?? ''
const AUTH_STORAGE_KEY = 'dramora-auth-session'

export type AgentStreamDoneFrame = {
  role: string
  output: string
  highlights: string[]
  token_count: number
  duration_ms: number
}

export type AgentStreamCallbacks = {
  onDelta?: (text: string) => void
  onDone?: (frame: AgentStreamDoneFrame) => void
  onError?: (message: string) => void
}

function readToken(): string | null {
  if (typeof window === 'undefined') return null
  const raw = window.localStorage.getItem(AUTH_STORAGE_KEY)
  if (!raw) return null
  try {
    const session = JSON.parse(raw) as AuthSession
    return session?.token ?? null
  } catch {
    return null
  }
}

export async function streamAgentRun(
  body: { role: string; source_text: string; context?: Record<string, string> },
  callbacks: AgentStreamCallbacks,
  signal?: AbortSignal,
): Promise<void> {
  const token = readToken()
  const response = await fetch(`${API_BASE_URL}/api/v1/agents/stream`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      accept: 'text/event-stream',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
    signal,
  })

  if (!response.ok || !response.body) {
    let message = `Request failed with status ${response.status}`
    try {
      const payload = (await response.json()) as { error?: { message?: string } }
      if (payload?.error?.message) message = payload.error.message
    } catch {
      // ignore parse failure, use default
    }
    callbacks.onError?.(message)
    return
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''
  let currentEvent = ''

  const dispatch = (event: string, data: string) => {
    if (!data) return
    if (event === 'delta') {
      try {
        const parsed = JSON.parse(data) as { content?: string }
        if (parsed.content) callbacks.onDelta?.(parsed.content)
      } catch {
        // ignore malformed frame
      }
    } else if (event === 'done') {
      try {
        const parsed = JSON.parse(data) as AgentStreamDoneFrame
        callbacks.onDone?.(parsed)
      } catch {
        callbacks.onError?.('failed to parse done frame')
      }
    } else if (event === 'error') {
      try {
        const parsed = JSON.parse(data) as { message?: string }
        callbacks.onError?.(parsed.message ?? 'stream error')
      } catch {
        callbacks.onError?.('stream error')
      }
    }
  }

  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let sepIndex = buffer.indexOf('\n\n')
    while (sepIndex !== -1) {
      const rawFrame = buffer.slice(0, sepIndex)
      buffer = buffer.slice(sepIndex + 2)
      const lines = rawFrame.split('\n')
      let event = ''
      let data = ''
      for (const line of lines) {
        if (line.startsWith('event: ')) event = line.slice(7).trim()
        else if (line.startsWith('data: ')) data += (data ? '\n' : '') + line.slice(6)
      }
      if (event) currentEvent = event
      dispatch(currentEvent || 'message', data)
      sepIndex = buffer.indexOf('\n\n')
    }
  }
}
