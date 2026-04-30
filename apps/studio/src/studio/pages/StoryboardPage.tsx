import {
  BookOpenText,
  Boxes,
  ChevronDown,
  Layers3,
  Library,
  ListFilter,
  Plus,
  Search,
  Sparkles,
  Subtitles,
  Zap,
} from 'lucide-react'
import { useMemo, useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  useApproveApprovalGate,
  useGenerateShotPromptPack,
  useResubmitApprovalGate,
  useRequestApprovalChanges,
  useSaveEpisodeTimeline,
  useSaveShotPromptPack,
  useSeedApprovalGates,
  useSeedEpisodeAssets,
  useSeedEpisodeProduction,
  useSeedStoryboardShots,
  useSeedStoryMap,
  useShotPromptPack,
  useGenerationJobRecovery,
  useStartShotVideoGeneration,
  useStartStoryAnalysis,
  useStoryboardWorkspace,
  useUpdateStoryboardShot,
} from '../../api/hooks'
import type {
  ApprovalGate,
  Asset,
  Episode,
  GenerationJob,
  Project,
  ShotPromptPack,
} from '../../api/types'
import {
  agentFollowUpFeedbackLabel,
  buildStoryAnalysisFollowUpReturnState,
  type AgentOutputHandoffState,
} from '../agentOutput'
import { ActionButton } from '../components/ActionButton'
import { ProductionFlowPanel } from '../components/ProductionFlowPanel'
import { demoReferences } from '../mockData'
import { useStudioSelection } from '../hooks/useStudioSelection'
import { studioRoutePaths } from '../routes'
import { RecoveryPanel } from '../components/RecoveryPanel'
import { ReviewSummaryChips } from '../components/ReviewSummaryChips'
import { StatePlaceholder } from '../components/StatePlaceholder'
import type { InspectorTab, ShotDraft, StudioShot, ViewMode } from '../types'
import type {
  ApprovalGateOverview,
  StoryboardCharacterReference,
  StoryboardCharacterReferenceAsset,
  StoryboardCharacterReferenceSelection,
} from '../utils'
import {
  approvalGateStatusLabel,
  approvalGateStatusTone,
  approvalGateTypeLabel,
  buildStoryboardCharacterReferences,
  buildTimelineRequest,
  createLocalShot,
  filterShots,
  formatDuration,
  mapWorkspaceDisplayShots,
  productionHint,
  selectShotCharacterReferences,
  shotDraftValue,
  statusLabel,
  summarizeApprovalGates,
  thumbnailByIndex,
} from '../utils'

const emptyReferenceSelection: StoryboardCharacterReferenceSelection = {
  mode: 'empty',
  references: [],
}
const characterConsistencyBlockPattern = /\n*【角色一致性约束】[\s\S]*?【\/角色一致性约束】\n*/g
type ReferenceCoverageFilter = 'all' | 'covered' | 'needs_coverage'
type AssetsGraphHandoffState = {
  agentLabel?: string
  agentRole?: string
  fromAgentOutput?: boolean
  fromAssetsGraph?: boolean
  inspectorTab?: InspectorTab
  selectedNodeCode?: string
  selectedNodeKind?: 'character' | 'prop' | 'scene'
  selectedNodeName?: string
  selectedShotCode?: string
}

