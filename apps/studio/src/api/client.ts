import type {
  CreateEpisodeRequest,
  CreateProjectRequest,
  Episode,
  ApprovalGate,
  ApprovalGateReviewRequest,
  Asset,
  Export,
  GenerationJob,
  Project,
  SeedEpisodeProductionResponse,
  SaveTimelineRequest,
  SaveShotPromptPackRequest,
  ShotPromptPack,
  StartStoryAnalysisResponse,
  StoryAnalysis,
  StoryMap,
  StoryboardShot,
  UpdateStoryboardShotRequest,
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

export async function listStoryAnalyses(episodeId: string): Promise<StoryAnalysis[]> {
  const payload = await fetchJSON<{ story_analyses: StoryAnalysis[] }>(`/api/v1/episodes/${episodeId}/story-analyses`)
  return payload.story_analyses
}

export async function listApprovalGates(episodeId: string): Promise<ApprovalGate[]> {
  const payload = await fetchJSON<{ approval_gates: ApprovalGate[] }>(`/api/v1/episodes/${episodeId}/approval-gates`)
  return payload.approval_gates
}

export async function seedApprovalGates(episodeId: string): Promise<ApprovalGate[]> {
  const payload = await fetchJSON<{ approval_gates: ApprovalGate[] }>(
    `/api/v1/episodes/${episodeId}/approval-gates:seed`,
    { method: 'POST' },
  )
  return payload.approval_gates
}

export async function approveApprovalGate(gateId: string, request: ApprovalGateReviewRequest): Promise<ApprovalGate> {
  const payload = await fetchJSON<{ approval_gate: ApprovalGate }>(`/api/v1/approval-gates/${gateId}:approve`, {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.approval_gate
}

export async function requestApprovalChanges(gateId: string, request: ApprovalGateReviewRequest): Promise<ApprovalGate> {
  const payload = await fetchJSON<{ approval_gate: ApprovalGate }>(
    `/api/v1/approval-gates/${gateId}:request-changes`,
    {
      body: JSON.stringify(request),
      method: 'POST',
    },
  )
  return payload.approval_gate
}

export async function getStoryAnalysis(analysisId: string): Promise<StoryAnalysis> {
  const payload = await fetchJSON<{ story_analysis: StoryAnalysis }>(`/api/v1/story-analyses/${analysisId}`)
  return payload.story_analysis
}

export async function getStoryMap(episodeId: string): Promise<StoryMap> {
  const payload = await fetchJSON<{ story_map: StoryMap }>(`/api/v1/episodes/${episodeId}/story-map`)
  return payload.story_map
}

export async function seedStoryMap(episodeId: string): Promise<StoryMap> {
  const payload = await fetchJSON<{ story_map: StoryMap }>(`/api/v1/episodes/${episodeId}/story-map:seed`, {
    method: 'POST',
  })
  return payload.story_map
}

export async function listStoryboardShots(episodeId: string): Promise<StoryboardShot[]> {
  const payload = await fetchJSON<{ storyboard_shots: StoryboardShot[] }>(
    `/api/v1/episodes/${episodeId}/storyboard-shots`,
  )
  return payload.storyboard_shots
}

export async function seedStoryboardShots(episodeId: string): Promise<StoryboardShot[]> {
  const payload = await fetchJSON<{ storyboard_shots: StoryboardShot[] }>(
    `/api/v1/episodes/${episodeId}/storyboard-shots:seed`,
    { method: 'POST' },
  )
  return payload.storyboard_shots
}

export async function updateStoryboardShot(shotId: string, request: UpdateStoryboardShotRequest): Promise<StoryboardShot> {
  const payload = await fetchJSON<{ storyboard_shot: StoryboardShot }>(`/api/v1/storyboard-shots/${shotId}:update`, {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.storyboard_shot
}

export async function getShotPromptPack(shotId: string): Promise<ShotPromptPack> {
  const payload = await fetchJSON<{ prompt_pack: ShotPromptPack }>(`/api/v1/storyboard-shots/${shotId}/prompt-pack`)
  return payload.prompt_pack
}

export async function generateShotPromptPack(shotId: string): Promise<ShotPromptPack> {
  const payload = await fetchJSON<{ prompt_pack: ShotPromptPack }>(
    `/api/v1/storyboard-shots/${shotId}/prompt-pack:generate`,
    { method: 'POST' },
  )
  return payload.prompt_pack
}

export async function saveShotPromptPack(shotId: string, request: SaveShotPromptPackRequest): Promise<ShotPromptPack> {
  const payload = await fetchJSON<{ prompt_pack: ShotPromptPack }>(
    `/api/v1/storyboard-shots/${shotId}/prompt-pack:save`,
    {
      body: JSON.stringify(request),
      method: 'POST',
    },
  )
  return payload.prompt_pack
}

export async function startShotVideoGeneration(shotId: string): Promise<GenerationJob> {
  const payload = await fetchJSON<{ generation_job: GenerationJob }>(`/api/v1/storyboard-shots/${shotId}/videos:generate`, {
    method: 'POST',
  })
  return payload.generation_job
}

export async function listEpisodeAssets(episodeId: string): Promise<Asset[]> {
  const payload = await fetchJSON<{ assets: Asset[] }>(`/api/v1/episodes/${episodeId}/assets`)
  return payload.assets
}

export async function seedEpisodeAssets(episodeId: string): Promise<Asset[]> {
  const payload = await fetchJSON<{ assets: Asset[] }>(`/api/v1/episodes/${episodeId}/assets:seed`, {
    method: 'POST',
  })
  return payload.assets
}

export async function seedEpisodeProduction(episodeId: string): Promise<SeedEpisodeProductionResponse> {
  return fetchJSON<SeedEpisodeProductionResponse>(`/api/v1/episodes/${episodeId}/production:seed`, {
    method: 'POST',
  })
}

export async function lockAsset(assetId: string): Promise<Asset> {
  const payload = await fetchJSON<{ asset: Asset }>(`/api/v1/assets/${assetId}:lock`, {
    method: 'POST',
  })
  return payload.asset
}

export async function saveEpisodeTimeline(episodeId: string, request: SaveTimelineRequest): Promise<Timeline> {
  const payload = await fetchJSON<{ timeline: Timeline }>(`/api/v1/episodes/${episodeId}/timeline`, {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.timeline
}

export async function getEpisodeTimeline(episodeId: string): Promise<Timeline> {
  const payload = await fetchJSON<{ timeline: Timeline }>(`/api/v1/episodes/${episodeId}/timeline`)
  return payload.timeline
}

export async function startEpisodeExport(episodeId: string): Promise<Export> {
  const payload = await fetchJSON<{ export: Export }>(`/api/v1/episodes/${episodeId}/exports`, {
    method: 'POST',
  })
  return payload.export
}

export async function getExport(exportId: string): Promise<Export> {
  const payload = await fetchJSON<{ export: Export }>(`/api/v1/exports/${exportId}`)
  return payload.export
}
