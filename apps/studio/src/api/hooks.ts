import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getCurrentSession,
  login,
	createEpisode,
	createProject,
	createStorySource,
  register,
  approveApprovalGate,
  generateShotPromptPack,
  getExport,
  getExportRecovery,
  getAssetRecovery,
  getEpisodeTimeline,
  getShotPromptPack,
  getStoryAnalysis,
  getStoryMap,
  getWorkflowRun,
  getGenerationJobRecovery,
  getShotPromptPackRecovery,
  getStoryboardWorkspace,
  listEpisodeAssets,
  listEpisodes,
  listApprovalGates,
  listGenerationJobs,
  listProjects,
  listProviderConfigs,
  listOrganizationInvitations,
  createOrganizationInvitation,
  revokeOrganizationInvitation,
  resendOrganizationInvitation,
  listInvitationAuditEvents,
  listSessions,
  revokeSession,
  listStorySources,
  listStoryAnalyses,
  listStoryboardShots,
  lockAsset,
  saveCharacterBible,
  saveEpisodeTimeline,
  saveProviderConfig,
  saveShotPromptPack,
  seedApprovalGates,
	seedEpisodeAssets,
	seedEpisodeProduction,
	seedStoryboardShots,
	seedStoryMap,
	requestApprovalChanges,
	resubmitApprovalGate,
	startShotVideoGeneration,
	startEpisodeExport,
	startStoryAnalysis,
	testProviderConfig,
	smokeChatProvider,
	fetchWorkerMetrics,
	fetchLLMTelemetry,
	resetLLMTelemetry,
	fetchProviderAuditEvents,
	updateStoryboardShot,
} from './client'
import type { InvitationAuditFilter, InvitationAuditPage } from './client'
import type {
  LoginRequest,
  RegisterRequest,
  CreateEpisodeRequest,
  CreateInvitationRequest,
  CreateProjectRequest,
  CreateStorySourceRequest,
  Export,
  SaveProviderConfigRequest,
  SaveCharacterBibleRequest,
  SaveShotPromptPackRequest,
  SaveTimelineRequest,
  UpdateStoryboardShotRequest,
} from './types'

export function useCurrentSession(enabled = true) {
  return useQuery({
    enabled,
    queryFn: getCurrentSession,
    queryKey: ['auth-session'],
    retry: false,
  })
}

export function useRegister() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: RegisterRequest) => register(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['auth-session'] }),
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: LoginRequest) => login(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['auth-session'] }),
  })
}

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

export function useGenerationJobRecovery(jobId?: string, options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? Boolean(jobId)
  return useQuery({
    enabled,
    queryFn: () => getGenerationJobRecovery(jobId ?? ''),
    queryKey: ['generation-job-recovery', jobId],
    refetchInterval: 15_000,
  })
}

export function useShotPromptPackRecovery(shotId?: string, options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? Boolean(shotId)
  return useQuery({
    enabled,
    queryFn: () => getShotPromptPackRecovery(shotId ?? ''),
    queryKey: ['prompt-pack-recovery', shotId],
    refetchInterval: 15_000,
  })
}

export function useWorkflowRun(workflowRunId?: string) {
  return useQuery({
    enabled: Boolean(workflowRunId),
    queryFn: () => getWorkflowRun(workflowRunId ?? ''),
    queryKey: ['workflow-run', workflowRunId],
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
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', episodeId] })
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

export function useStorySources(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => listStorySources(episodeId ?? ''),
    queryKey: ['story-sources', episodeId],
  })
}

export function useCreateStorySource(episodeId?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateStorySourceRequest) => createStorySource(episodeId ?? '', request),
    onSuccess: (source) => queryClient.invalidateQueries({ queryKey: ['story-sources', source.episode_id] }),
  })
}

export function useEpisodeApprovalGates(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => listApprovalGates(episodeId ?? ''),
    queryKey: ['approval-gates', episodeId],
  })
}

export function useSeedApprovalGates() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => seedApprovalGates(episodeId),
    onSuccess: (_gates, episodeId) => {
      queryClient.invalidateQueries({ queryKey: ['approval-gates', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', episodeId] })
    },
  })
}