export function StoryboardPage() {
  const { activeEpisode, selectedProject } = useStudioSelection()
  const location = useLocation()
  const { data: storyboardWorkspace } = useStoryboardWorkspace(activeEpisode?.id)
  const analysesCount = storyboardWorkspace?.summary.analysis_count ?? 0
  const assets = storyboardWorkspace?.assets ?? []
  const canSeedStoryboard = analysesCount > 0 && (storyboardWorkspace?.summary.story_map_ready ?? false)
  const gates = storyboardWorkspace?.approval_gates ?? []
  const jobs = storyboardWorkspace?.generation_jobs ?? []
  const storyboardShots = storyboardWorkspace?.storyboard_shots ?? []
  const assetsGraphHandoff = location.state as (AssetsGraphHandoffState & AgentOutputHandoffState) | null
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>(() =>
    initialInspectorTabForHandoff(assetsGraphHandoff),
  )
  const [localShots, setLocalShots] = useState<StudioShot[]>([])
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [onlyNeedsWork, setOnlyNeedsWork] = useState(false)
  const [promptDrafts, setPromptDrafts] = useState<Record<string, string>>({})
  const [referenceCoverageFilter, setReferenceCoverageFilter] = useState<ReferenceCoverageFilter>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedShotKey, setSelectedShotKey] = useState<string>()
  const [shotDrafts, setShotDrafts] = useState<Record<string, ShotDraft>>({})
  const [viewMode, setViewMode] = useState<ViewMode>('grid')
  const savePromptPack = useSaveShotPromptPack()
  const updateStoryboardShot = useUpdateStoryboardShot()
  const approvalOverview = useMemo(() => summarizeApprovalGates(gates), [gates])
  const displayShots = useMemo(
    () => [...mapWorkspaceDisplayShots(storyboardShots), ...localShots],
    [localShots, storyboardShots],
  )
  const characterReferences = useMemo(
    () => buildStoryboardCharacterReferences(storyboardWorkspace?.story_map, assets),
    [assets, storyboardWorkspace?.story_map],
  )
  const shotReferenceSelections = useMemo(
    () =>
      new Map(
        displayShots.map((shot) => [shot.key, selectShotCharacterReferences(shot, characterReferences)]),
      ),
    [characterReferences, displayShots],
  )
  const shotCoverageCounts = useMemo(
    () =>
      displayShots.reduce(
        (counts, shot) => {
          const selection = shotReferenceSelections.get(shot.key) ?? emptyReferenceSelection
          if (hasPromptAutoConsistencyCoverage(selection)) {
            counts.covered += 1
          } else {
            counts.needsCoverage += 1
          }
          return counts
        },
        { covered: 0, needsCoverage: 0 },
      ),
    [displayShots, shotReferenceSelections],
  )
  const filteredShots = useMemo(() => {
    const searchedShots = filterShots(displayShots, searchQuery)
    const needsWorkFiltered = onlyNeedsWork
      ? searchedShots.filter((shot) => ['draft', 'generating', 'queued'].includes(shot.status))
      : searchedShots

    if (referenceCoverageFilter === 'all') return needsWorkFiltered

    return needsWorkFiltered.filter((shot) => {
      const selection = shotReferenceSelections.get(shot.key) ?? emptyReferenceSelection
      const hasCoverage = hasPromptAutoConsistencyCoverage(selection)
      return referenceCoverageFilter === 'covered' ? hasCoverage : !hasCoverage
    })
  }, [displayShots, onlyNeedsWork, referenceCoverageFilter, searchQuery, shotReferenceSelections])
  const handoffShot = assetsGraphHandoff?.selectedShotCode
    ? displayShots.find((shot) => shot.code === assetsGraphHandoff.selectedShotCode)
    : undefined
  const selectedShot = useMemo(
    () =>
      displayShots.find((shot) => shot.key === selectedShotKey) ??
      handoffShot ??
      filteredShots[0] ??
      displayShots[0],
    [displayShots, filteredShots, handoffShot, selectedShotKey],
  )
  const selectedShotReferenceSelection = useMemo(
    () =>
      (selectedShot ? shotReferenceSelections.get(selectedShot.key) : undefined) ?? emptyReferenceSelection,
    [selectedShot, shotReferenceSelections],
  )
  const autoConsistencyReferences = useMemo(
    () => promptAutoConsistencyReferences(selectedShotReferenceSelection),
    [selectedShotReferenceSelection],
  )

  const selectRelativeShot = (offset: number) => {
    const shotList = filteredShots.length > 0 ? filteredShots : displayShots
    if (shotList.length === 0 || !selectedShot) return
    const currentIndex = Math.max(
      0,
      shotList.findIndex((shot) => shot.key === selectedShot.key),
    )
    const nextIndex = (currentIndex + offset + shotList.length) % shotList.length
    setSelectedShotKey(shotList[nextIndex].key)
  }

  const addLocalShot = () => {
    const shot = createLocalShot(displayShots.length + 1)
    setLocalShots((shots) => [...shots, shot])
    setSelectedShotKey(shot.key)
    setInspectorTab('prompt')
  }

  const updateLocalShot = (shotKey: string, values: Partial<StudioShot>) => {
    setLocalShots((shots) =>
      shots.map((shot) => (shot.key === shotKey ? { ...shot, ...values } : shot)),
    )
  }

  const saveSelectedShot = () => {
    if (!selectedShot) return
    const draft = shotDraftValue(selectedShot, shotDrafts[selectedShot.key])
    const prompt = promptDrafts[selectedShot.key] ?? selectedShot.prompt
    if (!selectedShot.id) {
      updateLocalShot(selectedShot.key, {
        description: draft.description,
        durationMS: draft.durationMS,
        prompt,
        title: draft.title,
        sceneName: draft.title,
      })
      return
    }

    updateStoryboardShot.mutate({
      request: {
        description: draft.description,
        duration_ms: draft.durationMS,
        prompt,
        title: draft.title,
      },
      shotId: selectedShot.id,
    })
  }

  const saveSelectedPrompt = () => {
    if (!selectedShot) return
    const directPrompt = promptDrafts[selectedShot.key] ?? selectedShot.prompt
    const mergedPrompt = mergeCharacterConsistencyPrompt(directPrompt, autoConsistencyReferences)
    setPromptDrafts((current) => ({ ...current, [selectedShot.key]: mergedPrompt }))
    if (!selectedShot.id) {
      updateLocalShot(selectedShot.key, { prompt: mergedPrompt })
      return
    }
    savePromptPack.mutate({
      request: { direct_prompt: mergedPrompt },
      shotId: selectedShot.id,
    })
  }

  if (!selectedShot) {
    return (
      <section className="studio-page" aria-labelledby="storyboard-title">
        <h1 id="storyboard-title" className="sr-only">Storyboard</h1>
        <StatePlaceholder
          tone="empty"
          icon={Boxes}
          title="还没有镜头卡"
          description="先完成故事解析，再生成故事图谱与分镜卡。"
        />
      </section>
    )
  }

  return (
    <section className="studio-page storyboard-page" aria-labelledby="storyboard-title">
      <div className="board-header">
        <div>
          <h1 id="storyboard-title">Storyboard</h1>
          <span>镜头卡、提示词包和检查器已经拆成独立页面。</span>
        </div>
        <div className="board-actions">
          <label className="search-box storyboard-search">
            <Search aria-hidden="true" />
            <span className="search-box-label">检索</span>
            <input
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="搜索镜头、场景、角色编号..."
              value={searchQuery}
            />
          </label>
          <Link className="hero-secondary-action" to={studioRoutePaths.storyAnalysis}>
            <BookOpenText aria-hidden="true" />
            回到解析
          </Link>
          <Link className="hero-secondary-action" to={studioRoutePaths.timelineExport}>
            <Subtitles aria-hidden="true" />
            去时间线
          </Link>
        </div>
      </div>

      {assetsGraphHandoff?.fromAssetsGraph ? (
        <div className="board-notice storyboard-handoff-notice">
          已从 Assets / Graph 聚焦到 {storyboardHandoffLabel(assetsGraphHandoff)} ·{' '}
          {handoffShot
            ? `当前定位第 ${handoffShot.code} 镜`
            : assetsGraphHandoff.selectedShotCode
              ? `当前还没找到第 ${assetsGraphHandoff.selectedShotCode} 镜，请先生成或刷新分镜`
              : '当前未指定镜头'}
        </div>
      ) : assetsGraphHandoff?.fromAgentOutput ? (
        <>
          <div className="board-notice storyboard-handoff-notice">
            已从 {assetsGraphHandoff.agentLabel ?? 'Agent 输出'} 跳转到 Storyboard ·
            {` 当前默认打开 ${assetsGraphHandoff.inspectorTab === 'prompt' ? 'Prompt' : 'details'} 检查器`}
            {assetsGraphHandoff.followUpFeedback
              ? ` · 当前标记 ${agentFollowUpFeedbackLabel(assetsGraphHandoff.followUpFeedback)}`
              : ''}
            {assetsGraphHandoff.reviewContext?.storyboardPendingCount
              ? ` · Storyboard 侧剩余 ${assetsGraphHandoff.reviewContext.storyboardPendingCount} 个待跟进`
              : ''}
            {assetsGraphHandoff.reviewContext?.assetsGraphPendingCount
              ? ` · Assets / Graph 侧还有 ${assetsGraphHandoff.reviewContext.assetsGraphPendingCount} 个待跟进`
              : ''}
            {assetsGraphHandoff.reviewContext?.storyboardReturnedCount
              ? ` · Storyboard 已回传 ${assetsGraphHandoff.reviewContext.storyboardReturnedCount} 条`
              : ''}
            {assetsGraphHandoff.followUpFeedback === 'needs_follow_up'
              ? ' · 完成调整后建议回到 Story Analysis 收口反馈'
              : ''}
            <Link
              className="ghost-action"
              to={studioRoutePaths.storyAnalysis}
              state={buildStoryAnalysisFollowUpReturnState(assetsGraphHandoff, 'Storyboard')}
            >
              回到解析
            </Link>
            {assetsGraphHandoff.followUpFeedback === 'needs_follow_up' ? (
              <Link
                className="ghost-action"
                to={studioRoutePaths.storyAnalysis}
                state={buildStoryAnalysisFollowUpReturnState(
                  assetsGraphHandoff,
                  'Storyboard',
                  'adopted',
                  '已在 Storyboard 完成下游调整，可回到解析确认收口。',
                )}
              >
                处理完成并回到解析
              </Link>
            ) : null}
            {assetsGraphHandoff.reviewContext?.assetsGraphPendingCount ? (
              <Link className="ghost-action" to={studioRoutePaths.assetsGraph}>
                去 Assets / Graph 继续处理
              </Link>
            ) : null}
          </div>
          {assetsGraphHandoff.reviewContext ? (
            <ReviewSummaryChips
              currentSide="storyboard"
              storyboardPendingCount={assetsGraphHandoff.reviewContext.storyboardPendingCount}
              assetsGraphPendingCount={assetsGraphHandoff.reviewContext.assetsGraphPendingCount}
              totalReturnedCount={assetsGraphHandoff.reviewContext.totalReturnedCount}
              storyAnalysisLinkState={buildStoryAnalysisFollowUpReturnState(assetsGraphHandoff, 'Storyboard')}
            />
          ) : null}
        </>
      ) : null}

      <div className="storyboard-page-grid">
        <section className="storyboard-workspace storyboard-board" aria-labelledby="storyboard-board-title">
          <div className="board-header">
            <div>
              <h2 id="storyboard-board-title">分镜看板</h2>
              <span>{filteredShots.length} 个镜头</span>
            </div>
            <div className="board-actions">
              <StoryboardToolbar
                activeEpisode={activeEpisode}
                analysesCount={analysesCount}
                canSeedStoryboard={canSeedStoryboard}
                coveredShotsCount={shotCoverageCounts.covered}
                gatesCount={gates.length}
                needsCoverageShotsCount={shotCoverageCounts.needsCoverage}
                onlyNeedsWork={onlyNeedsWork}
                onCoverageFilterChange={setReferenceCoverageFilter}
                onToggleNeedsWork={() => setOnlyNeedsWork((value) => !value)}
                onViewModeChange={setViewMode}
                referenceCoverageFilter={referenceCoverageFilter}
                storyMapReady={storyboardWorkspace?.summary.story_map_ready ?? false}
                viewMode={viewMode}
              />
            </div>
          </div>

          {!selectedProject || !activeEpisode ? (
            <div className="board-notice">
              正在展示演示分镜。创建项目和剧集后，故事解析、资产图谱、生成队列和导出动作会接入当前剧集。
            </div>
          ) : null}

          <StoryboardFlowSection
              activeEpisode={activeEpisode}
              analysesCount={analysesCount}
              approvalOverview={approvalOverview}
              assets={assets}
              displayShots={displayShots}
              gates={gates}
            jobs={jobs}
            onlyNeedsWork={onlyNeedsWork}
            selectedProject={selectedProject}
            storyMapReady={storyboardWorkspace?.summary.story_map_ready ?? false}
          />

          <ShotGrid
            displayShots={filteredShots}
            onAddLocalShot={addLocalShot}
            onSelectShot={setSelectedShotKey}
            referenceSelections={shotReferenceSelections}
            selectedShotKey={selectedShot.key}
            viewMode={viewMode}
          />
        </section>

        <ShotInspector
          activeEpisode={activeEpisode}
          approvalOverview={approvalOverview}
          assets={assets}
          autoConsistencyReferences={autoConsistencyReferences}
          characterReferenceSelection={selectedShotReferenceSelection}
          displayShots={displayShots}
          gates={gates}
          inspectorTab={inspectorTab}
          note={notes[selectedShot.key] ?? ''}
          onInspectorTabChange={setInspectorTab}
          onNoteChange={(value) =>
            setNotes((current) => ({ ...current, [selectedShot.key]: value }))
          }
          onPromptDraftChange={(value) =>
            setPromptDrafts((current) => ({ ...current, [selectedShot.key]: value }))
          }
          onSavePrompt={saveSelectedPrompt}
          onSaveShot={saveSelectedShot}
          onSelectNext={() => selectRelativeShot(1)}
          onSelectPrevious={() => selectRelativeShot(-1)}
          onShotDraftChange={(draft) =>
            setShotDrafts((current) => ({ ...current, [selectedShot.key]: draft }))
          }
          project={selectedProject}
          promptDraft={promptDrafts[selectedShot.key]}
          savingPrompt={savePromptPack.isPending}
          savingShot={updateStoryboardShot.isPending}
          selectedShot={selectedShot}
          shotDraft={shotDrafts[selectedShot.key]}
        />
      </div>
    </section>
  )
}

