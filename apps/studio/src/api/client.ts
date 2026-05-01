import type {
  AuthSession,
  CreateEpisodeRequest,
  CreateInvitationRequest,
  CreateProjectRequest,
  CreateStorySourceRequest,
  Episode,
  ApprovalGate,
  ApprovalGateReviewRequest,
  SaveCharacterBibleRequest,
  Asset,
  Export,
  ExportRecovery,
  GenerationJob,
  GenerationJobRecovery,
  Project,
  ProviderConfig,
  LoginRequest,
  OrganizationInvitation,
  RegisterRequest,
  SaveProviderConfigRequest,
  SeedEpisodeProductionResponse,
  Session,
  SaveTimelineRequest,
  SaveShotPromptPackRequest,
  ShotPromptPack,
  StartStoryAnalysisResponse,
  StoryAnalysis,
  StorySource,
  StoryMap,
  StoryMapItem,
  StoryboardWorkspace,
  StoryboardShot,
  TestProviderResult,
  WorkerMetricsSnapshot,
  WorkflowRun,
  UpdateStoryboardShotRequest,
  Timeline,
} from './types'

const API_BASE_URL = import.meta.env.VITE_MANMU_API_BASE_URL ?? ''
const AUTH_STORAGE_KEY = 'dramora-auth-session'
const AUTH_PUBLIC_PATHS = [
  '/api/v1/auth/login',
  '/api/v1/auth/register',
  '/api/v1/auth/refresh',
  '/api/v1/auth/logout',
]

type ErrorEnvelope = {
  error?: {
    code?: string
    message?: string
  }
}

let inflightRefresh: Promise<string | null> | null = null
let onSessionRefreshed: ((session: AuthSession) => void) | null = null
let onSessionCleared: (() => void) | null = null

export function configureAuthBridge(handlers: {
  onRefreshed?: (session: AuthSession) => void
  onCleared?: () => void
}) {
  onSessionRefreshed = handlers.onRefreshed ?? null
  onSessionCleared = handlers.onCleared ?? null
}

function readStoredSession(): AuthSession | null {
  if (typeof window === 'undefined') {
    return null
  }
  const raw = window.localStorage.getItem(AUTH_STORAGE_KEY)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as AuthSession
  } catch {
    return null
  }
}

async function attemptRefresh(): Promise<string | null> {
  if (inflightRefresh) {
    return inflightRefresh
  }
  const stored = readStoredSession()
  if (!stored?.refresh_token) {
    return null
  }
  inflightRefresh = (async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
        body: JSON.stringify({ refresh_token: stored.refresh_token }),
        headers: { 'content-type': 'application/json' },
        method: 'POST',
      })
      if (!response.ok) {
        onSessionCleared?.()
        return null
      }
      const payload = (await response.json()) as { session: AuthSession }
      onSessionRefreshed?.(payload.session)
      return payload.session.token
    } catch {
      onSessionCleared?.()
      return null
    } finally {
      inflightRefresh = null
    }
  })()
  return inflightRefresh
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const isPublicAuthPath = AUTH_PUBLIC_PATHS.includes(path)
  const authHeader = readStoredAuthHeader()
  const buildHeaders = (token: string | null): HeadersInit => ({
    'content-type': 'application/json',
    ...(token ? { authorization: `Bearer ${token}` } : {}),
    ...init?.headers,
  })

  const initialToken = authHeader ? authHeader.replace(/^Bearer\s+/i, '') : null
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: buildHeaders(initialToken),
    ...init,
  })

  if (response.status === 401 && !isPublicAuthPath) {
    const refreshed = await attemptRefresh()
    if (refreshed) {
      const retryResponse = await fetch(`${API_BASE_URL}${path}`, {
        headers: buildHeaders(refreshed),
        ...init,
      })
      if (!retryResponse.ok) {
        const payload = (await retryResponse.json().catch(() => ({}))) as ErrorEnvelope
        throw new Error(payload.error?.message ?? `Request failed with status ${retryResponse.status}`)
      }
      return (await retryResponse.json()) as T
    }
  }

  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as ErrorEnvelope
    throw new Error(payload.error?.message ?? `Request failed with status ${response.status}`)
  }

  return (await response.json()) as T
}

function readStoredAuthHeader(): string | null {
  const session = readStoredSession()
  return session?.token ? `Bearer ${session.token}` : null
}

