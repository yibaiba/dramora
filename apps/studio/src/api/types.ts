export type ProjectStatus = 'draft' | 'active' | 'archived'
export type EpisodeStatus = 'draft' | 'planning' | 'generating' | 'editing' | 'exported' | 'archived'
export type GenerationJobStatus =
  | 'draft'
  | 'preflight'
  | 'queued'
  | 'submitting'
  | 'submitted'
  | 'polling'
  | 'downloading'
  | 'postprocessing'
  | 'needs_review'
  | 'succeeded'
  | 'blocked'
  | 'failed'
  | 'timed_out'
  | 'canceling'
  | 'canceled'
export type AssetStatus = 'draft' | 'generating' | 'ready' | 'failed' | 'archived'
export type ApprovalGateStatus = 'pending' | 'approved' | 'rejected' | 'changes_requested' | 'canceled'

export type AuthUser = {
  id: string
  email: string
  display_name: string
}

export type AuthSession = {
  token: string
  user: AuthUser
  organization_id: string
  role: string
  expires_at: string
  refresh_token?: string
  refresh_expires_at?: string
  current_session_id?: string
}

export type Session = {
  id: string
  organization_id: string
  role: string
  created_at: string
  expires_at: string
  revoked_at?: string | null
  replaced_by_id?: string
}

export type LoginRequest = {
  email: string
  password: string
}

export type RegisterRequest = LoginRequest & {
  display_name: string
  invitation_token?: string
}

export type OrganizationInvitation = {
  id: string
  organization_id: string
  email: string
  role: 'owner' | 'admin' | 'editor' | 'viewer'
  token: string
  status: 'pending' | 'accepted' | 'revoked'
  invited_by_user_id?: string
  expires_at: string
  accepted_at?: string | null
  accepted_by_user_id?: string
  created_at: string
}

export type CreateInvitationRequest = {
  email: string
  role?: 'owner' | 'admin' | 'editor' | 'viewer'
}

export type InvitationAuditEvent = {
  id: string
  organization_id: string
  invitation_id: string
  action: 'created' | 'accepted' | 'revoked' | 'resent'
  actor_user_id?: string
  actor_email?: string
  email: string
  role: 'owner' | 'admin' | 'editor' | 'viewer'
  note?: string
  created_at: string
}

export type Project = {
  id: string
  organization_id: string
  name: string
  description: string
  status: ProjectStatus
  created_at: string
  updated_at: string
}

export type Episode = {
  id: string
  project_id: string
  number: number
  title: string
  status: EpisodeStatus
  created_at: string
  updated_at: string
}

export type GenerationJob = {
  id: string
  project_id: string
  episode_id: string
  workflow_run_id: string
  provider: string
  model: string
  task_type: string
  status: GenerationJobStatus
  result_asset_id: string
  created_at: string
  updated_at: string
}

export type GenerationJobEvent = {
  id: string
  generation_job_id: string
  status: GenerationJobStatus
  message: string
  created_at: string
}

export type GenerationJobRecoverySummary = {
  is_terminal: boolean
  is_recoverable: boolean
  current_status: GenerationJobStatus
  status_entered_at: string
  last_event_at: string
  status_event_count: number
  total_event_count: number
  next_hint: string
}

export type GenerationJobRecovery = {
  generation_job: GenerationJob
  events: GenerationJobEvent[]
  summary: GenerationJobRecoverySummary
}

export type PromptPackRecoveryJob = {
  generation_job: GenerationJob
  summary: GenerationJobRecoverySummary
}

export type PromptPackRecoverySummary = {
  jobs_total: number
  terminal_count: number
  recoverable_count: number
  in_flight_count: number
  has_recoverable: boolean
  last_event_at: string
  next_hint: string
  latest_status?: GenerationJobStatus
  latest_status_job_id?: string
  latest_status_job_time?: string
}

export type PromptPackRecovery = {
  prompt_pack: ShotPromptPack
  jobs: PromptPackRecoveryJob[]
  summary: PromptPackRecoverySummary
}

export type WorkflowRun = {
  id: string
  project_id: string
  episode_id: string
  status: 'draft' | 'running' | 'waiting_approval' | 'succeeded' | 'failed' | 'canceled'
  checkpoint_summary?: WorkflowCheckpointSummary
  node_runs?: WorkflowNodeRun[]
  created_at: string
  updated_at: string
}