function StoryboardToolbar({
  activeEpisode,
  analysesCount,
  canSeedStoryboard,
  coveredShotsCount,
  gatesCount,
  needsCoverageShotsCount,
  onlyNeedsWork,
  onCoverageFilterChange,
  onToggleNeedsWork,
  onViewModeChange,
  referenceCoverageFilter,
  storyMapReady,
  viewMode,
}: {
  activeEpisode?: Episode
  analysesCount: number
  canSeedStoryboard: boolean
  coveredShotsCount: number
  gatesCount: number
  needsCoverageShotsCount: number
  onlyNeedsWork: boolean
  onCoverageFilterChange: (filter: ReferenceCoverageFilter) => void
  onToggleNeedsWork: () => void
  onViewModeChange: (mode: ViewMode) => void
  referenceCoverageFilter: ReferenceCoverageFilter
  storyMapReady: boolean
  viewMode: ViewMode
}) {
  const startStoryAnalysis = useStartStoryAnalysis()
  const seedStoryMap = useSeedStoryMap()
  const seedAssets = useSeedEpisodeAssets()
  const seedProduction = useSeedEpisodeProduction()
  const seedStoryboard = useSeedStoryboardShots()

  return (
    <>
      <ActionButton
        disabled={!activeEpisode || startStoryAnalysis.isPending}
        icon={BookOpenText}
        label={`故事解析 ${analysesCount}`}
        onClick={() => activeEpisode && startStoryAnalysis.mutate(activeEpisode.id)}
      />
      <ActionButton
        disabled={!activeEpisode || analysesCount === 0 || seedStoryMap.isPending}
        disabledReason={
          analysesCount === 0 ? '先完成故事解析，worker 会自动写入角色、场景和道具种子。' : undefined
        }
        icon={Layers3}
        label={storyMapReady ? '资产图谱就绪' : '生成资产图谱'}
        onClick={() => activeEpisode && seedStoryMap.mutate(activeEpisode.id)}
      />
      <ActionButton
        disabled={!activeEpisode || analysesCount === 0 || seedProduction.isPending}
        disabledReason={
          analysesCount === 0
            ? '先完成故事解析，然后可一键生成资产图谱、候选资产、分镜卡和人审关卡。'
            : undefined
        }
        icon={Sparkles}
        label={seedProduction.isPending ? '生产中...' : '一键生产分镜包'}
        onClick={() => activeEpisode && seedProduction.mutate(activeEpisode.id)}
      />
      <ActionButton
        disabled={!activeEpisode || !storyMapReady || seedAssets.isPending}
        disabledReason={!storyMapReady ? '资产图谱生成后才能创建候选角色、场景和道具资产。' : undefined}
        icon={Library}
        label="生成候选资产"
        onClick={() => activeEpisode && seedAssets.mutate(activeEpisode.id)}
      />
      <ActionButton
        disabled={!activeEpisode || !canSeedStoryboard || seedStoryboard.isPending}
        disabledReason={!storyMapReady ? '需要故事解析和非空资产图谱后才能生成分镜卡。' : undefined}
        icon={Boxes}
        label="生成分镜卡"
        onClick={() => activeEpisode && seedStoryboard.mutate(activeEpisode.id)}
      />
      <ActionButton
        icon={ListFilter}
        label={onlyNeedsWork ? `待处理 ${gatesCount}` : `审批点 ${gatesCount}`}
        onClick={onToggleNeedsWork}
      />
      <div className="toolbar-toggle-stack">
        <span>引用图覆盖</span>
        <div className="view-toggle" aria-label="按引用图覆盖率筛选镜头">
          <button
            className={referenceCoverageFilter === 'all' ? 'active' : ''}
            onClick={() => onCoverageFilterChange('all')}
            type="button"
          >
            全部
          </button>
          <button
            className={referenceCoverageFilter === 'covered' ? 'active' : ''}
            onClick={() => onCoverageFilterChange('covered')}
            type="button"
          >
            已覆盖 {coveredShotsCount}
          </button>
          <button
            className={referenceCoverageFilter === 'needs_coverage' ? 'active' : ''}
            onClick={() => onCoverageFilterChange('needs_coverage')}
            type="button"
          >
            待补 {needsCoverageShotsCount}
          </button>
        </div>
      </div>
      <div className="view-toggle" aria-label="切换分镜视图">
        <button
          className={viewMode === 'grid' ? 'active' : ''}
          onClick={() => onViewModeChange('grid')}
          type="button"
        >
          宫格
        </button>
        <button
          className={viewMode === 'compact' ? 'active' : ''}
          onClick={() => onViewModeChange('compact')}
          type="button"
        >
          紧凑
        </button>
      </div>
    </>
  )
}

