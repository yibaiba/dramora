import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getCurrentSession,
  login,
  changePassword,
	createEpisode,
  createUserAPIKey,
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
  listWorkspaces,
  listProviderConfigs,
  listOrganizationMembers,
  listOrganizationInvitations,
  createOrganizationInvitation,
  updateOrganizationMemberRole,
  removeOrganizationMember,
  revokeOrganizationInvitation,
  resendOrganizationInvitation,
  listInvitationAuditEvents,
  listSessions,
  listUserAPIKeys,
  revokeSession,
  listStorySources,
  listStoryAnalyses,
  listStoryboardShots,
  lockAsset,
  deleteAsset,
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
	smokeChatProviderStream,
	fetchWorkerMetrics,
	fetchLLMTelemetry,
	resetLLMTelemetry,
	fetchProviderAuditEvents,
	updateStoryboardShot,
	fetchWalletSnapshot,
	fetchWalletTransactions,
	creditWallet,
	debitWallet,
	getOperationCosts,
	fetchNotifications,
	markNotificationAsRead,
	markAllNotificationsAsRead,
	sendChatMessage,
	chargeWallet,
	initiateChargeWallet,
	getAdminOperationCosts,
	updateAdminOperationCosts,
	getAdminOperationCostHistory,
	getAdminBillingReports,
	generateAdminBillingReport,
	getAdminBillingReportByID,
	getAdminBillingReportSummary,
	batchGenerateShots,
	retryGenerationJob,
	updateGenerationJobPriority,
	pauseEpisodeQueue,
	resumeEpisodeQueue,
	redeemCode,
	generateRedemptionCodes,
	getCampaignStats,
	listShortVideoTemplates,
	getShortVideoTemplate,
	createShortVideoTemplate,
	deleteShortVideoTemplate,
	createShortVideo,
	listShortVideos,
	getShortVideo,
  deleteShortVideo,
  deleteUserAPIKey,
  createBatchSubmission,
  listBatchSubmissions,
  getBatchSubmission,
  cancelBatchSubmission,
  retryBatchVideo,
  toggleUserAPIKey,
  updateUserAPIKey,
} from './client'
import type { InvitationAuditFilter, InvitationAuditPage } from './client'
import { useAuthStore } from '../state/authStore'
import type {
  ChangePasswordRequest,
  LoginRequest,
  RegisterRequest,
  CreateEpisodeRequest,
  CreateUserAPIKeyRequest,
  CreateInvitationRequest,
  CreateProjectRequest,
  CreateStorySourceRequest,
  OrganizationRole,
  UpdateOrganizationMemberRoleRequest,
  UpdateOperationCostsRequest,
  Export,
  SaveProviderConfigRequest,
  SaveCharacterBibleRequest,
  SaveShotPromptPackRequest,
  SaveTimelineRequest,
  UpdateStoryboardShotRequest,
  WalletKind,
  WalletMutationRequest,
  ChatMessageRequest,
  ChargeWalletRequest,
  ChargeInitiateRequest,
  GenerateBillingReportRequest,
  BatchGenerateShotsRequest,
  RedeemCodeRequest,
  GenerateRedemptionCodesRequest,
  CreateShortVideoTemplateRequest,
  CreateShortVideoRequest,
  CreateBatchSubmissionRequest,
  ToggleUserAPIKeyRequest,
  UpdateUserAPIKeyRequest,
} from './types'

function shouldRetryAuthedQuery(failureCount: number, error: unknown): boolean {
  if (error instanceof Error && error.message.toLowerCase().includes('authentication required')) {
    return false
  }
  return failureCount < 2
}

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

export function useWorkspaces() {
  return useQuery({
    queryFn: listWorkspaces,
    queryKey: ['workspaces'],
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

export function useGenerationJobs(options?: { refetchInterval?: number }) {
  const hasSession = useAuthStore((state) => Boolean(state.session?.token))
  return useQuery({
    enabled: hasSession,
    queryFn: listGenerationJobs,
    queryKey: ['generation-jobs'],
    refetchInterval: options?.refetchInterval ?? 10_000,
    retry: shouldRetryAuthedQuery,
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

export function useSmokeChatProviderStream() {
  return useMutation({
    mutationFn: () => smokeChatProviderStream(),
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

type UpdateOrganizationMemberRoleInput = {
  userId: string
  role: OrganizationRole
}

export function useOrganizationMembers(enabled = true) {
  return useQuery({
    enabled,
    queryFn: listOrganizationMembers,
    queryKey: ['organization-members'],
  })
}

export function useUpdateOrganizationMemberRole() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, role }: UpdateOrganizationMemberRoleInput) =>
      updateOrganizationMemberRole(userId, { role } satisfies UpdateOrganizationMemberRoleRequest),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-members'] })
      queryClient.invalidateQueries({ queryKey: ['auth-session'] })
    },
  })
}

export function useRemoveOrganizationMember() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => removeOrganizationMember(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-members'] })
      queryClient.invalidateQueries({ queryKey: ['auth-session'] })
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

export function useChangePassword() {
  return useMutation({
    mutationFn: (request: ChangePasswordRequest) => changePassword(request),
  })
}

export function useUserAPIKeys(enabled = true) {
  return useQuery({
    enabled,
    queryFn: listUserAPIKeys,
    queryKey: ['user-api-keys'],
  })
}

export function useCreateUserAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateUserAPIKeyRequest) => createUserAPIKey(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['user-api-keys'] }),
  })
}

