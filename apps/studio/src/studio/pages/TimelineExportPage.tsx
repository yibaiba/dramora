import { Activity, Boxes, Download, Film, Edit2, AlertCircle } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  useEpisodeTimeline,
  useGenerationJobs,
  useStoryboardShots,
  useStoryAnalyses,
  useWorkflowRun,
  useSaveEpisodeTimeline,
} from '../../api/hooks'
import type { Timeline as APITimeline } from '../../api/types'
import { generateFCPXML } from '../../lib/fcpxml-generator'
import type { Timeline as EditorTimeline } from '../lib/editor/types'
import { TimelineWorkspace } from '../components/TimelineWorkspace'
import { WorkflowRecoveryTimeline } from '../components/WorkflowRecoveryTimeline'
import { EditVideoModal } from '../components/editor/EditVideoModal'
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

// Convert API Timeline to Editor Timeline
function convertApiTimelineToEditor(apiTimeline: APITimeline | undefined): EditorTimeline | undefined {
  if (!apiTimeline) return undefined

  return {
    tracks: apiTimeline.tracks.map((track) => ({
      id: track.id,
      type: track.kind as 'video' | 'audio' | 'subtitle',
      name: track.name,
      clips: track.clips.map((clip) => ({
        id: clip.id,
        trackId: track.id,
        startTime: clip.start_ms,
        duration: clip.duration_ms,
        sourceUrl: clip.asset_id,
        properties: {
          speed: 1.0,
          opacity: 1,
        },
      })),
      visible: true,
      locked: false,
      height: 60,
    })),
    duration: apiTimeline.duration_ms,
    fps: 30, // Default fps from API
  }
}

// Convert Editor Timeline to API Timeline (for saving)
function convertEditorTimelineToApi(
  editorTimeline: EditorTimeline,
  originalApiTimeline: APITimeline | undefined,
): APITimeline {
  const now = new Date().toISOString()
  return {
    id: originalApiTimeline?.id ?? '',
    episode_id: originalApiTimeline?.episode_id ?? '',
    status: originalApiTimeline?.status ?? 'draft',
    version: (originalApiTimeline?.version ?? 0) + 1,
    duration_ms: editorTimeline.duration,
    tracks: editorTimeline.tracks.map((track) => {
      const originalTrack = originalApiTimeline?.tracks.find((t) => t.id === track.id)
      return {
        id: track.id,
        kind: track.type,
        name: track.name,
        position: editorTimeline.tracks.indexOf(track),
        clips: track.clips.map((clip) => {
          const originalClip = originalTrack?.clips.find((c) => c.id === clip.id)
          return {
            id: clip.id,
            asset_id: clip.sourceUrl,
            kind: track.type,
            start_ms: clip.startTime,
            duration_ms: clip.duration,
            trim_start_ms: originalClip?.trim_start_ms ?? 0,
            created_at: originalClip?.created_at ?? now,
            updated_at: now,
          }
        }),
        created_at: originalTrack?.created_at ?? now,
        updated_at: now,
      }
    }),
    created_at: originalApiTimeline?.created_at ?? now,
    updated_at: now,
  }
}


export function TimelineExportPage() {
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [saveMessage, setSaveMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const { activeEpisode } = useStudioSelection()
  const location = useLocation()
  const { data: storyboardShots = [] } = useStoryboardShots(activeEpisode?.id)
  const { data: timeline } = useEpisodeTimeline(activeEpisode?.id)
  const { data: jobs = [] } = useGenerationJobs()
  const { data: analyses = [] } = useStoryAnalyses(activeEpisode?.id)
  const saveTimeline = useSaveEpisodeTimeline()
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
  const editorTimeline = useMemo(() => convertApiTimelineToEditor(timeline), [timeline])
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

  const handleExportFCPXML = () => {
    if (!timeline) {
      console.warn('Timeline not available for FCPXML export')
      return
    }

    try {
      const assetsMap = new Map()
      const fcpxmlContent = generateFCPXML(timeline, assetsMap)
      const blob = new Blob([fcpxmlContent], { type: 'application/xml' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `episode-${activeEpisode?.id}-timeline-${Date.now()}.fcpxml`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    } catch (error) {
      console.error('FCPXML 导出失败:', error)
    }
  }

  return (
    <section className="studio-page timeline-page" aria-labelledby="timeline-export-title">
      {saveMessage && (
        <div
          className={`board-notice timeline-${saveMessage.type}-notice`}
          style={{
            backgroundColor: saveMessage.type === 'success' ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
            borderColor: saveMessage.type === 'success' ? '#22c55e' : '#ef4444',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          <AlertCircle
            aria-hidden="true"
            style={{
              color: saveMessage.type === 'success' ? '#22c55e' : '#ef4444',
              flexShrink: 0,
            }}
          />
          <span>{saveMessage.text}</span>
        </div>
      )}
      <div className="board-header">
        <div>
          <h1 id="timeline-export-title">Timeline / Export</h1>
          <span>汇总镜头到剪辑时间线，保存版本并发起导出。</span>
        </div>
        <div className="board-actions">
          <button
            onClick={() => setIsEditOpen(true)}
            disabled={!timeline}
            className="hero-secondary-action"
            title={!timeline ? '请先保存时间线' : '打开高级编辑器'}
          >
            <Edit2 aria-hidden="true" />
            编辑时间线
          </button>
          <button
            onClick={handleExportFCPXML}
            disabled={!timeline}
            className="hero-secondary-action"
            title={!timeline ? '请先保存时间线' : '导出为 FCPXML 格式'}
          >
            <Download aria-hidden="true" />
            导出 FCPXML
          </button>
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

      {/* Advanced editing modal */}
      <EditVideoModal
        isOpen={isEditOpen}
        initialTimeline={editorTimeline}
        videoTitle={`编辑 Timeline v${timeline?.version ?? 1}`}
        onClose={() => setIsEditOpen(false)}
        onSave={(editedTimeline) => {
          // Convert edited timeline back to API format and save
          if (!activeEpisode || !timeline) {
            setSaveMessage({ type: 'error', text: '缺少必要的上下文信息' })
            return
          }

          try {
            const updatedApiTimeline = convertEditorTimelineToApi(editedTimeline, timeline)
            saveTimeline.mutate(
              {
                episodeId: activeEpisode.id,
                request: {
                  tracks: updatedApiTimeline.tracks,
                  duration_ms: updatedApiTimeline.duration_ms,
                },
              },
              {
                onSuccess: () => {
                  setSaveMessage({ type: 'success', text: '时间线已保存成功' })
                  setIsEditOpen(false)
                  // Auto-dismiss message after 3 seconds
                  setTimeout(() => setSaveMessage(null), 3000)
                },
                onError: (error) => {
                  setSaveMessage({
                    type: 'error',
                    text: `保存失败: ${error instanceof Error ? error.message : '未知错误'}`,
                  })
                },
              },
            )
          } catch (error) {
            setSaveMessage({
              type: 'error',
              text: `转换失败: ${error instanceof Error ? error.message : '未知错误'}`,
            })
          }
        }}
      />
    </section>
  )
}
