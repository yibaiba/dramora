import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createEpisode,
  createProject,
  generateShotPromptPack,
  getExport,
  getEpisodeTimeline,
  getShotPromptPack,
  getStoryAnalysis,
  getStoryMap,
  listEpisodeAssets,
  listEpisodes,
  listGenerationJobs,
  listProjects,
  listStoryAnalyses,
  listStoryboardShots,
  lockAsset,
  saveEpisodeTimeline,
  seedEpisodeAssets,
  seedStoryboardShots,
  seedStoryMap,
  startShotVideoGeneration,
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

export function useShotPromptPack(shotId?: string) {
  return useQuery({
    enabled: Boolean(shotId),
    queryFn: () => getShotPromptPack(shotId ?? ''),
    queryKey: ['shot-prompt-pack', shotId],
    retry: false,
  })
}

export function useGenerateShotPromptPack() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (shotId: string) => generateShotPromptPack(shotId),
    onSuccess: (pack) => queryClient.invalidateQueries({ queryKey: ['shot-prompt-pack', pack.shot_id] }),
  })
}

export function useStartShotVideoGeneration() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (shotId: string) => startShotVideoGeneration(shotId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['generation-jobs'] }),
  })
}

export function useEpisodeAssets(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => listEpisodeAssets(episodeId ?? ''),
    queryKey: ['assets', episodeId],
  })
}

export function useSeedEpisodeAssets() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => seedEpisodeAssets(episodeId),
    onSuccess: (_assets, episodeId) => queryClient.invalidateQueries({ queryKey: ['assets', episodeId] }),
  })
}

export function useLockAsset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ assetId }: { assetId: string; episodeId: string }) => lockAsset(assetId),
    onSuccess: (_asset, variables) => queryClient.invalidateQueries({ queryKey: ['assets', variables.episodeId] }),
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