export async function register(request: RegisterRequest): Promise<AuthSession> {
  const payload = await fetchJSON<{ session: AuthSession }>('/api/v1/auth/register', {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.session
}

export async function login(request: LoginRequest): Promise<AuthSession> {
  const payload = await fetchJSON<{ session: AuthSession }>('/api/v1/auth/login', {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.session
}

export async function getCurrentSession(): Promise<AuthSession> {
  const payload = await fetchJSON<{ session: AuthSession }>('/api/v1/auth/me')
  return payload.session
}

export async function logout(refreshToken?: string): Promise<void> {
  if (!refreshToken) {
    return
  }
  await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
    body: JSON.stringify({ refresh_token: refreshToken }),
    headers: { 'content-type': 'application/json' },
    method: 'POST',
  }).catch(() => undefined)
}

export async function listSessions(): Promise<Session[]> {
  const payload = await fetchJSON<{ sessions: Session[] }>('/api/v1/auth/sessions')
  return payload.sessions ?? []
}

export async function revokeSession(sessionId: string): Promise<void> {
  await fetchJSON<unknown>(`/api/v1/auth/sessions/${encodeURIComponent(sessionId)}:revoke`, {
    method: 'POST',
  })
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

export async function getGenerationJobRecovery(jobId: string): Promise<GenerationJobRecovery> {
  const payload = await fetchJSON<{ generation_job_recovery: GenerationJobRecovery }>(
    `/api/v1/generation-jobs/${jobId}/recovery`,
  )
  return payload.generation_job_recovery
}

export async function getWorkflowRun(workflowRunId: string): Promise<WorkflowRun> {
  const payload = await fetchJSON<{ workflow_run: WorkflowRun }>(`/api/v1/workflow-runs/${workflowRunId}`)
  return payload.workflow_run
}

export async function startStoryAnalysis(episodeId: string): Promise<StartStoryAnalysisResponse> {
  return fetchJSON<StartStoryAnalysisResponse>(`/api/v1/episodes/${episodeId}/story-analysis/start`, {
    method: 'POST',
  })
}

export async function createStorySource(episodeId: string, request: CreateStorySourceRequest): Promise<StorySource> {
  const payload = await fetchJSON<{ story_source: StorySource }>(`/api/v1/episodes/${episodeId}/story-sources`, {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.story_source
}

export async function listStorySources(episodeId: string): Promise<StorySource[]> {
  const payload = await fetchJSON<{ story_sources: StorySource[] }>(`/api/v1/episodes/${episodeId}/story-sources`)
  return payload.story_sources
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

export async function resubmitApprovalGate(gateId: string, request: ApprovalGateReviewRequest): Promise<ApprovalGate> {
  const payload = await fetchJSON<{ approval_gate: ApprovalGate }>(
    `/api/v1/approval-gates/${gateId}:resubmit`,
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

export async function saveCharacterBible(characterId: string, request: SaveCharacterBibleRequest): Promise<StoryMapItem> {
  const payload = await fetchJSON<{ story_map_item: StoryMapItem }>(
    `/api/v1/story-map-characters/${characterId}/character-bible:save`,
    {
      body: JSON.stringify(request),
      method: 'POST',
    },
  )
  return payload.story_map_item
}

export async function listStoryboardShots(episodeId: string): Promise<StoryboardShot[]> {
  const payload = await fetchJSON<{ storyboard_shots: StoryboardShot[] }>(
    `/api/v1/episodes/${episodeId}/storyboard-shots`,
  )
  return payload.storyboard_shots
}

export async function getStoryboardWorkspace(episodeId: string): Promise<StoryboardWorkspace> {
  const payload = await fetchJSON<{ storyboard_workspace: StoryboardWorkspace }>(
    `/api/v1/episodes/${episodeId}/storyboard-workspace`,
  )
  return payload.storyboard_workspace
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

export async function getExportRecovery(exportId: string): Promise<ExportRecovery> {
  const payload = await fetchJSON<{ export_recovery: ExportRecovery }>(
    `/api/v1/exports/${exportId}/recovery`,
  )
  return payload.export_recovery
}

// admin: provider configs

export async function listProviderConfigs(): Promise<ProviderConfig[]> {
  const payload = await fetchJSON<{ providers: ProviderConfig[] }>('/api/v1/admin/providers')
  return payload.providers
}

export async function saveProviderConfig(request: SaveProviderConfigRequest): Promise<ProviderConfig> {
  const payload = await fetchJSON<{ provider: ProviderConfig }>('/api/v1/admin/providers:save', {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.provider
}

export async function testProviderConfig(capability: string): Promise<TestProviderResult> {
  const payload = await fetchJSON<{ test_result: TestProviderResult }>(`/api/v1/admin/providers/${capability}:test`, {
    method: 'POST',
  })
  return payload.test_result
}

export async function fetchWorkerMetrics(): Promise<WorkerMetricsSnapshot> {
  const payload = await fetchJSON<{ worker_metrics: WorkerMetricsSnapshot }>(
    '/api/v1/admin/worker-metrics',
  )
  return payload.worker_metrics
}

// org invitations (owner/admin only)

export async function listOrganizationInvitations(): Promise<OrganizationInvitation[]> {
  const payload = await fetchJSON<{ invitations: OrganizationInvitation[] }>('/api/v1/organizations/invitations')
  return payload.invitations ?? []
}

export async function createOrganizationInvitation(
  request: CreateInvitationRequest,
): Promise<OrganizationInvitation> {
  const payload = await fetchJSON<{ invitation: OrganizationInvitation }>('/api/v1/organizations/invitations', {
    body: JSON.stringify(request),
    method: 'POST',
  })
  return payload.invitation
}

export async function revokeOrganizationInvitation(invitationId: string): Promise<void> {
  await fetchJSON<unknown>(
    `/api/v1/organizations/invitations/${encodeURIComponent(invitationId)}:revoke`,
    { method: 'POST' },
  )
}
