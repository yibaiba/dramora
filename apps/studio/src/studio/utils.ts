import type {
  ApprovalGate,
  ApprovalGateStatus,
  Asset,
  Episode,
  Export,
  GenerationJob,
  SaveTimelineRequest,
  StoryAnalysis,
  StoryMap,
  StoryMapItem,
  StoryboardWorkspaceShot,
  StoryboardShot,
  Timeline,
  WorkflowNodeRun,
  WorkflowRun,
} from '../api/types'
import { demoShots } from './mockData'
import type { FlowStepState, ShotDraft, StudioShot } from './types'

export type ApprovalGateOverviewTone =
  | 'approved'
  | 'canceled'
  | 'changes'
  | 'idle'
  | 'pending'
  | 'rejected'

export type ApprovalGateOverview = {
  total: number
  pending: number
  approved: number
  changesRequested: number
  rejected: number
  canceled: number
  headline: string
  description: string
  tone: ApprovalGateOverviewTone
}

export type StoryboardCharacterReferenceAsset = {
  angle: string
  assetId: string
  assetLabel: string
  assetUri: string
}

export type StoryboardCharacterReference = {
  anchor: string
  characterId: string
  code: string
  name: string
  notes: string
  referenceAssets: StoryboardCharacterReferenceAsset[]
  searchTerms: string[]
  wardrobe: string
}

export type StoryboardCharacterReferenceSelection = {
  mode: 'empty' | 'fallback' | 'matched'
  references: StoryboardCharacterReference[]
}

export type StoryMapNodeLinkedShot = {
  coverage: 'covered' | 'linked'
  reason: string
  shot: StudioShot
}

export function filterShots(shots: StudioShot[], query: string): StudioShot[] {
  const keyword = query.trim().toLowerCase()
  if (!keyword) return shots
  return shots.filter((shot) =>
    [
      shot.code,
      shot.description,
      shot.prompt,
      shot.sceneCode,
      shot.sceneName,
      shot.title,
      ...shot.tags,
    ].some((value) => value.toLowerCase().includes(keyword)),
  )
}

export function createLocalShot(position: number): StudioShot {
  const code = position.toString().padStart(2, '0')
  return {
    code,
    description: '本地草稿镜头，可先改写提示词和导演备注。',
    durationMS: 4000,
    key: `local-${Date.now()}`,
    progress: 0,
    prompt: '请在右侧提示词面板补充人物、场景、镜头运动、光影和风格约束。',
    sceneCode: `SC${Math.max(1, Math.ceil(position / 3)).toString().padStart(2, '0')}`,
    sceneName: '待定镜头',
    status: 'draft',
    tags: ['草稿'],
    thumbnail: thumbnailByIndex(position),
    title: `草稿镜头 ${code}`,
  }
}

export function mapDisplayShots(shots: StoryboardShot[]): StudioShot[] {
  if (shots.length === 0) return demoShots
  return shots.map((shot, index) => ({
    code: shot.code || `${index + 1}`.padStart(2, '0'),
    description: shot.description || '系统生成的分镜镜头',
    durationMS: shot.duration_ms,
    id: shot.id,
    key: shot.id,
    progress: index % 3 === 1 ? 72 : index % 3 === 2 ? 0 : 100,
    prompt: shot.prompt,
    sceneCode: `SC${((index % 3) + 1).toString().padStart(2, '0')}`,
    sceneName: shot.title,
    status: index % 3 === 1 ? 'generating' : index % 3 === 2 ? 'queued' : 'prompt_ready',
    tags: ['云澜', '白璃', '长老'].slice(0, (index % 3) + 1),
    thumbnail: thumbnailByIndex(index),
    title: shot.title,
  }))
}

