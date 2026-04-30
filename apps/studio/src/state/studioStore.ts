import { create } from 'zustand'

type StudioState = {
  eventLog: string[]
  selectedEpisodeId?: string
  selectedProjectId?: string
  clearEpisode: () => void
  logEvent: (message: string) => void
  selectEpisode: (episodeId: string) => void
  selectProject: (projectId: string) => void
}

export const useStudioStore = create<StudioState>((set) => ({
  eventLog: [],
  selectedEpisodeId: undefined,
  selectedProjectId: undefined,
  clearEpisode: () => set({ selectedEpisodeId: undefined }),
  logEvent: (message) =>
    set((state) => ({
      eventLog: [message, ...state.eventLog].slice(0, 6),
    })),
  selectEpisode: (episodeId) => set({ selectedEpisodeId: episodeId }),
  selectProject: (projectId) => set({ selectedProjectId: projectId }),
}))
