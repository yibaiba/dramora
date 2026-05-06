import { useCallback, useEffect, useState } from 'react'

export type ThemeMode = 'dark' | 'light'

const STORAGE_KEY = 'dramora.studio.themeMode.v2'
const LEGACY_STORAGE_KEY = 'dramora.studio.themeMode'

function readInitialTheme(): ThemeMode {
  if (typeof window === 'undefined') return 'dark'
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    if (stored === 'dark' || stored === 'light') return stored
    if (window.localStorage.getItem(LEGACY_STORAGE_KEY) === 'dark') return 'dark'
  } catch {
    // ignore storage failures
  }
  return 'dark'
}

function applyTheme(mode: ThemeMode) {
  if (typeof document === 'undefined') return
  document.documentElement.setAttribute('data-theme', mode)
}

export function useThemeMode() {
  const [mode, setMode] = useState<ThemeMode>(() => {
    const initial = readInitialTheme()
    applyTheme(initial)
    return initial
  })

  useEffect(() => {
    applyTheme(mode)
    try {
      window.localStorage.setItem(STORAGE_KEY, mode)
      window.localStorage.removeItem(LEGACY_STORAGE_KEY)
    } catch {
      // ignore storage failures
    }
  }, [mode])

  const toggle = useCallback(() => {
    setMode((prev) => (prev === 'dark' ? 'light' : 'dark'))
  }, [])

  return { mode, setMode, toggle }
}