function StoryboardFlowSection({
  activeEpisode,
  analysesCount,
  approvalOverview,
  assets,
  displayShots,
  gates,
  jobs,
  onlyNeedsWork,
  selectedProject,
  storyMapReady,
}: {
  activeEpisode?: Episode
  analysesCount: number
  approvalOverview: ApprovalGateOverview
  assets: Asset[]
  displayShots: StudioShot[]
  gates: ApprovalGate[]
  jobs: GenerationJob[]
  onlyNeedsWork: boolean
  selectedProject?: Project
  storyMapReady: boolean
}) {
  const shotsCount = displayShots.filter((shot) => Boolean(shot.id)).length
  const nextHint = productionHint({
    activeEpisode,
    hasAnalysis: analysesCount > 0,
    storyMapReady,
  })
  const episodeJobs = useMemo(
    () => jobs.filter((job) => job.episode_id === activeEpisode?.id).slice(0, 5),
    [activeEpisode?.id, jobs],
  )

  return (
    <>
      <div className="dashboard-grid">
        <article className="surface-card">
          <span className="section-kicker">Project focus</span>
          <strong>{selectedProject?.name ?? '演示工作区'}</strong>
          <p>{nextHint}</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Needs work filter</span>
          <strong>{onlyNeedsWork ? '仅显示待处理镜头' : '显示全部镜头'}</strong>
          <p>把 generating / queued / draft 镜头筛出来，方便导演复看。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Production snapshot</span>
          <strong>{shotsCount} 个已接入分镜</strong>
          <p>{assets.length} 个候选资产 · {gates.length} 个审批点</p>
        </article>
        <article className="surface-card approval-surface-card">
          <span className="section-kicker">Approval radar</span>
          <strong>{approvalOverview.headline}</strong>
          <p>{approvalOverview.description}</p>
          <div className="approval-summary-row" aria-label="审批状态概览">
            <span>待确认 {approvalOverview.pending}</span>
            <span>待返修 {approvalOverview.changesRequested}</span>
            <span>已通过 {approvalOverview.approved}</span>
          </div>
        </article>
      </div>
      <ProductionFlowPanel
        analysesCount={analysesCount}
        assetsCount={assets.length}
        gatesCount={gates.length}
        jobs={episodeJobs}
        nextHint={nextHint}
        shotsCount={shotsCount}
        storyMapReady={storyMapReady}
      />
    </>
  )
}

function ShotGrid({
  displayShots,
  onAddLocalShot,
  onSelectShot,
  referenceSelections,
  selectedShotKey,
  viewMode,
}: {
  displayShots: StudioShot[]
  onAddLocalShot: () => void
  onSelectShot: (shotKey: string) => void
  referenceSelections: Map<string, StoryboardCharacterReferenceSelection>
  selectedShotKey: string
  viewMode: ViewMode
}) {
  return (
    <div className={viewMode === 'compact' ? 'shot-grid compact' : 'shot-grid'} aria-label="分镜镜头卡片">
      {displayShots.map((shot) => (
        <ShotCard
          key={shot.key}
          onSelect={() => onSelectShot(shot.key)}
          referenceSelection={
            referenceSelections.get(shot.key) ?? emptyReferenceSelection
          }
          selected={shot.key === selectedShotKey}
          shot={shot}
        />
      ))}
      <button className="add-shot-card" onClick={onAddLocalShot} type="button">
        <Plus aria-hidden="true" />
        <span>添加草稿镜头</span>
        <kbd>⌘ N</kbd>
      </button>
    </div>
  )
}

function ShotCard({
  onSelect,
  referenceSelection,
  selected,
  shot,
}: {
  onSelect: () => void
  referenceSelection: StoryboardCharacterReferenceSelection
  selected: boolean
  shot: StudioShot
}) {
  const generatePromptPack = useGenerateShotPromptPack()
  const canGeneratePack = Boolean(shot.id)

  return (
    <article className={selected ? 'shot-card selected' : 'shot-card'}>
      <button className="shot-select-button" onClick={onSelect} type="button">
        <span className="sr-only">选中第 {shot.code} 镜</span>
      </button>
      <div className={`shot-thumbnail ${shot.thumbnail}`} aria-hidden="true">
        <span>{shot.code}</span>
        <small>{formatDuration(shot.durationMS)}</small>
      </div>
      <div className="shot-card-body">
        <div>
          <span className="scene-chip">{shot.sceneCode}</span>
          <strong>{shot.sceneName}</strong>
        </div>
        <p>{shot.description}</p>
        <div className="tag-row">
          {shot.tags.map((tag) => (
            <span key={tag}>{tag}</span>
          ))}
        </div>
        <ShotReferenceSummary referenceSelection={referenceSelection} />
      </div>
      <div className="shot-status-row">
        <span className={`status-dot ${shot.status}`}>{statusLabel(shot.status)}</span>
        <small>{shot.progress}%</small>
      </div>
      <div className="progress-track">
        <span style={{ width: `${shot.progress}%` }} />
      </div>
      <button
        className="shot-inline-action"
        disabled={!canGeneratePack || generatePromptPack.isPending}
        onClick={(event) => {
          event.stopPropagation()
          if (shot.id) generatePromptPack.mutate(shot.id)
        }}
        type="button"
      >
        <Zap aria-hidden="true" />
        生成提示词包
      </button>
    </article>
  )
}

function ShotReferenceSummary({
  referenceSelection,
}: {
  referenceSelection: StoryboardCharacterReferenceSelection
}) {
  const previewCharacters = referenceSelection.references.slice(0, 2)
  const extraCount = Math.max(0, referenceSelection.references.length - previewCharacters.length)

  return (
    <div className="shot-reference-summary" aria-label="角色引用图摘要">
      <div className="shot-reference-summary-head">
        <span className={`shot-reference-pill ${referenceSelection.mode}`}>
          {shotReferenceHeadline(referenceSelection)}
        </span>
        <small>{shotReferenceSubline(referenceSelection)}</small>
      </div>
      {previewCharacters.length > 0 ? (
        <div className="shot-reference-tags">
          {previewCharacters.map((reference) => (
            <span key={reference.characterId}>
              {reference.name} · {reference.referenceAssets.length} 角度
            </span>
          ))}
          {extraCount > 0 ? <span>+{extraCount} 个角色</span> : null}
        </div>
      ) : null}
    </div>
  )
}