export function useUpdateUserAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ keyId, request }: { keyId: string; request: UpdateUserAPIKeyRequest }) =>
      updateUserAPIKey(keyId, request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['user-api-keys'] }),
  })
}

export function useToggleUserAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ keyId, request }: { keyId: string; request: ToggleUserAPIKeyRequest }) =>
      toggleUserAPIKey(keyId, request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['user-api-keys'] }),
  })
}

export function useDeleteUserAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (keyId: string) => deleteUserAPIKey(keyId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['user-api-keys'] }),
  })
}

export function useWallet(enabled = true) {
  return useQuery({
    enabled,
    queryFn: fetchWalletSnapshot,
    queryKey: ['wallet'],
  })
}

export function useWalletTransactions(
  params: { limit?: number; offset?: number; kind?: WalletKind | WalletKind[] } = {},
  enabled = true,
) {
  const kindKey = Array.isArray(params.kind) ? [...params.kind].sort().join(',') : params.kind ?? ''
  return useQuery({
    enabled,
    queryFn: () => fetchWalletTransactions(params),
    queryKey: ['wallet', 'transactions', params.limit ?? 50, params.offset ?? 0, kindKey],
  })
}

export function useCreditWallet() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: WalletMutationRequest) => creditWallet(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] })
    },
  })
}

export function useDebitWallet() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: WalletMutationRequest) => debitWallet(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] })
    },
  })
}

export function useNotifications(
  params: { limit?: number; offset?: number; unread_only?: boolean } = {},
  enabled = true,
) {
  const hasSession = useAuthStore((state) => Boolean(state.session?.token))
  return useQuery({
    enabled: enabled && hasSession,
    queryFn: () => fetchNotifications(params),
    queryKey: ['notifications', params.limit ?? 50, params.offset ?? 0, params.unread_only ?? false],
    refetchInterval: 15000,
    retry: shouldRetryAuthedQuery,
  })
}

export function useMarkNotificationAsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (notificationId: string) => markNotificationAsRead(notificationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useMarkAllNotificationsAsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => markAllNotificationsAsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useOperationCosts(enabled = true) {
  return useQuery({
    enabled,
    queryFn: getOperationCosts,
    queryKey: ['operation-costs'],
  })
}

export function useChat() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (variables: { episodeId: string; request: ChatMessageRequest }) =>
      sendChatMessage(variables.episodeId, variables.request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet-snapshot'] })
      queryClient.invalidateQueries({ queryKey: ['operation-costs'] })
    },
  })
}

export function useChargeWallet() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: ChargeWalletRequest) => chargeWallet(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet-snapshot'] })
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useInitiateChargeWallet() {
  return useMutation({
    mutationFn: (request: ChargeInitiateRequest) => initiateChargeWallet(request),
  })
}

export function useAdminOperationCosts(enabled = false) {
  return useQuery({
    enabled,
    queryFn: getAdminOperationCosts,
    queryKey: ['admin', 'operation-costs'],
  })
}

export function useUpdateAdminOperationCosts() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: UpdateOperationCostsRequest) => updateAdminOperationCosts(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'operation-costs'] })
      queryClient.invalidateQueries({ queryKey: ['operation-costs'] })
    },
  })
}

export function useAdminOperationCostHistory(operationType: string, enabled = false) {
  return useQuery({
    enabled,
    queryFn: () => getAdminOperationCostHistory(operationType),
    queryKey: ['admin', 'operation-costs', operationType, 'history'],
  })
}

// Phase 10: Billing Reports Hooks
export function useAdminBillingReports(params: { limit?: number; offset?: number } = {}, enabled = false) {
  return useQuery({
    enabled,
    queryFn: () => getAdminBillingReports(params),
    queryKey: ['admin', 'billing-reports', params.limit, params.offset],
  })
}

export function useGenerateAdminBillingReport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: GenerateBillingReportRequest) => generateAdminBillingReport(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'billing-reports'] })
    },
  })
}

export function useAdminBillingReportByID(reportID: string, enabled = false) {
  return useQuery({
    enabled,
    queryFn: () => getAdminBillingReportByID(reportID),
    queryKey: ['admin', 'billing-reports', reportID],
  })
}