export function mapTimelineDisplayShots(
  timeline: Timeline,
  storyboardShots: StoryboardShot[],
): StudioShot[] {
  const fallbackShots = mapDisplayShots(storyboardShots)
  const videoTrack = timeline.tracks.find((track) => track.kind === 'video') ?? timeline.tracks[0]
  if (!videoTrack || videoTrack.clips.length === 0) return fallbackShots

  return videoTrack.clips.map((clip, index) => {
    const fallback = fallbackShots[index] ?? createLocalShot(index + 1)
    return {
      ...fallback,
      durationMS: clip.duration_ms,
      key: clip.id,
    }
  })
}

export function mapWorkspaceDisplayShots(shots: StoryboardWorkspaceShot[]): StudioShot[] {
  if (shots.length === 0) return demoShots
  return shots.map((shot, index) => {
    const latestJob = shot.latest_generation_job
    const hasPromptPack = Boolean(shot.prompt_pack)
    return {
      code: shot.code || `${index + 1}`.padStart(2, '0'),
      description: shot.description || '系统生成的分镜镜头',
      durationMS: shot.duration_ms,
      id: shot.id,
      key: shot.id,
      progress: workspaceShotProgress(latestJob?.status, hasPromptPack),
      prompt: shot.prompt,
      sceneCode: shot.scene?.code ?? `SC${((index % 3) + 1).toString().padStart(2, '0')}`,
      sceneName: shot.scene?.name ?? shot.title,
      status: workspaceShotStatus(latestJob?.status, hasPromptPack),
      tags: [
        shot.scene?.code,
        hasPromptPack ? 'SD2 Fast' : undefined,
        latestJob?.task_type?.replaceAll('_', ' '),
      ].filter((value): value is string => Boolean(value)),
      thumbnail: thumbnailByIndex(index),
      title: shot.title,
      latestGenerationJobId: latestJob?.id,
    }
  })
}

export function buildStoryboardCharacterReferences(
  storyMap: StoryMap | undefined,
  assets: Asset[],
): StoryboardCharacterReference[] {
  if (!storyMap) return []

  const readyAssetsById = new Map(
    assets.filter((asset) => asset.status === 'ready').map((asset) => [asset.id, asset]),
  )

  return storyMap.characters.flatMap((character) => {
    const bible = character.character_bible
    if (!bible) return []

    const referenceAssets = bible.reference_assets.flatMap((reference) => {
      const asset = readyAssetsById.get(reference.asset_id)
      if (!asset) return []

      return [{
        angle: reference.angle,
        assetId: asset.id,
        assetLabel: asset.uri.replace('manmu://episodes/', ''),
        assetUri: asset.uri,
      }]
    })

    if (referenceAssets.length === 0) return []

    return [{
      anchor: bible.anchor,
      characterId: character.id,
      code: character.code,
      name: character.name,
      notes: bible.notes,
      referenceAssets,
      searchTerms: characterSearchTerms(character),
      wardrobe: bible.wardrobe,
    }]
  })
}

export function selectShotCharacterReferences(
  shot: StudioShot | undefined,
  references: StoryboardCharacterReference[],
): StoryboardCharacterReferenceSelection {
  if (!shot || references.length === 0) {
    return { mode: 'empty', references: [] }
  }

  const haystack = normalizeReferenceSearch([
    shot.code,
    shot.description,
    shot.prompt,
    shot.sceneCode,
    shot.sceneName,
    shot.title,
    ...shot.tags,
  ].join(' '))

  const matched = references.filter((reference) =>
    reference.searchTerms.some((term) => haystack.includes(term)),
  )

  if (matched.length > 0) {
    return { mode: 'matched', references: matched }
  }

  return { mode: 'fallback', references }
}

