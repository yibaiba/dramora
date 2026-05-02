import {
  Activity,
  BookOpenText,
  Boxes,
  Clapperboard,
  Home,
  Layers3,
  MessageCircle,
  Radio,
  Sparkles,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  useEpisodeApprovalGates,
  useEpisodeAssets,
  useGenerationJobs,
  useSeedEpisodeProduction,
  useStartStoryAnalysis,
  useStoryAnalyses,
  useStoryboardShots,
  useStoryMap,
  useWorkflowRun,
} from '../../api/hooks'
import type { Episode, GenerationJob, Project, WorkflowRun } from '../../api/types'
import { agentFollowUpFeedbackLabel } from '../agentOutput'
import { AnalysisResultCard } from '../components/StoryAnalysisPanel'
import { ProductionFlowPanel } from '../components/ProductionFlowPanel'
import { WorkflowRecoveryTimeline } from '../components/WorkflowRecoveryTimeline'
import { ReviewSummaryChips } from '../components/ReviewSummaryChips'
import ChatDialog from '../components/ChatDialog'
import { useStudioSelection } from '../hooks/useStudioSelection'
import { buildStoryAnalysisReviewSnapshot } from '../reviewPersistence'
import { studioNavItems, studioRoutePaths } from '../routes'
import type { StudioShot } from '../types'
import {
  formatDuration,
  hasStoryMapItems,
  productionHint,
  resolveEpisodeWorkflowRunId,
  statusLabel,
  mapDisplayShots,
} from '../utils'

