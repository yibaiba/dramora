import type {
  CreateEpisodeRequest,
  CreateProjectRequest,
  Episode,
  GenerationJob,
  Project,
  SaveTimelineRequest,
  StartStoryAnalysisResponse,
  Timeline,
} from './types'

const API_BASE_URL = import.meta.env.VITE_MANMU_API_BASE_URL ?? ''

type ErrorEnvelope = {
  error?: {
    code?: string
    message?: string
  }
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'content-type': 'application/json',
      ...init?.headers,
    },
    ...init,
  })

  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as ErrorEnvelope
    throw new Error(payload.error?.message ?? `Request failed with status ${response.status}`)
  }

  return (await response.json()) as T
}

export async function listProjects(): Promise<Project[]> {
  const payload = await fetchJSON<{ projects: Project[] }>('/api/v1/projects')
  return payload.projects
}

export async function createProject(request: CreateProjectRequest): Promise<Project> {
  const payload = await fetchJSON<{ project: Project }>('/api/v1/projects', {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.project
}

export async function listEpisodes(projectId: string): Promise<Episode[]> {
  const payload = await fetchJSON<{ episodes: Episode[] }>(`/api/v1/projects/${projectId}/episodes`)
  return payload.episodes
}

export async function createEpisode(projectId: string, request: CreateEpisodeRequest): Promise<Episode> {
  const payload = await fetchJSON<{ episode: Episode }>(`/api/v1/projects/${projectId}/episodes`, {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.episode
}

export async function listGenerationJobs(): Promise<GenerationJob[]> {
  const payload = await fetchJSON<{ generation_jobs: GenerationJob[] }>('/api/v1/generation-jobs')
  return payload.generation_jobs
}

export async function startStoryAnalysis(episodeId: string): Promise<StartStoryAnalysisResponse> {
  return fetchJSON<StartStoryAnalysisResponse>(`/api/v1/episodes/${episodeId}/story-analysis/start`, {
    method: 'POST',
  })
}

export async function saveEpisodeTimeline(episodeId: string, request: SaveTimelineRequest): Promise<Timeline> {
  const payload = await fetchJSON<{ timeline: Timeline }>(`/api/v1/episodes/${episodeId}/timeline`, {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.timeline
}