export function selectShotsForStoryMapNode(
  node: StoryMapItem | undefined,
  kind: 'character' | 'scene' | 'prop' | undefined,
  shots: StudioShot[],
  references: StoryboardCharacterReference[],
): StoryMapNodeLinkedShot[] {
  if (!node || !kind || shots.length === 0) {
    return []
  }

  if (kind === 'character') {
    const matchedReference = references.find((reference) => reference.characterId === node.id)
    const terms = matchedReference?.searchTerms.length
      ? matchedReference.searchTerms
      : storyMapNodeSearchTerms(node)

    return shots.flatMap((shot) => {
      const haystack = shotSearchText(shot)
      if (!terms.some((term) => haystack.includes(term))) {
        return []
      }

      return [{
        coverage: matchedReference?.referenceAssets.length ? 'covered' : 'linked',
        reason: matchedReference?.referenceAssets.length
          ? '镜头正文已命中该角色，Prompt 可直接复用引用图。'
          : '镜头正文已命中该角色，但还没有可复用的引用图角度。',
        shot,
      }]
    })
  }

  return shots.flatMap((shot) => {
    const reason = matchStoryMapNodeShotReason(node, kind, shot)
    if (!reason) {
      return []
    }

    return [{
      coverage: 'linked',
      reason,
      shot,
    }]
  })
}

function characterSearchTerms(character: StoryMapItem): string[] {
  const bible = character.character_bible
  return Array.from(
    new Set(
      [character.code, character.name, character.description, bible?.anchor]
        .map((value) => normalizeReferenceSearch(value ?? ''))
        .filter((value) => value.length >= 2),
    ),
  )
}

function normalizeReferenceSearch(value: string): string {
  return value.trim().toLowerCase()
}

function shotSearchText(shot: StudioShot): string {
  return normalizeReferenceSearch(
    [
      shot.code,
      shot.description,
      shot.prompt,
      shot.sceneCode,
      shot.sceneName,
      shot.title,
      ...shot.tags,
    ].join(' '),
  )
}

function storyMapNodeSearchTerms(node: StoryMapItem): string[] {
  return Array.from(
    new Set(
      [node.code, node.name, node.description, node.character_bible?.anchor]
        .map((value) => normalizeReferenceSearch(value ?? ''))
        .filter((value) => value.length >= 2),
    ),
  )
}

function matchStoryMapNodeShotReason(
  node: StoryMapItem,
  kind: 'scene' | 'prop',
  shot: StudioShot,
): string | undefined {
  const terms = storyMapNodeSearchTerms(node)
  const haystack = shotSearchText(shot)
  const normalizedCode = normalizeReferenceSearch(node.code)
  const normalizedName = normalizeReferenceSearch(node.name)

  if (kind === 'scene') {
    if (normalizedCode && normalizeReferenceSearch(shot.sceneCode).includes(normalizedCode)) {
      return '镜头场景编码直接命中了这个场景节点。'
    }
    if (normalizedName && normalizeReferenceSearch(shot.sceneName).includes(normalizedName)) {
      return '镜头场景标题直接命中了这个场景节点。'
    }
    if (terms.some((term) => haystack.includes(term))) {
      return '镜头描述或提示词里提到了这个场景。'
    }
    return undefined
  }

  if (terms.some((term) => haystack.includes(term))) {
    return '镜头描述或提示词里提到了这个道具。'
  }

  return undefined
}

export function buildTimelineRequest(shots: StudioShot[]): SaveTimelineRequest {
  return {
    duration_ms: shots.reduce((total, shot) => total + shot.durationMS, 0),
    tracks: [
      {
        clips: shots.map((shot, index) => ({
          asset_id: undefined,
          duration_ms: shot.durationMS,
          kind: 'shot',
          start_ms: shots
            .slice(0, index)
            .reduce((total, item) => total + item.durationMS, 0),
          trim_start_ms: 0,
        })),
        kind: 'video',
        name: '分镜视频轨',
        position: 1,
      },
    ],
  }
}

export function approvalGateTypeLabel(gateType: string): string {
  const labels: Record<string, string> = {
    character_lock: '角色锁定',
    final_timeline: '时间线终审',
    prop_lock: '道具锁定',
    scene_lock: '场景锁定',
    story_direction: '故事方向',
    storyboard_approval: '分镜审批',
  }
  return labels[gateType] ?? gateType.replaceAll('_', ' ')
}