export function HomePage() {
  const { activeEpisode, selectedProject } = useStudioSelection()
  const [isChatOpen, setIsChatOpen] = useState(false)
  const startStoryAnalysis = useStartStoryAnalysis()
  const seedProduction = useSeedEpisodeProduction()
  const { data: analyses = [] } = useStoryAnalyses(activeEpisode?.id)
  const { data: storyMap } = useStoryMap(activeEpisode?.id)
  const { data: assets = [] } = useEpisodeAssets(activeEpisode?.id)
  const { data: gates = [] } = useEpisodeApprovalGates(activeEpisode?.id)
  const { data: jobs = [] } = useGenerationJobs()
  const { data: storyboardShots = [] } = useStoryboardShots(activeEpisode?.id)
  const displayShots = useMemo(() => mapDisplayShots(storyboardShots), [storyboardShots])
  const storyMapReady = hasStoryMapItems(storyMap)
  const currentWorkflowRunId = useMemo(
    () => resolveEpisodeWorkflowRunId(activeEpisode?.id, analyses, jobs),
    [activeEpisode?.id, analyses, jobs],
  )
  const { data: workflowRun } = useWorkflowRun(currentWorkflowRunId)
  const reviewSnapshot = useMemo(
    () => buildStoryAnalysisReviewSnapshot(analyses[0]),
    [analyses],
  )
  const nextHint = productionHint({
    activeEpisode,
    hasAnalysis: analyses.length > 0,
    storyMapReady,
  })
  const episodeJobs = useMemo(
    () => jobs.filter((job) => job.episode_id === activeEpisode?.id).slice(0, 5),
    [activeEpisode?.id, jobs],
  )
  const shotsCount = displayShots.filter((shot) => Boolean(shot.id)).length
  const reviewRelayHeadline = !reviewSnapshot
    ? '等待 review 队列'
    : reviewSnapshot.feedbackSummary.needs_follow_up > 0
      ? `${reviewSnapshot.feedbackSummary.needs_follow_up} 条待跟进待收口`
      : reviewSnapshot.feedbackSummary.unmarked > 0
        ? `${reviewSnapshot.feedbackSummary.unmarked} 条待标记`
        : reviewSnapshot.returnedHistorySummary.total > 0
          ? `已回传 ${reviewSnapshot.returnedHistorySummary.total} 条`
          : '当前协同已收口'
  const reviewRelayDescription = !reviewSnapshot
    ? '先完成一轮故事解析，导演台才会出现跨页 review relay。'
    : reviewSnapshot.feedbackSummary.needs_follow_up > 0
      ? `Storyboard ${reviewSnapshot.surfaceSummary.storyboard.needs_follow_up} 条、Assets / Graph ${reviewSnapshot.surfaceSummary.assetsGraph.needs_follow_up} 条待跟进，建议直接回到解析或跳去对应页面继续处理。`
      : reviewSnapshot.feedbackSummary.unmarked > 0
        ? '本轮已有回传，但仍有未标记 Agent；先回 Story Analysis 完成标记，再决定是否收口。'
        : reviewSnapshot.latestReturnedSummary
          ? `最近一次回传来自 ${reviewSnapshot.latestReturnedSummary.sourcePage} · ${reviewSnapshot.latestReturnedSummary.agentLabel} · ${agentFollowUpFeedbackLabel(reviewSnapshot.latestReturnedSummary.feedback)}。`
          : '当前没有跨页待跟进压力，可继续推进下一轮生产。'

  return (
    <section className="studio-page home-page" aria-labelledby="home-page-title">
      <div className="board-header">
        <div>
          <h1 id="home-page-title">Studio Home</h1>
          <span>把导演台拆成可扩展的多页面工作流。</span>
        </div>
        <div className="board-actions">
          <Link className="hero-secondary-action" to={studioRoutePaths.storyAnalysis}>
            <BookOpenText aria-hidden="true" />
            去故事解析
          </Link>
          <Link className="hero-secondary-action" to={studioRoutePaths.storyboard}>
            <Boxes aria-hidden="true" />
            打开分镜台
          </Link>
          <Link className="hero-secondary-action" to={studioRoutePaths.timelineExport}>
            <Activity aria-hidden="true" />
            查看时间线
          </Link>
        </div>
      </div>

      <WorkspaceHero
        activeEpisode={activeEpisode}
        analysesCount={analyses.length}
        assetsCount={assets.length}
        canSeedProduction={Boolean(activeEpisode && analyses.length > 0)}
        canStartAnalysis={Boolean(activeEpisode)}
        displayShots={displayShots}
        gatesCount={gates.length}
        jobs={episodeJobs}
        nextHint={nextHint}
        onSeedProduction={() => activeEpisode && seedProduction.mutate(activeEpisode.id)}
        onStartAnalysis={() => activeEpisode && startStoryAnalysis.mutate(activeEpisode.id)}
        onOpenChat={() => setIsChatOpen(true)}
        project={selectedProject}
        productionPending={seedProduction.isPending}
        shotsCount={shotsCount}
        startAnalysisPending={startStoryAnalysis.isPending}
        storyMapReady={storyMapReady}
      />

      <ProductionFlowPanel
        analysesCount={analyses.length}
        assetsCount={assets.length}
        gatesCount={gates.length}
        jobs={episodeJobs}
        nextHint={nextHint}
        shotsCount={shotsCount}
        storyMapReady={storyMapReady}
        workflowRun={workflowRun}
      />

      <WorkflowRecoverySummaryCard workflowRun={workflowRun} />

      <WorkflowRecoveryTimeline workflowRun={workflowRun} />

      <ChatDialog 
        isOpen={isChatOpen} 
        onClose={() => setIsChatOpen(false)} 
        episodeId={activeEpisode?.id || ''}
      />

      <div className="dashboard-grid">
        <article className="surface-card">
          <div className="panel-title-row">
            <div>
              <span>页面入口</span>
              <strong>当前 4 页 MVP</strong>
            </div>
          </div>
          <div className="page-link-grid">
            {studioNavItems.map((item) =>
              item.disabled ? (
                <div className="page-link-card disabled" key={item.key}>
                  <item.icon aria-hidden="true" />
                  <strong>{item.label}</strong>
                  <small>{item.description}</small>
                </div>
              ) : (
                <Link className="page-link-card" key={item.key} to={item.path}>
                  <item.icon aria-hidden="true" />
                  <strong>{item.label}</strong>
                  <small>{item.description}</small>
                </Link>
              ),
            )}
          </div>
        </article>

        <article className="surface-card">
          <div className="panel-title-row">
            <div>
              <span>最新解析摘要</span>
              <strong>{analyses[0]?.summary ?? '等待故事解析'}</strong>
            </div>
          </div>
          <AnalysisResultCard analysis={analyses[0]} />
        </article>

        <article className="surface-card review-relay-card">
          <div className="panel-title-row">
            <div>
              <span>Review relay</span>
              <strong>{reviewRelayHeadline}</strong>
            </div>
          </div>
          <p>{reviewRelayDescription}</p>
          {reviewSnapshot ? (
            <ReviewSummaryChips
              currentSide="storyAnalysis"
              storyboardPendingCount={reviewSnapshot.surfaceSummary.storyboard.needs_follow_up}
              assetsGraphPendingCount={reviewSnapshot.surfaceSummary.assetsGraph.needs_follow_up}
              totalReturnedCount={reviewSnapshot.returnedHistorySummary.total}
              returnedStoryboardCount={reviewSnapshot.returnedHistorySummary.storyboard}
              returnedAssetsGraphCount={reviewSnapshot.returnedHistorySummary.assetsGraph}
            />
          ) : null}
          <div className="review-relay-actions">
            <Link className="hero-secondary-action" to={studioRoutePaths.storyAnalysis}>
              <BookOpenText aria-hidden="true" />
              回导演台
            </Link>
            {reviewSnapshot?.surfaceSummary.storyboard.needs_follow_up ? (
              <Link className="hero-secondary-action" to={studioRoutePaths.storyboard}>
                <Boxes aria-hidden="true" />
                去 Storyboard
              </Link>
            ) : null}
            {reviewSnapshot?.surfaceSummary.assetsGraph.needs_follow_up ? (
              <Link className="hero-secondary-action" to={studioRoutePaths.assetsGraph}>
                <Layers3 aria-hidden="true" />
                去 Assets / Graph
              </Link>
            ) : null}
          </div>
        </article>

        <article className="surface-card">
          <div className="panel-title-row">
            <div>
              <span>分镜预览</span>
              <strong>{displayShots.length} 个镜头卡</strong>
            </div>
          </div>
          <div className="mini-shot-list" aria-label="分镜预览">
            {displayShots.slice(0, 3).map((shot) => (
              <div className="mini-shot-card" key={shot.key}>
                <span className={`reference-thumb ${shot.thumbnail}`} aria-hidden="true" />
                <div>
                  <strong>{shot.title}</strong>
                  <small>
                    {shot.sceneCode} · {formatDuration(shot.durationMS)}
                  </small>
                </div>
                <span className={`status-dot ${shot.status}`}>{statusLabel(shot.status)}</span>
              </div>
            ))}
          </div>
        </article>
      </div>
    </section>
  )
}

