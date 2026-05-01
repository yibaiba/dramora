import { Activity, Boxes, Download, Film } from 'lucide-react'
import { useMemo } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  useEpisodeTimeline,
  useGenerationJobs,
  useStoryboardShots,
  useStoryAnalyses,
  useWorkflowRun,
} from '../../api/hooks'
import { TimelineWorkspace } from '../components/TimelineWorkspace'
import { WorkflowRecoveryTimeline } from '../components/WorkflowRecoveryTimeline'
import { useStudioSelection } from '../hooks/useStudioSelection'
import { studioRoutePaths } from '../routes'
import {
  formatCheckpointSavedAt,
  formatTimecode,
  mapDisplayShots,
  mapTimelineDisplayShots,
  resolveEpisodeWorkflowRunId,
  workflowRunStatusLabel,
} from '../utils'

export function TimelineExportPage() {
  const { activeEpisode } = useStudioSelection()
  const location = useLocation()
  const { data: storyboardShots = [] } = useStoryboardShots(activeEpisode?.id)
  const { data: timeline } = useEpisodeTimeline(activeEpisode?.id)
  const { data: jobs = [] } = useGenerationJobs()
  const { data: analyses = [] } = useStoryAnalyses(activeEpisode?.id)
  const storyboardDisplayShots = useMemo(
    () => mapDisplayShots(storyboardShots),
    [storyboardShots],
  )
  const displayShots = useMemo(
    () =>
      timeline
        ? mapTimelineDisplayShots(timeline, storyboardShots)
        : storyboardDisplayShots,
    [storyboardDisplayShots, storyboardShots, timeline],
  )
  const timelineSource = timeline ? 'saved' : 'storyboard'
  const exportJobs = jobs.filter((job) => job.episode_id === activeEpisode?.id && job.task_type === 'export')
  const currentWorkflowRunId = useMemo(
    () => resolveEpisodeWorkflowRunId(activeEpisode?.id, analyses, jobs),
    [activeEpisode?.id, analyses, jobs],
  )
  const { data: workflowRun } = useWorkflowRun(currentWorkflowRunId)
  const checkpointSummary = workflowRun?.checkpoint_summary
  const duration = timeline?.duration_ms ?? displayShots.reduce((total, shot) => total + shot.durationMS, 0)
  const handoffState = location.state as
    | { fromStoryboard?: boolean; selectedShotCode?: string; shotsCount?: number }
    | null

  return (
    <section className="studio-page timeline-page" aria-labelledby="timeline-export-title">
      <div className="board-header">
        <div>
          <h1 id="timeline-export-title">Timeline / Export</h1>
          <span>汇总镜头到剪辑时间线，保存版本并发起导出。</span>
        </div>
        <div className="board-actions">
          <Link className="hero-secondary-action" to={studioRoutePaths.storyboard}>
            <Boxes aria-hidden="true" />
            回到分镜台
          </Link>
          <Link className="hero-secondary-action" to={studioRoutePaths.home}>
            <Activity aria-hidden="true" />
            返回 Home
          </Link>
        </div>
      </div>

      {handoffState?.fromStoryboard ? (
        <div className="board-notice timeline-handoff-notice">
          已从 Storyboard 接入 {handoffState.shotsCount ?? displayShots.length} 个镜头到 Timeline 草稿
          {handoffState.selectedShotCode ? ` · 当前来自第 ${handoffState.selectedShotCode} 镜` : ''}
          {timeline ? ` · 当前时间线版本 v${timeline.version}` : ''}
        </div>
      ) : null}

      <div className="dashboard-grid">
        <article className="surface-card">
          <span className="section-kicker">Timeline version</span>
          <strong>{timeline ? `v${timeline.version}` : '未保存'}</strong>
          <p>{timeline ? `${timeline.tracks.length} 条轨道 · ${timeline.status}` : '当前仍显示演示时间线。'}</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Shot assembly</span>
          <strong>{displayShots.length} 个镜头</strong>
          <p>
            {timelineSource === 'saved'
              ? `当前按已保存 timeline v${timeline?.version ?? 1} 展示，总时长 ${formatTimecode(duration)}。`
              : `当前按 storyboard 镜头派生草稿展示，总时长 ${formatTimecode(duration)}。`}
          </p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Export queue</span>
          <strong>{exportJobs.length} 个导出任务</strong>
          <p>导出任务沿用现有 generation jobs，不在组件里复制服务端状态。</p>
        </article>
        <article className="surface-card checkpoint-rail">
          <span className="section-kicker">Recovery snapshot</span>
          <strong>{workflowRun ? workflowRunStatusLabel(workflowRun.status) : '等待上游解析'}</strong>
          <p>
            {checkpointSummary
              ? `沿用故事解析 workflow #${checkpointSummary.sequence}，${formatCheckpointSavedAt(checkpointSummary.saved_at)}。`
              : '当前还没有可复用的 checkpoint；先回到 Story Analysis 启动或恢复上游解析。'}
          </p>
          {checkpointSummary ? (
            <div className="job-pill-row" aria-label="上游 workflow checkpoint 摘要">
              <span className="job-pill succeeded">已完成 {checkpointSummary.completed_nodes}</span>
              <span className={`job-pill ${checkpointSummary.waiting_nodes > 0 ? 'queued' : 'succeeded'}`}>
                待执行 {checkpointSummary.waiting_nodes}
              </span>
              <span className={`job-pill ${checkpointSummary.running_nodes > 0 ? 'running' : 'queued'}`}>
                运行中 {checkpointSummary.running_nodes}
              </span>
              <span className={`job-pill ${checkpointSummary.failed_nodes > 0 ? 'failed' : 'succeeded'}`}>
                失败 {checkpointSummary.failed_nodes}
              </span>
              <span className="job-pill">Blackboard {checkpointSummary.blackboard_roles.length}</span>
            </div>
          ) : null}
        </article>
      </div>

      <WorkflowRecoveryTimeline workflowRun={workflowRun} />

      <article className="surface-card">
        <div className="panel-title-row">
          <div>
            <span>编辑台</span>
            <strong>时间线与导出面板</strong>
          </div>
        </div>
        <TimelineWorkspace
          activeEpisode={activeEpisode}
          displayShots={displayShots}
          timeline={timeline}
          timelineSource={timelineSource}
        />
      </article>

      <div className="page-link-grid">
        <Link className="page-link-card" to={studioRoutePaths.storyboard}>
          <Film aria-hidden="true" />
          <strong>继续补镜头</strong>
          <small>先回到 Storyboard 调整提示词或审批状态。</small>
        </Link>
        <Link className="page-link-card" to={studioRoutePaths.storyAnalysis}>
          <Download aria-hidden="true" />
          <strong>补故事源</strong>
          <small>如分镜质量不足，可回到分析页补充原文并重跑解析。</small>
        </Link>
      </div>
    </section>
  )
}