export function approvalGateStatusLabel(status: ApprovalGateStatus): string {
  const labels: Record<ApprovalGateStatus, string> = {
    approved: '已通过',
    canceled: '已取消',
    changes_requested: '待返修',
    pending: '待确认',
    rejected: '已拒绝',
  }
  return labels[status]
}

export function approvalGateStatusTone(
  status: ApprovalGateStatus,
): Exclude<ApprovalGateOverviewTone, 'idle'> {
  const tones: Record<ApprovalGateStatus, Exclude<ApprovalGateOverviewTone, 'idle'>> = {
    approved: 'approved',
    canceled: 'canceled',
    changes_requested: 'changes',
    pending: 'pending',
    rejected: 'rejected',
  }
  return tones[status]
}

export function summarizeApprovalGates(gates: ApprovalGate[]): ApprovalGateOverview {
  const summary = gates.reduce(
    (current, gate) => {
      switch (gate.status) {
        case 'approved':
          current.approved += 1
          break
        case 'canceled':
          current.canceled += 1
          break
        case 'changes_requested':
          current.changesRequested += 1
          break
        case 'pending':
          current.pending += 1
          break
        case 'rejected':
          current.rejected += 1
          break
      }
      return current
    },
    { approved: 0, canceled: 0, changesRequested: 0, pending: 0, rejected: 0 },
  )

  if (gates.length === 0) {
    return {
      ...summary,
      description: '先生成分镜与审批点，导演台才会出现可操作的送审队列。',
      headline: '待建立审批队列',
      tone: 'idle',
      total: 0,
    }
  }

  if (summary.changesRequested > 0) {
    return {
      ...summary,
      description: '已有镜头被打回，需要修改后再进入下一轮确认。',
      headline: `${summary.changesRequested} 个待返修`,
      tone: 'changes',
      total: gates.length,
    }
  }

  if (summary.pending > 0) {
    return {
      ...summary,
      description: '存在待确认审批点，进入昂贵生成前应先人工确认。',
      headline: `${summary.pending} 个待确认`,
      tone: 'pending',
      total: gates.length,
    }
  }

  if (summary.rejected > 0) {
    return {
      ...summary,
      description: '当前有审批点被拒绝，需要重新评估对应资产或分镜输出。',
      headline: `${summary.rejected} 个已拒绝`,
      tone: 'rejected',
      total: gates.length,
    }
  }

  if (summary.approved === gates.length) {
    return {
      ...summary,
      description: '当前工作台审批链路全部通过，可以继续推进生成或时间线编排。',
      headline: '审批全部通过',
      tone: 'approved',
      total: gates.length,
    }
  }

  return {
    ...summary,
    description: '当前审批链路没有待确认项，但仍保留历史审批记录供复盘。',
    headline: `${summary.canceled} 个已取消`,
    tone: 'canceled',
    total: gates.length,
  }
}

export function statusLabel(status: StudioShot['status']): string {
  const labels: Record<StudioShot['status'], string> = {
    approved: '已通过',
    draft: '草稿',
    generating: '生成中',
    prompt_ready: '提示词就绪',
    queued: '排队中',
  }
  return labels[status]
}

export function shotDraftValue(shot: StudioShot, draft?: ShotDraft): ShotDraft {
  return (
    draft ?? {
      description: shot.description,
      durationMS: shot.durationMS,
      title: shot.title || shot.sceneName,
    }
  )
}

export function hasStoryMapItems(storyMap?: StoryMap): boolean {
  if (!storyMap) return false
  return storyMap.characters.length + storyMap.scenes.length + storyMap.props.length > 0
}

export function productionHint({
  activeEpisode,
  hasAnalysis,
  storyMapReady,
}: {
  activeEpisode?: Episode
  hasAnalysis: boolean
  storyMapReady: boolean
}): string {
  if (!activeEpisode) return '先创建项目和剧集。'
  if (!hasAnalysis) return '下一步：启动故事解析。'
  if (!storyMapReady) return '下一步：一键生产资产图谱、候选资产和分镜卡。'
  return '下一步：生成候选资产和分镜卡。'
}