function WorkspaceHero({
  activeEpisode,
  analysesCount,
  assetsCount,
  canSeedProduction,
  canStartAnalysis,
  displayShots,
  gatesCount,
  jobs,
  nextHint,
  onSeedProduction,
  onStartAnalysis,
  onOpenChat,
  project,
  productionPending,
  shotsCount,
  startAnalysisPending,
  storyMapReady,
}: {
  activeEpisode?: Episode
  analysesCount: number
  assetsCount: number
  canSeedProduction: boolean
  canStartAnalysis: boolean
  displayShots: StudioShot[]
  gatesCount: number
  jobs: GenerationJob[]
  nextHint: string
  onSeedProduction: () => void
  onStartAnalysis: () => void
  onOpenChat: () => void
  project?: Project
  productionPending: boolean
  shotsCount: number
  startAnalysisPending: boolean
  storyMapReady: boolean
}) {
  const heroShot = displayShots[0]
  const activeJobs = jobs.filter((job) =>
    ['draft', 'polling', 'postprocessing', 'preflight', 'queued', 'submitting'].includes(
      job.status,
    ),
  ).length
  const readyShots = displayShots.filter((shot) =>
    ['approved', 'prompt_ready'].includes(shot.status),
  ).length
  const workspaceState = storyMapReady
    ? '生产链路已就绪'
    : analysesCount > 0
      ? '正在补齐资产图谱'
      : '等待故事解析'
  const heroMetrics: Array<{
    description: string
    icon: LucideIcon
    label: string
    tone: 'accent' | 'emerald' | 'violet' | 'warning'
    value: string
  }> = [
    {
      description: storyMapReady ? '角色 / 场景 / 道具可继续驱动资产生产。' : '先完成故事解析，再补齐资产图谱。',
      icon: Layers3,
      label: '资产图谱',
      tone: storyMapReady ? 'emerald' : 'warning',
      value: storyMapReady ? '已就绪' : '待生成',
    },
    {
      description: '首屏能看到真实分镜沉淀与当前镜头质量。',
      icon: Boxes,
      label: '分镜沉淀',
      tone: readyShots > 0 ? 'violet' : 'warning',
      value: `${readyShots}/${displayShots.length} 可用`,
    },
    {
      description: activeJobs > 0 ? '生成队列正在推进，适合盯关键镜头。' : '目前没有阻塞中的任务。',
      icon: Activity,
      label: '生成队列',
      tone: activeJobs > 0 ? 'accent' : 'emerald',
      value: `${activeJobs} 个活跃任务`,
    },
    {
      description: gatesCount > 0 ? '进入昂贵生成前可直接做人审确认。' : '审批点会在一键生产后自动出现。',
      icon: Sparkles,
      label: '导演确认',
      tone: gatesCount > 0 ? 'warning' : 'accent',
      value: `${gatesCount} 个审批点`,
    },
  ]

  return (
    <section className="workspace-hero" aria-label="导演工作台总览">
      <div className="hero-spotlight">
        <span className="hero-kicker">Director cockpit</span>
        <div className="hero-title-row">
          <div>
            <h2>AI 漫剧导演台</h2>
            <p>把故事源、资产图谱、分镜卡、时间线和导出状态分流到独立页面，但在首页保留整条生产链的核心信号。</p>
          </div>
          <span
            className={
              storyMapReady ? 'hero-state ready' : analysesCount > 0 ? 'hero-state active' : 'hero-state'
            }
          >
            {workspaceState}
          </span>
        </div>
        <div className="hero-chip-row">
          <span className="hero-chip">
            <Home aria-hidden="true" />
            {project?.name ?? '演示工作区'}
          </span>
          <span className="hero-chip">
            <Clapperboard aria-hidden="true" />
            {activeEpisode
              ? `第 ${activeEpisode.number.toString().padStart(2, '0')} 集 · ${activeEpisode.title}`
              : '创建项目后接入真实剧集'}
          </span>
          <span className="hero-chip">
            <Radio aria-hidden="true" />
            {nextHint}
          </span>
        </div>
        <div className="hero-action-row">
          <button
            className="hero-primary-action"
            disabled={!canSeedProduction || productionPending}
            onClick={onSeedProduction}
            type="button"
          >
            <Sparkles aria-hidden="true" />
            {productionPending ? '生产中...' : '一键生产分镜包'}
          </button>
          <button
            className="hero-secondary-action"
            disabled={!canStartAnalysis || startAnalysisPending}
            onClick={onStartAnalysis}
            type="button"
          >
            <BookOpenText aria-hidden="true" />
            {startAnalysisPending ? '解析中...' : '启动故事解析'}
          </button>
          <button
            className="hero-secondary-action"
            onClick={onOpenChat}
            type="button"
            title="打开 Chat 对话框"
          >
            <MessageCircle aria-hidden="true" />
            Ask AI
          </button>
          <Link className="hero-link-action" to={studioRoutePaths.timelineExport}>
            <Activity aria-hidden="true" />
            查看时间线
          </Link>
        </div>
        {heroShot ? (
          <div className="hero-preview-card">
            <div className={`hero-preview-thumb ${heroShot.thumbnail}`} aria-hidden="true" />
            <div className="hero-preview-copy">
              <span>当前关键镜头</span>
              <strong>{heroShot.title}</strong>
              <p>{heroShot.description}</p>
              <div className="hero-preview-meta">
                <span className={`status-dot ${heroShot.status}`}>{statusLabel(heroShot.status)}</span>
                <span>{heroShot.progress}% 进度</span>
                <span>{formatDuration(heroShot.durationMS)}</span>
              </div>
            </div>
          </div>
        ) : null}
      </div>

      <div className="hero-stat-grid">
        <article className="hero-stat-card feature">
          <span className="hero-card-label">工作台摘要</span>
          <strong>{shotsCount > 0 ? `${shotsCount} 个真实分镜已接入` : '先生成真实分镜卡'}</strong>
          <p>
            {analysesCount > 0
              ? `已生成 ${analysesCount} 份剧情分析，可继续推资产与镜头。`
              : '当前仍在演示态，可先录入故事源并启动多 Agent 解析。'}
          </p>
          <div className="hero-stat-inline">
            <span>{assetsCount} 个候选资产</span>
            <span>{displayShots.length} 个镜头卡</span>
          </div>
        </article>
        {heroMetrics.map((metric) => (
          <article className={`hero-stat-card ${metric.tone}`} key={metric.label}>
            <div className="hero-card-icon">
              <metric.icon aria-hidden="true" />
            </div>
            <span className="hero-card-label">{metric.label}</span>
            <strong>{metric.value}</strong>
            <p>{metric.description}</p>
          </article>
        ))}
      </div>
    </section>
  )
}

