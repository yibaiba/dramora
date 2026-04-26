import { create } from 'zustand'

type StudioState = {
  eventLog: string[]
  selectedProjectId?: string
  logEvent: (message: string) => void
  selectProject: (projectId: string) => void
}

export const useStudioStore = create<StudioState>((set) => ({
  eventLog: [],
  selectedProjectId: undefined,
  logEvent: (message) =>
    set((state) => ({
      eventLog: [message, ...state.eventLog].slice(0, 6),
    })),
  selectProject: (projectId) => set({ selectedProjectId: projectId }),
}))