export type WorkflowCheckpointSummary = {
  sequence: number
  saved_at: string
  completed_nodes: number
  waiting_nodes: number
  running_nodes: number
  failed_nodes: number
  skipped_nodes: number
  blackboard_roles: string[]
}

export type WorkflowNodeRun = {
  node_id: string
  kind: string
  status: 'pending' | 'running' | 'waiting_approval' | 'succeeded' | 'failed' | 'skipped' | 'canceled'
  summary: string
  highlights: string[]
  error_message: string
  upstream_node_ids: string[]
}

export type ApprovalGate = {
  id: string
  project_id: string
  episode_id: string
  workflow_run_id: string
  gate_type: string
  subject_type: string
  subject_id: string
  status: ApprovalGateStatus
  reviewed_by: string
  review_note: string
  reviewed_at: string
  created_at: string
  updated_at: string
}

export type Timeline = {
  id: string
  episode_id: string
  status: 'draft' | 'saved' | 'exporting' | 'exported'
  version: number
  duration_ms: number
  tracks: TimelineTrack[]
  created_at: string
  updated_at: string
}

export type TimelineTrack = {
  id: string
  kind: string
  name: string
  position: number
  clips: TimelineClip[]
  created_at: string
  updated_at: string
}

export type TimelineClip = {
  id: string
  asset_id: string
  kind: string
  start_ms: number
  duration_ms: number
  trim_start_ms: number
  created_at: string
  updated_at: string
}

export type StoryAnalysis = {
  id: string
  project_id: string
  episode_id: string
  story_source_id: string
  workflow_run_id: string
  generation_job_id: string
  version: number
  status: 'generated' | 'approved'
  summary: string
  themes: string[]
  character_seeds: string[]
  scene_seeds: string[]
  prop_seeds: string[]
  outline: StoryBeat[]
  agent_outputs: StoryAgentOutput[]
  created_at: string
  updated_at: string
}

export type StorySource = {
  id: string
  project_id: string
  episode_id: string
  source_type: 'idea' | 'outline' | 'novel' | 'script' | 'file' | 'url'
  title: string
  content_text: string
  language: string
  created_at: string
  updated_at: string
}

export type StoryBeat = {
  code: string
  title: string
  summary: string
  visual_goal: string
}

export type StoryAgentOutput = {
  role: string
  status: string
  output: string
  highlights: string[]
}

export type StoryMap = {
  characters: StoryMapItem[]
  scenes: StoryMapItem[]
  props: StoryMapItem[]
}

export type CharacterBiblePalette = {
  skin: string
  hair: string
  accent: string
  eyes: string
  costume: string
}

export type CharacterBibleReferenceAsset = {
  angle: string
  asset_id: string
}

export type CharacterBible = {
  anchor: string
  palette: CharacterBiblePalette
  expressions: string[]
  reference_angles: string[]
  reference_assets: CharacterBibleReferenceAsset[]
  wardrobe: string
  notes: string
}

export type StoryMapItem = {
  id: string
  project_id: string
  episode_id: string
  code: string
  name: string
  description: string
  character_bible?: CharacterBible
  created_at: string
  updated_at: string
}

export type SaveCharacterBibleRequest = {
  character_bible: CharacterBible
}

export type StoryboardShot = {
  id: string
  project_id: string
  episode_id: string
  scene_id: string
  code: string
  title: string
  description: string
  prompt: string
  position: number
  duration_ms: number
  created_at: string
  updated_at: string
}

export type StoryboardShotPromptPackSummary = {
  id: string
  shot_id: string
  provider: string
  model: string
  preset: string
  task_type: ShotPromptPack['task_type']
  updated_at: string
}

export type StoryboardWorkspaceShot = StoryboardShot & {
  scene: StoryMapItem | null
  prompt_pack: StoryboardShotPromptPackSummary | null
  latest_generation_job: GenerationJob | null
}

export type StoryboardWorkspaceSummary = {
  analysis_count: number
  story_map_ready: boolean
  ready_assets_count: number
  pending_approval_gates_count: number
}

export type StoryboardWorkspace = {
  episode_id: string
  summary: StoryboardWorkspaceSummary
  story_map: StoryMap
  storyboard_shots: StoryboardWorkspaceShot[]
  assets: Asset[]
  approval_gates: ApprovalGate[]
  generation_jobs: GenerationJob[]
}

export type UpdateStoryboardShotRequest = {
  title: string
  description?: string
  prompt: string
  duration_ms: number
}