function ShotInspector({
  activeEpisode,
  approvalOverview,
  assets,
  autoConsistencyReferences,
  characterReferenceSelection,
  displayShots,
  gates,
  inspectorTab,
  note,
  onInspectorTabChange,
  onNoteChange,
  onPromptDraftChange,
  onSavePrompt,
  onSaveShot,
  onSelectNext,
  onSelectPrevious,
  onShotDraftChange,
  project,
  promptDraft,
  savingPrompt,
  savingShot,
  selectedShot,
  shotDraft,
}: {
  activeEpisode?: Episode
  approvalOverview: ApprovalGateOverview
  assets: Asset[]
  autoConsistencyReferences: StoryboardCharacterReference[]
  characterReferenceSelection: StoryboardCharacterReferenceSelection
  displayShots: StudioShot[]
  gates: ApprovalGate[]
  inspectorTab: InspectorTab
  note: string
  onInspectorTabChange: (tab: InspectorTab) => void
  onNoteChange: (value: string) => void
  onPromptDraftChange: (value: string) => void
  onSavePrompt: () => void
  onSaveShot: () => void
  onSelectNext: () => void
  onSelectPrevious: () => void
  onShotDraftChange: (draft: ShotDraft) => void
  project?: Project
  promptDraft?: string
  savingPrompt: boolean
  savingShot: boolean
  selectedShot: StudioShot
  shotDraft?: ShotDraft
}) {
  const { data: promptPack } = useShotPromptPack(selectedShot.id)
  const approveGate = useApproveApprovalGate()
  const requestChangesGate = useRequestApprovalChanges()
  const resubmitGate = useResubmitApprovalGate()

  return (
    <aside className="shot-inspector" aria-label="当前镜头检查器">
      <InspectorHeader
        onSelectNext={onSelectNext}
        onSelectPrevious={onSelectPrevious}
        selectedShot={selectedShot}
      />
      <InspectorTabs activeTab={inspectorTab} onChange={onInspectorTabChange} />
      {inspectorTab === 'details' ? (
        <SceneTimingCard
          draft={shotDraftValue(selectedShot, shotDraft)}
          onDraftChange={onShotDraftChange}
          onSave={onSaveShot}
          saving={savingShot}
          selectedShot={selectedShot}
        />
      ) : null}
      {inspectorTab === 'prompt' ? (
        <PromptPackCard
          autoConsistencyReferences={autoConsistencyReferences}
          characterReferenceSelection={characterReferenceSelection}
          onPromptDraftChange={onPromptDraftChange}
          onSavePrompt={onSavePrompt}
          pack={promptPack}
          promptDraft={promptDraft}
          savingPrompt={savingPrompt}
          selectedShot={selectedShot}
        />
      ) : null}
      {inspectorTab === 'references' ? (
        <ReferenceTokens
          assets={assets}
          characterReferenceSelection={characterReferenceSelection}
          pack={promptPack}
        />
      ) : null}
      {inspectorTab === 'notes' ? (
        <ShotNotes note={note} onNoteChange={onNoteChange} selectedShot={selectedShot} />
      ) : null}
      <ModelPresetCard />
      <ApprovalStatusCard
        activeEpisode={activeEpisode}
        approvalOverview={approvalOverview}
        approveGate={approveGate}
        gates={gates}
        resubmitGate={resubmitGate}
        requestChangesGate={requestChangesGate}
      />
      <GenerationRecoveryCard jobId={selectedShot.latestGenerationJobId} />
      <InspectorActions
        activeEpisode={activeEpisode}
        disabled={!project || !selectedShot.id}
        displayShots={displayShots}
        selectedShot={selectedShot}
      />
    </aside>
  )
}

function InspectorHeader({
  onSelectNext,
  onSelectPrevious,
  selectedShot,
}: {
  onSelectNext: () => void
  onSelectPrevious: () => void
  selectedShot: StudioShot
}) {
  return (
    <header className="inspector-header">
      <div>
        <strong>第 {selectedShot.code} 镜</strong>
        <span>已选中</span>
      </div>
      <div className="inspector-nav">
        <button aria-label="上一镜" onClick={onSelectPrevious} type="button">
          ‹
        </button>
        <button aria-label="下一镜" onClick={onSelectNext} type="button">
          ›
        </button>
      </div>
    </header>
  )
}

function InspectorTabs({
  activeTab,
  onChange,
}: {
  activeTab: InspectorTab
  onChange: (tab: InspectorTab) => void
}) {
  const tabs: Array<{ key: InspectorTab; label: string }> = [
    { key: 'details', label: '镜头信息' },
    { key: 'prompt', label: '提示词' },
    { key: 'references', label: '参考资产' },
    { key: 'notes', label: '导演备注' },
  ]

  return (
    <nav className="inspector-tabs" aria-label="镜头详情分区">
      {tabs.map((tab) => (
        <button
          className={activeTab === tab.key ? 'active' : ''}
          key={tab.key}
          onClick={() => onChange(tab.key)}
          type="button"
        >
          {tab.label}
        </button>
      ))}
    </nav>
  )
}

function SceneTimingCard({
  draft,
  onDraftChange,
  onSave,
  saving,
  selectedShot,
}: {
  draft: ShotDraft
  onDraftChange: (draft: ShotDraft) => void
  onSave: () => void
  saving: boolean
  selectedShot: StudioShot
}) {
  return (
    <section className="inspector-section" aria-labelledby="scene-timing-title">
      <div className="section-title-row">
        <h2 id="scene-timing-title">场景与节奏</h2>
        <button disabled={saving} onClick={onSave} type="button">
          保存镜头
        </button>
      </div>
      <label className="field-editor">
        <span>镜头标题</span>
        <input
          onChange={(event) => onDraftChange({ ...draft, title: event.target.value })}
          value={draft.title}
        />
      </label>
      <label className="field-editor">
        <span>镜头说明</span>
        <textarea
          onChange={(event) =>
            onDraftChange({ ...draft, description: event.target.value })
          }
          value={draft.description}
        />
      </label>
      <label className="field-editor">
        <span>时长 ms</span>
        <input
          min={1}
          onChange={(event) =>
            onDraftChange({ ...draft, durationMS: Number(event.target.value) })
          }
          type="number"
          value={draft.durationMS}
        />
      </label>
      <dl className="detail-list">
        <div>
          <dt>场景</dt>
          <dd>
            <span className="scene-chip">{selectedShot.sceneCode}</span> {selectedShot.sceneName}
          </dd>
        </div>
        <div>
          <dt>镜头时长</dt>
          <dd>{formatDuration(selectedShot.durationMS)}</dd>
        </div>
        <div>
          <dt>帧率</dt>
          <dd>24 fps</dd>
        </div>
        <div>
          <dt>画幅</dt>
          <dd>16:9 宽银幕</dd>
        </div>
      </dl>
    </section>
  )
}

function PromptPackCard({
  autoConsistencyReferences,
  characterReferenceSelection,
  onPromptDraftChange,
  onSavePrompt,
  pack,
  promptDraft,
  savingPrompt,
  selectedShot,
}: {
  autoConsistencyReferences: StoryboardCharacterReference[]
  characterReferenceSelection: StoryboardCharacterReferenceSelection
  onPromptDraftChange: (value: string) => void
  onSavePrompt: () => void
  pack?: ShotPromptPack
  promptDraft?: string
  savingPrompt: boolean
  selectedShot: StudioShot
}) {
  const generatePromptPack = useGenerateShotPromptPack()
  const promptEditorRef = useRef<HTMLTextAreaElement>(null)
  const promptText = promptDraft ?? pack?.direct_prompt ?? selectedShot.prompt
  const hasUnsavedPrompt = promptText !== (pack?.direct_prompt ?? selectedShot.prompt)
  const promptContextChips = useMemo(
    () => [
      { label: '场景', value: `场景：${selectedShot.sceneName}` },
      { label: '镜头说明', value: `镜头说明：${selectedShot.description}` },
      { label: '镜头时长', value: `镜头时长：${formatDuration(selectedShot.durationMS)}` },
      { label: '角色一致性', value: '保持角色造型、服装与面部特征一致。' },
      { label: '镜头运动', value: '镜头运动平稳，主体动作清晰，避免突兀跳切。' },
    ],
    [selectedShot.description, selectedShot.durationMS, selectedShot.sceneName],
  )
  const autoConsistencyPreview = useMemo(
    () => buildCharacterConsistencyPreview(autoConsistencyReferences),
    [autoConsistencyReferences],
  )
  const characterReferences = characterReferenceSelection.references
  const referenceChips = pack?.reference_bindings ?? []

  const insertSnippet = (snippet: string) => {
    const textarea = promptEditorRef.current
    if (!textarea) {
      onPromptDraftChange(joinPromptSnippet(promptText, snippet))
      return
    }

    const { selectionEnd, selectionStart } = textarea
    const nextPrompt = insertPromptSnippet(promptText, snippet, selectionStart, selectionEnd)
    onPromptDraftChange(nextPrompt)

    requestAnimationFrame(() => {
      const cursor = selectionStart + promptInsertionOffset(promptText, snippet, selectionStart, selectionEnd)
      textarea.focus()
      textarea.setSelectionRange(cursor, cursor)
    })
  }

  return (
    <section className="inspector-section prompt-section" aria-labelledby="prompt-pack-title">
      <div className="section-title-row">
        <h2 id="prompt-pack-title">SD2 镜头提示词包</h2>
        <div className="section-actions">
          <button disabled={savingPrompt} onClick={onSavePrompt} type="button">
            保存提示词
          </button>
          <button
            disabled={!selectedShot.id || generatePromptPack.isPending}
            onClick={() => selectedShot.id && generatePromptPack.mutate(selectedShot.id)}
            type="button"
          >
            重新生成
          </button>
        </div>
      </div>
      <div className="prompt-shortcut-section">
        <span>参考令牌快捷插入</span>
        {referenceChips.length > 0 ? (
          <div className="prompt-chip-row">
            {referenceChips.map((binding) => (
              <button
                className="prompt-chip-button"
                key={binding.token}
                onClick={() => insertSnippet(binding.token)}
                type="button"
              >
                {binding.token} · {referenceRoleLabel(binding.role)}
              </button>
            ))}
          </div>
        ) : (
          <small className="prompt-token-empty">先生成提示词包，系统会在这里提供可插入的参考令牌。</small>
        )}
      </div>
      <CharacterReferenceLibrary
        allowInsert
        matchMode={characterReferenceSelection.mode}
        onInsertReference={insertSnippet}
        references={characterReferences}
      />
      <PromptAutoConsistencyCard
        autoConsistencyReferences={autoConsistencyReferences}
        fallbackReferences={characterReferenceSelection.references}
        fallbackMode={characterReferenceSelection.mode}
        previewLines={autoConsistencyPreview}
      />
      <div className="prompt-shortcut-section">
        <span>镜头语义快捷插入</span>
        <div className="prompt-chip-row">
          {promptContextChips.map((chip) => (
            <button
              className="prompt-chip-button prompt-chip-button-secondary"
              key={chip.label}
              onClick={() => insertSnippet(chip.value)}
              type="button"
            >
              {chip.label}
            </button>
          ))}
        </div>
      </div>
      <label className="prompt-editor">
        <span>可编辑提示词</span>
        <textarea
          ref={promptEditorRef}
          onChange={(event) => onPromptDraftChange(event.target.value)}
          value={promptText}
        />
      </label>
      <div className="prompt-editor-footer">
        <small>
          {pack
            ? `${pack.provider} · ${pack.preset} · ${promptTaskTypeLabel(pack.task_type)}`
            : '尚未生成 prompt pack'}
        </small>
        <small>
          {hasUnsavedPrompt ? '未保存' : '已同步'} · {promptText.length}/2000
        </small>
      </div>
    </section>
  )
}

