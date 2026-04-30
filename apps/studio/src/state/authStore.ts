import { create } from 'zustand'
import type { AuthSession } from '../api/types'

export const AUTH_STORAGE_KEY = 'dramora-auth-session'

type AuthState = {
  session?: AuthSession
  clearSession: () => void
  setSession: (session: AuthSession) => void
}

function readStoredSession(): AuthSession | undefined {
  if (typeof window === 'undefined') {
    return undefined
  }
  const raw = window.localStorage.getItem(AUTH_STORAGE_KEY)
  if (!raw) {
    return undefined
  }
  try {
    return JSON.parse(raw) as AuthSession
  } catch {
    window.localStorage.removeItem(AUTH_STORAGE_KEY)
    return undefined
  }
}

function writeStoredSession(session?: AuthSession) {
  if (typeof window === 'undefined') {
    return
  }
  if (!session) {
    window.localStorage.removeItem(AUTH_STORAGE_KEY)
    return
  }
  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session))
}

export const useAuthStore = create<AuthState>((set) => ({
  session: readStoredSession(),
  clearSession: () => {
    writeStoredSession(undefined)
    set({ session: undefined })
  },
  setSession: (session) => {
    writeStoredSession(session)
    set({ session })
  },
}))