export function useAdminBillingReportSummary(reportID: string, enabled = false) {
  return useQuery({
    enabled,
    queryFn: () => getAdminBillingReportSummary(reportID),
    queryKey: ['admin', 'billing-reports', reportID, 'summary'],
  })
}


export function useBatchGenerateShots() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      episodeId,
      request,
    }: {
      episodeId: string
      request: BatchGenerateShotsRequest
    }) => batchGenerateShots(episodeId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace'] })
    },
  })
}

export function useDeleteAsset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ assetId }: { assetId: string; episodeId: string }) =>
      deleteAsset(assetId),
    onSuccess: (_void, variables) => {
      queryClient.invalidateQueries({ queryKey: ['assets', variables.episodeId] })
      queryClient.invalidateQueries({ queryKey: ['storyboard-workspace', variables.episodeId] })
    },
  })
}

export function useRetryGenerationJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (jobId: string) => retryGenerationJob(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
    },
  })
}

export function useUpdateGenerationJobPriority() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ jobId, priority }: { jobId: string; priority: number }) =>
      updateGenerationJobPriority(jobId, priority),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
    },
  })
}

export function usePauseEpisodeQueue() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => pauseEpisodeQueue(episodeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
    },
  })
}

export function useResumeEpisodeQueue() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: string) => resumeEpisodeQueue(episodeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['generation-jobs'] })
    },
  })
}

// Redemption Code hooks
export function useRedeemCode() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: RedeemCodeRequest) => redeemCode(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] })
    },
  })
}

export function useGenerateRedemptionCodes() {
  return useMutation({
    mutationFn: (req: GenerateRedemptionCodesRequest) => generateRedemptionCodes(req),
  })
}

export function useGetCampaignStats(campaignId?: string) {
  return useQuery({
    queryKey: ['redemption-campaign-stats', campaignId],
    queryFn: async () => {
      if (!campaignId) {
        throw new Error('Campaign ID is required')
      }
      return getCampaignStats(campaignId)
    },
    enabled: !!campaignId,
  })
}

// Short Video hooks
export function useShortVideoTemplates() {
  return useQuery({
    queryKey: ['short-video-templates'],
    queryFn: () => listShortVideoTemplates(),
  })
}

export function useShortVideoTemplate(templateId?: string) {
  return useQuery({
    queryKey: ['short-video-template', templateId],
    queryFn: async () => {
      if (!templateId) {
        throw new Error('Template ID is required')
      }
      return getShortVideoTemplate(templateId)
    },
    enabled: !!templateId,
  })
}

export function useCreateShortVideoTemplate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateShortVideoTemplateRequest) => createShortVideoTemplate(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['short-video-templates'] })
    },
  })
}

export function useDeleteShortVideoTemplate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (templateId: string) => deleteShortVideoTemplate(templateId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['short-video-templates'] })
    },
  })
}

export function useShortVideos(limit = 20, offset = 0) {
  return useQuery({
    queryKey: ['short-videos', limit, offset],
    queryFn: () => listShortVideos(limit, offset),
    refetchInterval: 2000, // Refetch every 2 seconds
  })
}

export function useShortVideo(videoId?: string, options?: { refetchInterval?: number }) {
  return useQuery({
    queryKey: ['short-video', videoId],
    queryFn: async () => {
      if (!videoId) {
        throw new Error('Video ID is required')
      }
      return getShortVideo(videoId)
    },
    enabled: !!videoId,
    refetchInterval: options?.refetchInterval ?? 2000, // Auto-refetch every 2 seconds for real-time status
  })
}

export function useCreateShortVideo() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateShortVideoRequest) => createShortVideo(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['short-videos'] })
    },
  })
}

export function useDeleteShortVideo() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (videoId: string) => deleteShortVideo(videoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['short-videos'] })
    },
  })
}

// Batch Submission hooks
export function useBatchSubmissions(limit = 20, offset = 0) {
  return useQuery({
    queryKey: ['batch-submissions', limit, offset],
    queryFn: () => listBatchSubmissions(limit, offset),
    refetchInterval: 5000, // Refetch every 5 seconds
  })
}

export function useBatchSubmission(batchId?: string) {
  return useQuery({
    queryKey: ['batch-submission', batchId],
    queryFn: async () => {
      if (!batchId) {
        throw new Error('Batch ID is required')
      }
      return getBatchSubmission(batchId)
    },
    enabled: !!batchId,
    refetchInterval: 2000, // Auto-refetch every 2 seconds for real-time status
  })
}

export function useCreateBatchSubmission() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateBatchSubmissionRequest) => createBatchSubmission(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batch-submissions'] })
    },
  })
}

export function useCancelBatchSubmission() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (batchId: string) => cancelBatchSubmission(batchId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batch-submissions'] })
    },
  })
}

export function useRetryBatchVideo() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ batchId, videoId }: { batchId: string; videoId: string }) => retryBatchVideo(batchId, videoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batch-submissions'] })
    },
  })
}