export function useApproveApprovalGate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ gateId }: { episodeId: string; gateId: string }) => approveApprovalGate(gateId, {}),
    onSuccess: (gate) => {
      queryClient.invalidateQueries({ queryKey: ['approval-gates', gate.episode_id] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', gate.episode_id] })
    },
  })
}

export function useRequestApprovalChanges() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ gateId }: { episodeId: string; gateId: string }) =>
      requestApprovalChanges(gateId, { review_note: 'Changes requested from Studio approval board.' }),
    onSuccess: (gate) => {
      queryClient.invalidateQueries({ queryKey: ['approval-gates', gate.episode_id] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', gate.episode_id] })
    },
  })
}

export function useResubmitApprovalGate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ gateId }: { episodeId: string; gateId: string }) =>
      resubmitApprovalGate(gateId, { review_note: 'Resubmitted from Studio approval board.' }),
    onSuccess: (gate) => {
      queryClient.invalidateQueries({ queryKey: ['approval-gates', gate.episode_id] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', gate.episode_id] })
    },
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
    onSuccess: (_storyMap, episodeId) => {
      queryClient.invalidateQueries({ queryKey: ['story-map', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', episodeId] })
    },
  })
}

export function useSaveCharacterBible() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      characterId,
      request,
    }: {
      characterId: string
      episodeId: string
      request: SaveCharacterBibleRequest
    }) => saveCharacterBible(characterId, request),
    onSuccess: (_storyMapItem, variables) => {
      queryClient.invalidateQueries({ queryKey: ['story-map', variables.episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', variables.episodeId] })
    },
  })
}

export function useStoryboardShots(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => listStoryboardShots(episodeId ?? ''),
    queryKey: ['storyboard-shots', episodeId],
  })
}

export function useStoryboardWorkspace(episodeId?: string) {
  return useQuery({
    enabled: Boolean(episodeId),
    queryFn: () => getStoryboardWorkspace(episodeId ?? ''),
    queryKey: ['storyboard-workspace', episodeId],
    refetchInterval: 10_000,
  })
}

export function useSeedStoryboardShots() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => seedStoryboardShots(episodeId),
    onSuccess: (_shots, episodeId) => {
      queryClient.invalidateQueries({ queryKey: ['storyboard-shots', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', episodeId] })
    },
  })
}

export function useUpdateStoryboardShot() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ request, shotId }: { request: UpdateStoryboardShotRequest; shotId: string }) =>
      updateStoryboardShot(shotId, request),
    onSuccess: (shot) => {
      queryClient.invalidateQueries({ queryKey: ['storyboard-shots', shot.episode_id] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', shot.episode_id] })
    },
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
    onSuccess: (pack) => {
      queryClient.invalidateQueries({ queryKey: ['shot-prompt-pack', pack.shot_id] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', pack.episode_id] })
    },
  })
}

export function useSaveShotPromptPack() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ request, shotId }: { request: SaveShotPromptPackRequest; shotId: string }) =>
      saveShotPromptPack(shotId, request),
    onSuccess: (pack) => {
      queryClient.invalidateQueries({ queryKey: ['shot-prompt-pack', pack.shot_id] })
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', pack.episode_id] })
    },
  })
}

export function useStartShotVideoGeneration() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (shotId: string) => startShotVideoGeneration(shotId),
    onSuccess: (job) => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', job.episode_id] })
    },
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
    onSuccess: (_assets, episodeId) => {
      queryClient.invalidateQueries({ queryKey: ['assets', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', episodeId] })
    },
  })
}

export function useSeedEpisodeProduction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => seedEpisodeProduction(episodeId),
    onSuccess: (_result, episodeId) => {
      queryClient.invalidateQueries({ queryKey: ['approval-gates', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['assets', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['story-map', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-shots', episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', episodeId] })
    },
  })
}

export function useLockAsset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ assetId }: { assetId: string; episodeId: string }) => lockAsset(assetId),
    onSuccess: (_asset, variables) => {
      queryClient.invalidateQueries({ queryKey: ['assets', variables.episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', variables.episodeId] })
    },
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
    refetchInterval: (query) => (isExportInProgress(query.state.data?.status) ? 3_000 : false),
  })
}