export function productionSteps({
  analysesCount,
  assetsCount,
  gatesCount,
  shotsCount,
  storyMapReady,
}: {
  analysesCount: number
  assetsCount: number
  gatesCount: number
  shotsCount: number
  storyMapReady: boolean
}): Array<{ label: string; state: FlowStepState; value: string }> {
  const hasAnalysis = analysesCount > 0
  const hasAssets = assetsCount > 0
  const hasShots = shotsCount > 0
  return [
    {
      label: '故事解析',
      state: hasAnalysis ? 'done' : 'active',
      value: hasAnalysis ? `${analysesCount} 份` : '等待 worker',
    },
    {
      label: '资产图谱',
      state: stepState(storyMapReady, hasAnalysis),
      value: storyMapReady ? '已生成' : '待生成',
    },
    {
      label: '候选资产',
      state: stepState(hasAssets, storyMapReady),
      value: `${assetsCount} 个`,
    },
    {
      label: '分镜卡',
      state: stepState(hasShots, storyMapReady),
      value: `${shotsCount} 镜`,
    },
    {
      label: '人审关卡',
      state: stepState(gatesCount > 0, hasShots),
      value: `${gatesCount} 个`,
    },
  ]
}

export function jobTaskLabel(taskType: string): string {
  const labels: Record<string, string> = {
    export: '导出',
    first_last_frame_to_video: '首尾帧视频',
    image_to_video: '图生视频',
    story_analysis: '故事解析',
    text_to_video: '文生视频',
  }
  return labels[taskType] ?? taskType.replaceAll('_', ' ')
}

export function agentRoleLabel(role: string): string {
  const labels: Record<string, string> = {
    character_analyst: '人物分析',
    cinematographer: '摄影指导',
    director: '导演',
    outline_planner: '大纲规划',
    prop_analyst: '道具分析',
    scene_analyst: '场景分析',
    screenwriter: '编剧',
    story_analyst: '故事分析',
    voice_subtitle: '配音字幕',
  }
  return labels[role] ?? role.replaceAll('_', ' ')
}

export function agentStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    failed: '失败',
    running: '执行中',
    skipped: '已跳过',
    succeeded: '已完成',
    waiting: '等待中',
  }
  return labels[status] ?? status
}

export function jobStatusLabel(status: GenerationJob['status']): string {
  const labels: Record<GenerationJob['status'], string> = {
    blocked: '已阻塞',
    canceled: '已取消',
    canceling: '取消中',
    downloading: '下载中',
    draft: '草稿',
    failed: '失败',
    needs_review: '待人审',
    polling: '轮询中',
    postprocessing: '后处理',
    preflight: '预检中',
    queued: '排队中',
    submitted: '已提交',
    submitting: '提交中',
    succeeded: '已完成',
    timed_out: '超时',
  }
  return labels[status] ?? status.replaceAll('_', ' ')
}

export function workflowRunStatusLabel(status: WorkflowRun['status']): string {
  const labels: Record<WorkflowRun['status'], string> = {
    canceled: '已取消',
    draft: '草稿',
    failed: '失败',
    running: '运行中',
    succeeded: '已完成',
    waiting_approval: '待审批',
  }
  return labels[status] ?? status.replaceAll('_', ' ')
}

export function workflowNodeStatusLabel(status: WorkflowNodeRun['status']): string {
  const labels: Record<WorkflowNodeRun['status'], string> = {
    canceled: '已取消',
    failed: '失败',
    pending: '待执行',
    running: '运行中',
    skipped: '已跳过',
    succeeded: '已完成',
    waiting_approval: '待审批',
  }
  return labels[status] ?? status.replaceAll('_', ' ')
}

