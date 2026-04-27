import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createEpisode,
  createProject,
  getExport,
  getEpisodeTimeline,
  getStoryAnalysis,
  getStoryMap,
  listEpisodes,
  listGenerationJobs,
  listProjects,
  listStoryAnalyses,
  listStoryboardShots,
  saveEpisodeTimeline,
  seedStoryboardShots,
  seedStoryMap,
  startEpisodeExport,
  startStoryAnalysis,
} from './client'
import type { CreateEpisodeRequest, CreateProjectRequest, SaveTimelineRequest } from './types'

export function useProjects() {
  return useQuery({
    queryFn: listProjects,
    queryKey: ['projects'],
  })
}

export function useCreateProject() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateProjectRequest) => createProject(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['projects'] }),
  })
}

export function useEpisodes(projectId?: string) {
  return useQuery({
    enabled: Boolean(projectId),
    queryFn: () => listEpisodes(projectId ?? ''),
    queryKey: ['episodes', projectId],
  })
}

export function useCreateEpisode(projectId?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateEpisodeRequest) => {
      if (!projectId) {
        throw new Error('Select a project before creating an episode')
      }
      return createEpisode(projectId, request)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['episodes', projectId] }),
  })
}

export function useGenerationJobs() {
  return useQuery({
    queryFn: listGenerationJobs,
    queryKey: ['generation-jobs'],
    refetchInterval: 10_000,
  })
}

export function useStartStoryAnalysis() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => startStoryAnalysis(episodeId),
    onSuccess: (_result, episodeId) => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
      queryClient.invalidateQueries({ queryKey: ['story-analyses', episodeId] })
    },
  })
}

export function useStoryAnalyses(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => listStoryAnalyses(episodeId ?? ''),
    queryKey: ['story-analyses', episodeId],
    refetchInterval: 10_000,
  })
}

export function useStoryAnalysis(analysisId?: string) {
  return useQuery({
    enabled: Boolean(analysisId),
    queryFn: () => getStoryAnalysis(analysisId ?? ''),
    queryKey: ['story-analysis', analysisId],
  })
}

export function useStoryMap(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => getStoryMap(episodeId ?? ''),
    queryKey: ['story-map', episodeId],
  })
}

export function useSeedStoryMap() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => seedStoryMap(episodeId),
    onSuccess: (_storyMap, episodeId) => queryClient.invalidateQueries({ queryKey: ['story-map', episodeId] }),
  })
}

export function useStoryboardShots(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => listStoryboardShots(episodeId ?? ''),
    queryKey: ['storyboard-shots', episodeId],
  })
}

export function useSeedStoryboardShots() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => seedStoryboardShots(episodeId),
    onSuccess: (_shots, episodeId) => queryClient.invalidateQueries({ queryKey: ['storyboard-shots', episodeId] }),
  })
}

export function useSaveEpisodeTimeline() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ episodeId, request }: { episodeId: string; request: SaveTimelineRequest }) =>
      saveEpisodeTimeline(episodeId, request),
    onSuccess: (timeline) => queryClient.invalidateQueries({ queryKey: ['timeline', timeline.episode_id] }),
  })
}

export function useEpisodeTimeline(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => getEpisodeTimeline(episodeId ?? ''),
    queryKey: ['timeline', episodeId],
  })
}

export function useStartEpisodeExport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => startEpisodeExport(episodeId),
    onSuccess: (item) => queryClient.invalidateQueries({ queryKey: ['export', item.id] }),
  })
}

export function useExport(exportId?: string) {
  return useQuery({
    enabled: Boolean(exportId),
    queryFn: () => getExport(exportId ?? ''),
    queryKey: ['export', exportId],
  })
}