export type PromptTimeSlice = {
  start_ms: number
  end_ms: number
  prompt: string
  camera_work: string
  shot_size: string
  visual_focus: string
}

export type PromptReferenceBinding = {
  token: string
  role: 'first_frame' | 'last_frame' | 'reference_image'
  asset_id: string
  kind: string
  uri: string
}

export type ShotPromptPack = {
  id: string
  project_id: string
  episode_id: string
  shot_id: string
  provider: string
  model: string
  preset: string
  task_type: 'text_to_video' | 'image_to_video' | 'first_last_frame_to_video'
  direct_prompt: string
  negative_prompt: string
  time_slices: PromptTimeSlice[]
  reference_bindings: PromptReferenceBinding[]
  params: Record<string, string | number | boolean>
  created_at: string
  updated_at: string
}

export type SaveShotPromptPackRequest = {
  direct_prompt: string
}

export type Asset = {
  id: string
  project_id: string
  episode_id: string
  kind: string
  purpose: string
  uri: string
  status: AssetStatus
  created_at: string
  updated_at: string
}

export type SeedEpisodeProductionResponse = {
  approval_gates: ApprovalGate[]
  assets: Asset[]
  story_map: StoryMap
  storyboard_shots: StoryboardShot[]
}

export type Export = {
  id: string
  timeline_id: string
  status: 'queued' | 'rendering' | 'succeeded' | 'failed' | 'canceled'
  format: string
  created_at: string
  updated_at: string
}

export type ExportRecoveryEvent = {
  status: Export['status']
  message: string
  created_at: string
}

export type ExportRecoverySummary = {
  is_terminal: boolean
  is_recoverable: boolean
  current_status: Export['status']
  status_entered_at: string
  last_event_at: string
  total_event_count: number
  next_hint: string
}

export type ExportRecovery = {
  export: Export
  events: ExportRecoveryEvent[]
  summary: ExportRecoverySummary
}

export type AssetRecoveryEvent = {
  status: Asset['status']
  message: string
  created_at: string
}

export type AssetRecoverySummary = {
  is_terminal: boolean
  is_recoverable: boolean
  is_locked: boolean
  current_status: Asset['status']
  status_entered_at: string
  last_event_at: string
  total_event_count: number
  next_hint: string
}

export type AssetRecovery = {
  asset: Asset
  events: AssetRecoveryEvent[]
  summary: AssetRecoverySummary
}

export type CreateProjectRequest = {
  name: string
  description?: string
}

export type CreateStorySourceRequest = {
  source_type?: StorySource['source_type']
  title?: string
  content_text: string
  language?: string
}

export type StartStoryAnalysisResponse = {
  workflow_run: WorkflowRun
  generation_job: GenerationJob
}

export type SaveTimelineRequest = {
  duration_ms: number
  tracks?: SaveTimelineTrackRequest[]
}

export type SaveTimelineTrackRequest = {
  kind: string
  name: string
  position: number
  clips?: SaveTimelineClipRequest[]
}

export type SaveTimelineClipRequest = {
  asset_id?: string
  kind: string
  start_ms: number
  duration_ms: number
  trim_start_ms?: number
}

export type CreateEpisodeRequest = {
  title: string
  number?: number
}

export type ApprovalGateReviewRequest = {
  reviewed_by?: string
  review_note?: string
}

export type ProviderCapability = 'chat' | 'image' | 'video' | 'audio'

export type ProviderType = 'openai' | 'anthropic' | 'mock' | 'seedance'

export type ProviderConfig = {
  id: string
  capability: ProviderCapability
  provider_type: ProviderType
  base_url: string
  api_key: string
  model: string
  credits_per_unit: number
  credit_unit: string
  timeout_ms: number
  max_retries: number
  is_enabled: boolean
  updated_at: string
  updated_by: string
}

export type SaveProviderConfigRequest = {
  capability: ProviderCapability
  provider_type: ProviderType
  base_url: string
  api_key: string
  model: string
  credits_per_unit: number
  credit_unit: string
  timeout_ms: number
  max_retries: number
}

export type TestProviderResult = {
  ok: boolean
  capability?: string
  provider_type?: string
  model: string
  probe?: string
  latency_ms: number
  error?: string
}

export type SmokeChatResult = {
  ok: boolean
  capability: string
  provider_type?: string
  model: string
  content?: string
  token_count?: number
  latency_ms: number
  streamed?: boolean
  chunk_count?: number
  error?: string
}