function PromptAutoConsistencyCard({
  autoConsistencyReferences,
  fallbackMode,
  fallbackReferences,
  previewLines,
}: {
  autoConsistencyReferences: StoryboardCharacterReference[]
  fallbackMode: StoryboardCharacterReferenceSelection['mode']
  fallbackReferences: StoryboardCharacterReference[]
  previewLines: string[]
}) {
  const [previewOpen, setPreviewOpen] = useState(false)

  return (
    <div className="prompt-shortcut-section">
      <span>保存增强</span>
      {autoConsistencyReferences.length > 0 ? (
        <div className="prompt-auto-consistency-card">
          <div className="prompt-auto-consistency-head">
            <div>
              <strong>保存提示词时，会自动写入角色一致性约束。</strong>
              <small>
                {autoConsistencyReferences.map((reference) => reference.name).join(' / ')} 的锚点、服装和优先视角会被整理进
                `direct_prompt`，重复保存时会自动替换旧的约束块。
              </small>
            </div>
            <button
              aria-expanded={previewOpen}
              className="prompt-auto-consistency-toggle"
              onClick={() => setPreviewOpen((value) => !value)}
              type="button"
            >
              {previewOpen ? '收起一致性块' : '预览一致性块'}
            </button>
          </div>
          {previewOpen ? (
            <div className="prompt-auto-consistency-preview">
              {previewLines.map((line) => (
                <span key={line}>{line}</span>
              ))}
            </div>
          ) : null}
        </div>
      ) : fallbackMode === 'fallback' && fallbackReferences.length > 0 ? (
        <small className="prompt-token-empty">
          当前镜头只命中了“全剧集回退参考”；为避免把无关角色写进提示词，保存时不会自动注入角色一致性约束。先在正文写入角色名，或点上方“插入角度描述”即可激活自动增强。
        </small>
      ) : (
        <small className="prompt-token-empty">当前没有可自动注入的角色一致性约束。</small>
      )}
    </div>
  )
}