export function useExportRecovery(exportId?: string, options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? Boolean(exportId)
  return useQuery({
    enabled,
    queryFn: () => getExportRecovery(exportId ?? ''),
    queryKey: ['export-recovery', exportId],
    refetchInterval: 15_000,
  })
}

export function useAssetRecovery(assetId?: string, options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? Boolean(assetId)
  return useQuery({
    enabled,
    queryFn: () => getAssetRecovery(assetId ?? ''),
    queryKey: ['asset-recovery', assetId],
    refetchInterval: 15_000,
  })
}

function isExportInProgress(status?: Export['status']) {
  return status === 'queued' || status === 'rendering'
}

// admin: provider configs

export function useProviderConfigs() {
  return useQuery({
    queryFn: listProviderConfigs,
    queryKey: ['provider-configs'],
  })
}

export function useSaveProviderConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: SaveProviderConfigRequest) => saveProviderConfig(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['provider-configs'] }),
  })
}

export function useTestProviderConfig() {
  return useMutation({
    mutationFn: (capability: string) => testProviderConfig(capability),
  })
}

export function useSmokeChatProvider() {
  return useMutation({
    mutationFn: () => smokeChatProvider(),
  })
}

export function useWorkerMetrics(enabled = true) {
  return useQuery({
    enabled,
    queryFn: fetchWorkerMetrics,
    queryKey: ['admin', 'worker-metrics'],
    refetchInterval: enabled ? 15000 : false,
  })
}

export function useLLMTelemetry(enabled = true) {
  return useQuery({
    enabled,
    queryFn: fetchLLMTelemetry,
    queryKey: ['admin', 'llm-telemetry'],
    refetchInterval: enabled ? 10000 : false,
  })
}

export function useResetLLMTelemetry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: resetLLMTelemetry,
    onSuccess: (snapshot) => {
      queryClient.setQueryData(['admin', 'llm-telemetry'], snapshot)
    },
  })
}

export type ProviderAuditFilter = {
  action?: string
  capability?: string
  actor?: string
  sinceMinutes?: number
  limit?: number
}

export function useProviderAuditEvents(filter?: ProviderAuditFilter, enabled = true) {
  const params: ProviderAuditFilter = { limit: 50, ...(filter ?? {}) }
  return useQuery({
    enabled,
    queryFn: () => {
      const since =
        params.sinceMinutes != null
          ? new Date(Date.now() - params.sinceMinutes * 60 * 1000).toISOString()
          : undefined
      return fetchProviderAuditEvents({
        action: params.action,
        capability: params.capability,
        actor: params.actor,
        since,
        limit: params.limit,
      })
    },
    queryKey: ['admin', 'provider-audit', params],
    refetchInterval: enabled ? 15000 : false,
  })
}

export function useOrganizationInvitations(enabled = true) {
  return useQuery({
    enabled,
    queryFn: listOrganizationInvitations,
    queryKey: ['organization-invitations'],
  })
}

export function useCreateInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateInvitationRequest) => createOrganizationInvitation(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-invitations'] })
      queryClient.invalidateQueries({ queryKey: ['invitation-audit'] })
    },
  })
}

export function useRevokeInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (invitationId: string) => revokeOrganizationInvitation(invitationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-invitations'] })
      queryClient.invalidateQueries({ queryKey: ['invitation-audit'] })
    },
  })
}

export function useResendInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (invitationId: string) => resendOrganizationInvitation(invitationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-invitations'] })
      queryClient.invalidateQueries({ queryKey: ['invitation-audit'] })
    },
  })
}

export function useInvitationAuditEvents(enabled = true, filter: InvitationAuditFilter = {}) {
  return useQuery<InvitationAuditPage>({
    enabled,
    queryFn: () => listInvitationAuditEvents(filter),
    queryKey: [
      'invitation-audit',
      filter.limit ?? 50,
      filter.offset ?? 0,
      (filter.actions ?? []).slice().sort().join(','),
      filter.email ?? '',
      filter.since ?? '',
      filter.until ?? '',
    ],
  })
}

export function useSessions(enabled = true) {
  return useQuery({
    enabled,
    queryFn: listSessions,
    queryKey: ['auth-sessions'],
  })
}

export function useRevokeSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (sessionId: string) => revokeSession(sessionId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['auth-sessions'] }),
  })
}
