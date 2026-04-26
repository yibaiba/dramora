import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createEpisode,
  createProject,
  listEpisodes,
  listGenerationJobs,
  listProjects,
  saveEpisodeTimeline,
  startStoryAnalysis,
} from './client'
import type { CreateEpisodeRequest, CreateProjectRequest } from './types'

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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['generation-jobs'] }),
  })
}

export function useSaveEpisodeTimeline() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ durationMs, episodeId }: { durationMs: number; episodeId: string }) =>
      saveEpisodeTimeline(episodeId, { duration_ms: durationMs }),
    onSuccess: (timeline) => queryClient.invalidateQueries({ queryKey: ['timeline', timeline.episode_id] }),
  })
}