export type WorkerMetricsSnapshot = {
  generation_org_unresolved_skips: number
  export_org_unresolved_skips: number
  last_skip_kind?: string
  last_skip_reason?: string
  last_skip_at?: string
  source?: 'local' | 'aggregated'
}

export type LLMTelemetryEvent = {
  started_at: string
  vendor: string
  model: string
  role: string
  capability?: 'chat' | 'image' | 'video' | 'audio' | string
  mode: 'complete' | 'stream' | 'generate' | 'synthesize' | 'submit' | 'poll' | string
  duration_ms: number
  token_count: number
  success: boolean
  error_message?: string
}

export type LLMTelemetrySnapshot = {
  total_calls: number
  success_calls: number
  error_calls: number
  by_vendor: Record<string, number>
  avg_duration_ms_by_vendor: Record<string, number>
  by_capability: Record<string, number>
  avg_duration_ms_by_capability?: Record<string, number>
  errors_by_vendor?: Record<string, number>
  errors_by_capability?: Record<string, number>
  recent_events: LLMTelemetryEvent[]
  last_event_at?: string
  window?: LLMTelemetryWindowSnapshot
}

export type LLMTelemetryWindowSnapshot = {
  days: number
  since_day_utc: string
  total_calls: number
  error_calls: number
  by_vendor: Record<string, number>
  by_capability: Record<string, number>
  errors_by_vendor: Record<string, number>
  errors_by_capability: Record<string, number>
  avg_duration_ms_by_vendor: Record<string, number>
  avg_duration_ms_by_capability: Record<string, number>
}

export type ProviderAuditEvent = {
  id: string
  organization_id: string
  action: 'save' | 'test' | string
  actor_user_id?: string
  actor_email?: string
  capability: string
  provider_type: string
  model?: string
  success: boolean
  message?: string
  created_at: string
}

export type ProviderAuditPage = {
  events: ProviderAuditEvent[]
  has_more: boolean
}

export type WalletKind = 'credit' | 'debit' | 'refund' | 'adjust'

export type Wallet = {
  organization_id: string
  balance: number
  updated_at: string
}

export type WalletTransaction = {
  id: string
  organization_id: string
  kind: WalletKind
  direction: 1 | -1
  amount: number
  reason?: string
  ref_type?: string
  ref_id?: string
  balance_after: number
  actor_user_id?: string
  created_at: string
}

export type WalletSnapshot = {
  wallet: Wallet
  recent_transactions: WalletTransaction[]
}

export type WalletTransactionPage = {
  transactions: WalletTransaction[]
  has_more: boolean
}

export type WalletMutationRequest = {
  amount: number
  reason?: string
  ref_type?: string
  ref_id?: string
}

export type OperationType = 'chat' | 'story_analysis' | 'image_generation' | 'video_generation' | 'storyboard_edit' | 'character_edit' | 'scene_edit'

export type OperationCost = {
  type: OperationType
  cost: number
}

export type NotificationKind = 'wallet_credit' | 'wallet_debit' | 'invitation_created' | 'invitation_resent' | 'provider_config_save'

export type Notification = {
  id: string
  organization_id: string
  recipient_user_id?: string
  kind: NotificationKind
  title: string
  body: string
  metadata?: Record<string, any>
  read_at?: string
  created_at: string
}

export type NotificationsPage = {
  notifications: Notification[]
  has_more: boolean
  unread_count: number
}

export type ChatMessage = {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export type ChatMessageRequest = {
  messages: ChatMessage[]
  provider?: string
}

export type ChatResponse = {
  id: string
  content: string
  token_usage?: {
    input_tokens?: number
    output_tokens?: number
  }
  latency_ms: number
}

export type ChargeWalletRequest = {
  amount: number
  description?: string
  payment_method?: string
}

export type ChargeInitiateRequest = {
  amount: number
  currency: string
}

export type ChargeInitiateResponse = {
  sessionId: string
  url: string
  orderId: string
}

export type OperationCostAdminDTO = {
  id: number
  operation_type: string
  organization_id: string
  credits_cost: number
  effective_at: number
  updated_at: number
}

export type OperationCostHistoryDTO = {
  id: number
  operation_type: string
  organization_id: string
  old_cost: number | null
  new_cost: number
  effective_at: number
  reason: string | null
  changed_by: string
  changed_at: number
}

export type UpdateOperationCostRequest = {
  operation_type: string
  credits_cost: number
}

export type UpdateOperationCostsRequest = {
  updates: UpdateOperationCostRequest[]
  reason?: string
}