function CharacterReferenceLibrary({
  allowInsert = false,
  matchMode,
  onInsertReference,
  references,
}: {
  allowInsert?: boolean
  matchMode: StoryboardCharacterReferenceSelection['mode']
  onInsertReference?: (snippet: string) => void
  references: StoryboardCharacterReference[]
}) {
  return (
    <div className="prompt-shortcut-section">
      <span>Character Bible 引用图</span>
      {references.length === 0 ? (
        <small className="prompt-token-empty">
          先在 Assets / Graph 为角色保存引用图，Storyboard 才会出现跨页复用的角色参考。
        </small>
      ) : (
        <>
          <small className="prompt-reference-helper">{characterReferenceHelper(matchMode)}</small>
          <div className="storyboard-character-reference-grid">
            {references.map((reference) => (
              <article className="storyboard-character-reference-card" key={reference.characterId}>
                <div className="storyboard-character-reference-head">
                  <div>
                    <span className="scene-chip">{reference.code}</span>
                    <strong>{reference.name}</strong>
                  </div>
                  <span className="asset-status-chip ready">{reference.referenceAssets.length} 张引用图</span>
                </div>
                <p className="storyboard-character-reference-anchor">{reference.anchor}</p>
                <div className="storyboard-character-reference-meta">
                  <span>{reference.wardrobe || '服装待补充'}</span>
                  <span>{reference.referenceAssets.map((asset) => asset.angle).join(' / ')}</span>
                </div>
                <div className="storyboard-character-reference-assets">
                  {reference.referenceAssets.map((asset) => (
                    <div className="storyboard-character-reference-asset" key={`${reference.characterId}-${asset.angle}`}>
                      <div>
                        <strong>{asset.angle}</strong>
                        <small>{asset.assetLabel}</small>
                      </div>
                      {allowInsert && onInsertReference ? (
                        <button
                          className="prompt-chip-button prompt-chip-button-secondary"
                          onClick={() => onInsertReference(buildCharacterReferenceSnippet(reference, asset))}
                          type="button"
                        >
                          插入角度描述
                        </button>
                      ) : (
                        <span className="storyboard-character-reference-badge">已持久化</span>
                      )}
                    </div>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function shotReferenceHeadline(selection: StoryboardCharacterReferenceSelection) {
  if (selection.mode === 'matched') {
    return `命中 ${selection.references.length} 个角色参考`
  }
  if (selection.mode === 'fallback') {
    return `可复用 ${selection.references.length} 个角色参考`
  }
  return '暂无角色引用图'
}

function shotReferenceSubline(selection: StoryboardCharacterReferenceSelection) {
  if (selection.mode === 'matched') {
    return '右侧 Prompt 面板会优先展示这些角色的引用图。'
  }
  if (selection.mode === 'fallback') {
    return '当前镜头未命中角色名，右侧会回退展示本集全部已持久化引用图。'
  }
  return '先去 Assets / Graph 保存 Character Bible 引用图。'
}

function storyboardHandoffLabel(state: AssetsGraphHandoffState): string {
  const kindLabels = {
    character: '角色节点',
    prop: '道具节点',
    scene: '场景节点',
  } as const

  const kindLabel = state.selectedNodeKind ? kindLabels[state.selectedNodeKind] : '图谱节点'
  const nodeLabel = state.selectedNodeName ?? state.selectedNodeCode ?? '未命名节点'
  return `${kindLabel} ${nodeLabel}`
}

function initialInspectorTabForHandoff(state: AssetsGraphHandoffState | null): InspectorTab {
  if (state?.fromAgentOutput) {
    return state.inspectorTab ?? 'details'
  }
  if (!state?.fromAssetsGraph) {
    return 'details'
  }

  return state.selectedNodeKind === 'character' ? 'prompt' : 'details'
}

function referenceRoleLabel(role: ShotPromptPack['reference_bindings'][number]['role']) {
  const labels: Record<ShotPromptPack['reference_bindings'][number]['role'], string> = {
    first_frame: '首帧',
    last_frame: '尾帧',
    reference_image: '参考图',
  }
  return labels[role]
}

function promptTaskTypeLabel(taskType: ShotPromptPack['task_type']) {
  const labels: Record<ShotPromptPack['task_type'], string> = {
    first_last_frame_to_video: '首尾帧视频',
    image_to_video: '图生视频',
    text_to_video: '文生视频',
  }
  return labels[taskType]
}

function joinPromptSnippet(promptText: string, snippet: string) {
  if (!promptText.trim()) return snippet
  return `${promptText.trimEnd()}\n${snippet}`
}

function insertPromptSnippet(
  promptText: string,
  snippet: string,
  selectionStart: number,
  selectionEnd: number,
) {
  const before = promptText.slice(0, selectionStart)
  const after = promptText.slice(selectionEnd)
  const prefix = before && !before.endsWith('\n') ? '\n' : ''
  const suffix = after && !after.startsWith('\n') ? '\n' : ''
  return `${before}${prefix}${snippet}${suffix}${after}`
}

function promptInsertionOffset(
  promptText: string,
  snippet: string,
  selectionStart: number,
  selectionEnd: number,
) {
  const before = promptText.slice(0, selectionStart)
  const after = promptText.slice(selectionEnd)
  const prefixLength = before && !before.endsWith('\n') ? 1 : 0
  const suffixLength = after && !after.startsWith('\n') ? 1 : 0
  return prefixLength + snippet.length + suffixLength
}

function ReferenceTokens({
  assets,
  characterReferenceSelection,
  pack,
}: {
  assets: Asset[]
  characterReferenceSelection: StoryboardCharacterReferenceSelection
  pack?: ShotPromptPack
}) {
  const refs =
    pack?.reference_bindings.map((ref, index) => ({
      label: ref.token,
      thumbnail: thumbnailByIndex(index),
    })) ?? demoReferences

  return (
    <section className="inspector-section" aria-labelledby="reference-token-title">
      <div className="section-title-row">
        <h2 id="reference-token-title">参考资产令牌</h2>
        <span>
          {refs.length} / {Math.max(10, assets.length)}
        </span>
      </div>
      <div className="reference-grid">
        {refs.slice(0, 4).map((ref) => (
          <div className="reference-card" key={ref.label}>
            <span className={`reference-thumb ${ref.thumbnail}`} aria-hidden="true" />
            <strong>{ref.label}</strong>
          </div>
        ))}
      </div>
      <CharacterReferenceLibrary
        matchMode={characterReferenceSelection.mode}
        references={characterReferenceSelection.references}
      />
    </section>
  )
}

function characterReferenceHelper(mode: StoryboardCharacterReferenceSelection['mode']) {
  if (mode === 'matched') {
    return '已根据镜头标题、说明、提示词与标签自动匹配到相关角色引用图。'
  }
  if (mode === 'fallback') {
    return '当前镜头还没命中明确角色名，先展示本集全部已持久化的角色引用图。'
  }
  return '当前没有可用的角色引用图。'
}

function buildCharacterReferenceSnippet(
  reference: StoryboardCharacterReference,
  asset: StoryboardCharacterReferenceAsset,
) {
  const parts = [`角色 ${reference.name} 保持 ${reference.anchor}`]

  if (reference.wardrobe) {
    parts.push(`服装维持 ${reference.wardrobe}`)
  }

  parts.push(`镜头优先采用 ${asset.angle} 视角参考`)

  if (reference.notes) {
    parts.push(`补充约束：${reference.notes}`)
  }

  return `${parts.join('，')}。`
}

function promptAutoConsistencyReferences(selection: StoryboardCharacterReferenceSelection) {
  if (selection.mode === 'matched') {
    return selection.references
  }

  if (selection.mode === 'fallback' && selection.references.length === 1) {
    return selection.references
  }

  return []
}

function hasPromptAutoConsistencyCoverage(selection: StoryboardCharacterReferenceSelection) {
  return promptAutoConsistencyReferences(selection).length > 0
}

function buildCharacterConsistencyPreview(references: StoryboardCharacterReference[]) {
  return references.map((reference) => {
    const parts = [`${reference.name}：保持 ${reference.anchor}`]

    if (reference.wardrobe) {
      parts.push(`服装 ${reference.wardrobe}`)
    }

    const angles = reference.referenceAssets.map((asset) => asset.angle).join(' / ')
    if (angles) {
      parts.push(`优先 ${angles} 视角`)
    }

    if (reference.notes) {
      parts.push(`补充 ${reference.notes}`)
    }

    return parts.join('；')
  })
}

function buildCharacterConsistencyBlock(references: StoryboardCharacterReference[]) {
  const lines = buildCharacterConsistencyPreview(references)
  if (lines.length === 0) return ''

  return ['【角色一致性约束】', ...lines.map((line) => `- ${line}`), '【/角色一致性约束】'].join('\n')
}

function mergeCharacterConsistencyPrompt(
  promptText: string,
  references: StoryboardCharacterReference[],
) {
  const sanitizedPrompt = promptText.replace(characterConsistencyBlockPattern, '').trim()
  const block = buildCharacterConsistencyBlock(references)

  if (!block) return sanitizedPrompt
  if (!sanitizedPrompt) return block

  return `${sanitizedPrompt}\n\n${block}`
}

function ModelPresetCard() {
  return (
    <section className="inspector-section" aria-labelledby="model-preset-title">
      <h2 id="model-preset-title">生成模型预设</h2>
      <button className="model-select" type="button">
        <span>
          <strong>Seedance Fast</strong>
          <small>优先保证出片速度与动作连续性</small>
        </span>
        <span className="model-badge">SD2.1</span>
        <ChevronDown aria-hidden="true" />
      </button>
    </section>
  )
}

function ApprovalStatusCard({
  activeEpisode,
  approvalOverview,
  approveGate,
  gates,
  resubmitGate,
  requestChangesGate,
}: {
  activeEpisode?: Episode
  approvalOverview: ApprovalGateOverview
  approveGate: ReturnType<typeof useApproveApprovalGate>
  gates: ApprovalGate[]
  resubmitGate: ReturnType<typeof useResubmitApprovalGate>
  requestChangesGate: ReturnType<typeof useRequestApprovalChanges>
}) {
  const seedApprovalGates = useSeedApprovalGates()
  const orderedGates = useMemo(
    () =>
      [...gates].sort(
        (left, right) =>
          approvalGatePriority(left.status) - approvalGatePriority(right.status) ||
          left.gate_type.localeCompare(right.gate_type),
      ),
    [gates],
  )

  return (
    <section className="inspector-section approval-status" aria-labelledby="approval-status-title">
      <div className="approval-status-header">
        <div>
          <h2 id="approval-status-title">人审状态</h2>
          <small>{approvalOverview.description}</small>
        </div>
        <span className={`approval-pill ${approvalOverview.tone}`}>{approvalOverview.headline}</span>
      </div>
      <div className="approval-stat-grid" aria-label="审批统计">
        <article className="approval-stat">
          <strong>{approvalOverview.pending}</strong>
          <span>待确认</span>
        </article>
        <article className="approval-stat">
          <strong>{approvalOverview.changesRequested}</strong>
          <span>待返修</span>
        </article>
        <article className="approval-stat">
          <strong>{approvalOverview.approved}</strong>
          <span>已通过</span>
        </article>
      </div>
      {orderedGates.length === 0 ? (
        <StatePlaceholder
          tone="empty"
          title="还没有审批点"
          description="先生成分镜和审批点，导演确认流才会出现。"
        />
      ) : (
        <div className="approval-gate-list">
          {orderedGates.map((gate) => {
            const tone = approvalGateStatusTone(gate.status)
            const isChangesRequested = gate.status === 'changes_requested'
            const isPending = gate.status === 'pending'

            return (
              <article className={`approval-gate-card ${tone}`} key={gate.id}>
                <div className="approval-gate-head">
                  <div>
                    <strong>{approvalGateTypeLabel(gate.gate_type)}</strong>
                    <span>
                      {gate.subject_type.replaceAll('_', ' ')} · {gate.subject_id.slice(0, 8)}
                    </span>
                  </div>
                  <span className={`approval-pill ${tone}`}>{approvalGateStatusLabel(gate.status)}</span>
                </div>
                <p>{approvalGateNarrative(gate)}</p>
                {isPending ? (
                  <div className="approval-actions">
                    <button
                      disabled={requestChangesGate.isPending}
                      onClick={() =>
                        requestChangesGate.mutate({
                          episodeId: gate.episode_id,
                          gateId: gate.id,
                        })
                      }
                      type="button"
                    >
                      {requestChangesGate.isPending ? '提交中...' : '请求修改'}
                    </button>
                    <button
                      disabled={approveGate.isPending}
                      onClick={() =>
                        approveGate.mutate({
                          episodeId: gate.episode_id,
                          gateId: gate.id,
                        })
                      }
                      type="button"
                    >
                      {approveGate.isPending ? '确认中...' : '确认通过'}
                    </button>
                  </div>
                ) : isChangesRequested ? (
                  <div className="approval-actions">
                    <button
                      disabled={resubmitGate.isPending}
                      onClick={() =>
                        resubmitGate.mutate({
                          episodeId: gate.episode_id,
                          gateId: gate.id,
                        })
                      }
                      type="button"
                    >
                      {resubmitGate.isPending ? '送审中...' : '重新送审'}
                    </button>
                  </div>
                ) : (
                  <div className="approval-gate-meta">{approvalGateMeta(gate)}</div>
                )}
              </article>
            )
          })}
        </div>
      )}
      <div className="approval-actions approval-actions-footer">
        <button
          disabled={!activeEpisode || seedApprovalGates.isPending}
          onClick={() => activeEpisode && seedApprovalGates.mutate(activeEpisode.id)}
          type="button"
        >
          刷新审批点
        </button>
      </div>
    </section>
  )
}

function approvalGatePriority(status: ApprovalGate['status']) {
  const priority: Record<ApprovalGate['status'], number> = {
    pending: 0,
    changes_requested: 1,
    rejected: 2,
    approved: 3,
    canceled: 4,
  }
  return priority[status]
}

function approvalGateNarrative(gate: ApprovalGate): string {
  if (gate.status === 'changes_requested') {
    return gate.review_note || '导演已打回，等待上游素材或分镜修改后重新送审。'
  }
  if (gate.status === 'approved') {
    return gate.review_note || '当前审批点已通过，可以继续推进下一步生成。'
  }
  if (gate.status === 'rejected') {
    return gate.review_note || '当前审批点已拒绝，需要重新评估对应内容。'
  }
  if (gate.status === 'canceled') {
    return '当前审批点已取消，仅保留历史记录。'
  }
  return '当前审批点待导演确认；通过后才能更安心地进入高成本生成。'
}

function approvalGateMeta(gate: ApprovalGate): string {
  if (!gate.reviewed_by) return '暂无审核人记录'
  const reviewedAt =
    gate.reviewed_at && !gate.reviewed_at.startsWith('0001-01-01')
      ? new Date(gate.reviewed_at).toLocaleString('zh-CN', {
          hour: '2-digit',
          minute: '2-digit',
          month: '2-digit',
          day: '2-digit',
        })
      : '时间待补充'
  return `${gate.reviewed_by} · ${reviewedAt}`
}

function ShotNotes({
  note,
  onNoteChange,
  selectedShot,
}: {
  note: string
  onNoteChange: (value: string) => void
  selectedShot: StudioShot
}) {
  return (
    <section className="inspector-section prompt-section" aria-labelledby="shot-notes-title">
      <h2 id="shot-notes-title">导演备注</h2>
      <label className="prompt-editor">
        <span>给后续生成和剪辑的备注</span>
        <textarea
          onChange={(event) => onNoteChange(event.target.value)}
          placeholder={`例如：第 ${selectedShot.code} 镜需要保留云海开阔感，人物不要贴脸。`}
          value={note}
        />
      </label>
    </section>
  )
}

function GenerationRecoveryCard({ jobId }: { jobId?: string }) {
  const { data, isLoading, isError } = useGenerationJobRecovery(jobId)
  if (!jobId) return null
  return (
    <RecoveryPanel
      title="生成任务恢复"
      subtitle="最近一次生成的轨迹"
      isLoading={isLoading}
      isError={isError}
      status={data?.summary.current_status}
      isTerminal={data?.summary.is_terminal}
      isRecoverable={data?.summary.is_recoverable}
      statusEnteredAt={data?.summary.status_entered_at}
      lastEventAt={data?.summary.last_event_at}
      totalEventCount={data?.summary.total_event_count}
      sameStatusCount={data?.summary.status_event_count}
      nextHint={data?.summary.next_hint}
      events={data?.events.map((event) => ({
        status: event.status,
        message: event.message,
        created_at: event.created_at,
      }))}
    />
  )
}



function InspectorActions({
  activeEpisode,
  disabled,
  displayShots,
  selectedShot,
}: {
  activeEpisode?: Episode
  disabled: boolean
  displayShots: StudioShot[]
  selectedShot: StudioShot
}) {
  const generatePromptPack = useGenerateShotPromptPack()
  const navigate = useNavigate()
  const saveTimeline = useSaveEpisodeTimeline()
  const startVideoGeneration = useStartShotVideoGeneration()
  const persistedShots = useMemo(
    () => displayShots.filter((shot) => Boolean(shot.id)),
    [displayShots],
  )

  const regenerate = () => {
    if (!selectedShot.id) return
    generatePromptPack.mutate(selectedShot.id, {
      onSuccess: () => startVideoGeneration.mutate(selectedShot.id ?? ''),
    })
  }

  return (
    <div className="inspector-actions">
      <button
        className="primary-inspector-action"
        disabled={disabled || startVideoGeneration.isPending || generatePromptPack.isPending}
        onClick={regenerate}
        type="button"
      >
        <Sparkles aria-hidden="true" />
        重新出片
      </button>
      <button
        disabled={disabled || generatePromptPack.isPending}
        onClick={() => selectedShot.id && generatePromptPack.mutate(selectedShot.id)}
        type="button"
      >
        重写提示词
      </button>
      <button
        disabled={disabled || !activeEpisode || saveTimeline.isPending}
        onClick={() =>
          activeEpisode &&
          saveTimeline.mutate({
            episodeId: activeEpisode.id,
            request: buildTimelineRequest(persistedShots),
          }, {
            onSuccess: () =>
              navigate(studioRoutePaths.timelineExport, {
                state: {
                  fromStoryboard: true,
                  selectedShotCode: selectedShot.code,
                  shotsCount: persistedShots.length,
                },
              }),
          })
        }
        type="button"
      >
        {saveTimeline.isPending ? '送入中...' : '送入 Timeline'}
      </button>
    </div>
  )
}