function WorkflowRecoverySummaryCard({ workflowRun }: { workflowRun?: WorkflowRun }) {
  if (!workflowRun) return null
  const summary = workflowRun.checkpoint_summary
  const nodes = workflowRun.node_runs ?? []
  const hasAny =
    Boolean(summary) ||
    nodes.length > 0 ||
    (workflowRun.status && workflowRun.status !== 'draft')
  if (!hasAny) return null
  const counts = nodes.reduce<Record<string, number>>((acc, node) => {
    acc[node.status] = (acc[node.status] ?? 0) + 1
    return acc
  }, {})
  const total = nodes.length
  const succeeded = counts.succeeded ?? 0
  const failed = counts.failed ?? 0
  const running = counts.running ?? 0
  const waiting = (counts.pending ?? 0) + (counts.waiting_approval ?? 0)
  const skipped = counts.skipped ?? 0
  const checkpointSeq = summary?.sequence ?? 0
  const savedAt = summary?.saved_at
  const savedAtLabel = savedAt
    ? new Date(savedAt).toLocaleString()
    : '尚未保存检查点'
  const headline =
    workflowRun.status === 'succeeded'
      ? '工作流已收口'
      : workflowRun.status === 'failed'
        ? '工作流失败需介入'
        : workflowRun.status === 'waiting_approval'
          ? '工作流等待审批'
          : workflowRun.status === 'running'
            ? '工作流执行中'
            : '工作流就绪'
  return (
    <article
      className="surface-card"
      aria-label="Workflow recovery summary"
      style={{ display: 'flex', flexDirection: 'column', gap: 10 }}
    >
      <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 12 }}>
        <div>
          <span className="section-kicker">Recovery snapshot</span>
          <strong style={{ display: 'block', marginTop: 4 }}>{headline}</strong>
        </div>
        <small className="muted" style={{ fontSize: 12 }}>
          检查点 #{checkpointSeq} · {savedAtLabel}
        </small>
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(112px, 1fr))',
          gap: 8,
        }}
      >
        <SummaryStat label="节点总数" value={total} />
        <SummaryStat label="已完成" value={succeeded} tone={succeeded > 0 ? 'ok' : undefined} />
        <SummaryStat label="运行中" value={running} tone={running > 0 ? 'warn' : undefined} />
        <SummaryStat label="待执行" value={waiting} />
        <SummaryStat label="失败" value={failed} tone={failed > 0 ? 'err' : undefined} />
        <SummaryStat label="跳过" value={skipped} />
      </div>
      {summary && summary.blackboard_roles.length > 0 && (
        <div style={{ fontSize: 12, opacity: 0.75 }}>
          黑板角色：{summary.blackboard_roles.join('、')}
        </div>
      )}
    </article>
  )
}

function SummaryStat({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone?: 'ok' | 'warn' | 'err'
}) {
  const color = tone === 'ok' ? '#3ddc84' : tone === 'warn' ? '#ff8c42' : tone === 'err' ? '#ff6b6b' : undefined
  return (
    <div
      style={{
        padding: '8px 10px',
        borderRadius: 8,
        background: 'rgba(255,255,255,0.04)',
        border: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      <div style={{ fontSize: 11, opacity: 0.65 }}>{label}</div>
      <div style={{ fontSize: 18, fontWeight: 600, color }}>{value}</div>
    </div>
  )
}