export function workflowNodeLabel(nodeId: string, kind?: string): string {
  const roleLabel = agentRoleLabel(nodeId)
  if (roleLabel !== nodeId.replaceAll('_', ' ')) {
    return roleLabel
  }
  return kind ? kind.replaceAll('_', ' ') : nodeId.replaceAll('_', ' ')
}

export function formatCheckpointSavedAt(savedAt?: string): string {
  if (!savedAt) return '等待首个 checkpoint'
  const date = new Date(savedAt)
  if (Number.isNaN(date.getTime())) {
    return '最近已保存 checkpoint'
  }
  return `最近保存 ${date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })}`
}

export function resolveEpisodeWorkflowRunId(
  episodeId: string | undefined,
  analyses: Pick<StoryAnalysis, 'workflow_run_id'>[],
  jobs: Pick<GenerationJob, 'episode_id' | 'status' | 'task_type' | 'updated_at' | 'workflow_run_id'>[],
): string | undefined {
  const storyAnalysisJobs = jobs
    .filter((job) => job.episode_id === episodeId && job.task_type === 'story_analysis')
    .sort((left, right) => right.updated_at.localeCompare(left.updated_at))
  const activeJob =
    storyAnalysisJobs.find(
      (job) => !['blocked', 'canceled', 'failed', 'succeeded', 'timed_out'].includes(job.status),
    ) ?? storyAnalysisJobs[0]
  return activeJob?.workflow_run_id || analyses[0]?.workflow_run_id
}

export function exportStatusLabel(status: Export['status'] | 'completed' | 'processing' | 'requested'): string {
  const labels: Record<string, string> = {
    canceled: '已取消',
    completed: '已完成',
    failed: '导出失败',
    processing: '导出中',
    queued: '排队中',
    rendering: '渲染中',
    requested: '已提交',
    succeeded: '已完成',
  }
  return labels[status] ?? status.replaceAll('_', ' ')
}

export function thumbnailByIndex(index: number): string {
  const thumbnails = [
    'thumb-cloud',
    'thumb-hero',
    'thumb-muse',
    'thumb-temple',
    'thumb-battle',
    'thumb-beast',
  ]
  return thumbnails[index % thumbnails.length]
}

export function formatDuration(durationMS: number): string {
  return `00:${(durationMS / 1000).toFixed(1).padStart(4, '0')}`
}

function workspaceShotStatus(
  status: GenerationJob['status'] | undefined,
  hasPromptPack: boolean,
): StudioShot['status'] {
  if (status === 'succeeded') return 'approved'
  if (status === 'queued' || status === 'preflight' || status === 'submitting') return 'queued'
  if (
    status === 'submitted' ||
    status === 'polling' ||
    status === 'downloading' ||
    status === 'postprocessing'
  ) {
    return 'generating'
  }
  if (hasPromptPack) return 'prompt_ready'
  return 'draft'
}

function workspaceShotProgress(
  status: GenerationJob['status'] | undefined,
  hasPromptPack: boolean,
): number {
  if (status === 'succeeded') return 100
  if (status === 'queued' || status === 'preflight' || status === 'submitting') return 28
  if (
    status === 'submitted' ||
    status === 'polling' ||
    status === 'downloading' ||
    status === 'postprocessing'
  ) {
    return 72
  }
  if (hasPromptPack) return 100
  return 0
}

export function formatTimecode(durationMS: number): string {
  const totalSeconds = Math.round(durationMS / 1000)
  return `00:00:${totalSeconds.toString().padStart(2, '0')}:11`
}

export function subtitleForShot(code: string): string {
  const subtitles: Record<string, string> = {
    '01': '云海之上，天门将开。',
    '02': '我云澜，必登九霄！',
    '03': '白璃师姐？',
    '04': '天玄宗，外门广场。',
    '05': '今日试炼，开始！',
    '06': '这...这是什么？',
  }
  return subtitles[code] ?? '镜头对白待生成。'
}

function stepState(done: boolean, unlocked: boolean): FlowStepState {
  if (done) return 'done'
  return unlocked ? 'active' : 'waiting'
}
