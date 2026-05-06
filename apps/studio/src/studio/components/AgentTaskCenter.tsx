import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Activity, CheckCircle2, ChevronDown, ChevronUp, CircleAlert, Loader2, XCircle } from 'lucide-react'

import { invalidateAgentTaskQueries } from '../../api/agentTaskInvalidation'
import { getAgentRunStatus, reconnectAgentRun } from '../../api/agentStream'
import type { StoryAgentOutput } from '../../api/types'
import type { AgentTask, AgentTaskStatus } from '../../state/agentTaskCenterStore'
import { useAgentTaskCenterStore } from '../../state/agentTaskCenterStore'
import type { AgentStreamEvent } from '../../api/types'
import { AgentTaskResultCards } from './AgentTaskResultCards'
import { agentRoleLabel } from '../utils'

type AgentTaskCenterProps = {
  externalAgents?: StoryAgentOutput[]
  selectedExternalRole?: string
  onSelectExternalRole?: (role: string) => void
}

type RenderTask = {
  origin: 'live' | 'analysis'
  task: AgentTask
}

function taskStatusLabel(status: AgentTaskStatus): string {
  const labels = {
    cancelled: '已取消',
    done: '已完成',
    error: '失败',
    running: '运行中',
  }
  return labels[status]
}

function taskStatusTone(status: AgentTaskStatus): string {
  const tones = {
    cancelled: 'cancelled',
    done: 'done',
    error: 'error',
    running: 'running',
  }
  return tones[status]
}

function toTaskStatus(status: StoryAgentOutput['status']): AgentTaskStatus {
  switch (status) {
    case 'failed':
      return 'error'
    case 'running':
    case 'waiting':
      return 'running'
    case 'skipped':
      return 'cancelled'
    default:
      return 'done'
  }
}

function buildExternalTask(agent: StoryAgentOutput): AgentTask {
  return {
    id: `analysis-${agent.role}`,
    role: agent.role,
    sourceText:
      agent.highlights[0] ??
      (agent.output.slice(0, 120) || `${agentRoleLabel(agent.role)} 输出`),
    contextKeys: [],
    createdAt: '',
    updatedAt: '',
    status: toTaskStatus(agent.status),
    streamedText: agent.output,
    errorMessage: agent.status === 'failed' ? '该 Agent 在分析流程中返回失败状态。' : undefined,
    events: [],
  }
}

function formatTaskTimestamp(value: string, fallback: string): string {
  return value ? new Date(value).toLocaleString() : fallback
}

function eventTypeLabel(type: AgentStreamEvent['type']): string {
  const labels: Record<AgentStreamEvent['type'], string> = {
    CANCELLED: '已取消',
    CONTENT: '内容',
    DONE: '完成',
    ERROR: '错误',
    REASONING: '推理',
    TOOL_CALL: '工具调用',
    TOOL_FINISHED: '工具完成',
  }
  return labels[type]
}

function eventPreview(event: AgentStreamEvent): string {
  if (event.content) return event.content
  if (event.output) return event.output
  if (event.error) return event.error
  if (event.tool_name) return event.tool_name
  if (event.tool_calls?.length) return event.tool_calls.map((tool) => tool.name).join(', ')
  return '等待更多事件…'
}

export function AgentTaskCenter({
  externalAgents = [],
  selectedExternalRole,
  onSelectExternalRole,
}: AgentTaskCenterProps) {
  const [reconnectingTaskId, setReconnectingTaskId] = useState<string | null>(null)
  const [reconnectNotice, setReconnectNotice] = useState<string | null>(null)
  const invalidatedSignaturesRef = useRef<Set<string>>(new Set())
  const queryClient = useQueryClient()
  const tasks = useAgentTaskCenterStore((state) => state.tasks)
  const notificationOpen = useAgentTaskCenterStore((state) => state.notificationOpen)
  const panelExpanded = useAgentTaskCenterStore((state) => state.panelExpanded)
  const selectedTaskId = useAgentTaskCenterStore((state) => state.selectedTaskId)
  const restoredTaskCount = useAgentTaskCenterStore((state) => state.restoredTaskCount)
  const clearCompleted = useAgentTaskCenterStore((state) => state.clearCompleted)
  const attachRun = useAgentTaskCenterStore((state) => state.attachRun)
  const appendTaskEvent = useAgentTaskCenterStore((state) => state.appendEvent)
  const completeTask = useAgentTaskCenterStore((state) => state.completeTask)
  const dismissRestoreNotice = useAgentTaskCenterStore((state) => state.dismissRestoreNotice)
  const failTask = useAgentTaskCenterStore((state) => state.failTask)
  const removeTask = useAgentTaskCenterStore((state) => state.removeTask)
  const resumeTask = useAgentTaskCenterStore((state) => state.resumeTask)
  const selectTask = useAgentTaskCenterStore((state) => state.selectTask)
  const setNotificationOpen = useAgentTaskCenterStore((state) => state.setNotificationOpen)
  const setPanelExpanded = useAgentTaskCenterStore((state) => state.setPanelExpanded)

  const renderTasks: RenderTask[] = [
    ...tasks.map((task) => ({ origin: 'live' as const, task })),
    ...externalAgents.map((agent) => ({ origin: 'analysis' as const, task: buildExternalTask(agent) })),
  ]
  const selectedTask =
    (selectedExternalRole ? renderTasks.find((entry) => entry.task.id === `analysis-${selectedExternalRole}`) : undefined) ??
    renderTasks.find((entry) => entry.task.id === selectedTaskId) ??
    renderTasks[0]
  const runningCount = renderTasks.filter((entry) => entry.task.status === 'running').length
  const doneCount = renderTasks.filter((entry) => entry.task.status === 'done').length
  const errorCount = renderTasks.filter((entry) => entry.task.status === 'error').length
  const cancelledCount = renderTasks.filter((entry) => entry.task.status === 'cancelled').length
  const hasFinishedLiveTasks = tasks.some((task) => task.status !== 'running')

  useEffect(() => {
    const invalidateReadyTasks = async () => {
      for (const task of tasks) {
        if (task.status !== 'done' || !task.episodeId) continue
        const signature = `${task.id}:${task.lastEventSequence ?? 0}:${task.status}`
        if (invalidatedSignaturesRef.current.has(signature)) continue
        await invalidateAgentTaskQueries(queryClient, task)
        invalidatedSignaturesRef.current.add(signature)
      }
    }
    void invalidateReadyTasks()
  }, [queryClient, tasks])

  const handleReconnect = async () => {
    if (!selectedTask || selectedTask.origin !== 'live' || !selectedTask.task.runId) {
      return
    }
    setReconnectNotice(null)
    setReconnectingTaskId(selectedTask.task.id)
    try {
      const snapshot = await getAgentRunStatus(selectedTask.task.runId)
      if (snapshot.status === 'running') {
        resumeTask(selectedTask.task.id)
      }
      await reconnectAgentRun(
        selectedTask.task.runId,
        selectedTask.task.lastEventSequence ?? 0,
        {
          onRunStarted: (runId) => attachRun(selectedTask.task.id, runId),
          onEvent: (event) => appendTaskEvent(selectedTask.task.id, event),
          onDone: (frame) => completeTask(selectedTask.task.id, frame),
          onError: (message) => failTask(selectedTask.task.id, message),
        },
      )
      setReconnectNotice('已完成服务端状态同步。')
    } catch (error) {
      const message = error instanceof Error ? error.message : '恢复连接失败'
      failTask(selectedTask.task.id, message)
      setReconnectNotice(message)
    } finally {
      setReconnectingTaskId(null)
    }
  }

  return (
    <section className="agent-task-center" aria-labelledby="agent-task-center-title">
      <header className="agent-task-center-header">
        <div>
          <span className="section-kicker">Agent task center</span>
          <h2 id="agent-task-center-title">任务中心</h2>
          <p>集中查看实时任务与分析结果快照，并把输出提炼成可读的结构化结果卡片。</p>
        </div>
        <div className="agent-task-center-actions">
          <button
            type="button"
            className="btn-ghost"
            onClick={() => setNotificationOpen(!notificationOpen)}
            aria-expanded={notificationOpen}
          >
            {notificationOpen ? <ChevronUp size={16} aria-hidden="true" /> : <ChevronDown size={16} aria-hidden="true" />}
            {notificationOpen ? '收起任务中心' : '展开任务中心'}
          </button>
          <button
            type="button"
            className="btn-ghost"
            onClick={() => setPanelExpanded(!panelExpanded)}
            disabled={!notificationOpen || tasks.length === 0}
          >
            {panelExpanded ? '紧凑视图' : '详情视图'}
          </button>
          <button type="button" className="btn-ghost" onClick={clearCompleted} disabled={!hasFinishedLiveTasks}>
            清理已结束
          </button>
        </div>
      </header>

      <div className="agent-task-center-summary" aria-live="polite">
        <span className="agent-task-chip">
          <Loader2 size={14} aria-hidden="true" />
          运行中 {runningCount}
        </span>
        <span className="agent-task-chip done">
          <CheckCircle2 size={14} aria-hidden="true" />
          已完成 {doneCount}
        </span>
        <span className="agent-task-chip error">
          <CircleAlert size={14} aria-hidden="true" />
          失败 {errorCount}
        </span>
        <span className="agent-task-chip cancelled">
          <XCircle size={14} aria-hidden="true" />
          已取消 {cancelledCount}
        </span>
      </div>

      {restoredTaskCount > 0 ? (
        <div className="agent-task-restore-banner" role="status">
          <div>
            <strong>已恢复 {restoredTaskCount} 个最近任务</strong>
            <p>此前未结束的流式任务已标记为“已取消”，当前内容来自本地回放缓存。</p>
          </div>
          <button type="button" className="btn-ghost" onClick={dismissRestoreNotice}>
            知道了
          </button>
        </div>
      ) : null}

      {!notificationOpen ? null : renderTasks.length === 0 ? (
        <div className="agent-task-center-empty">
          <Activity size={18} aria-hidden="true" />
          <div>
            <strong>还没有任务</strong>
            <p>运行一次流式调试，或进入已有 Agent 分析结果页面后，这里会自动显示对应任务。</p>
          </div>
        </div>
      ) : (
        <div className={`agent-task-center-layout ${panelExpanded ? 'expanded' : 'compact'}`}>
          <div className="agent-task-center-list" aria-label="Agent 任务列表">
            {renderTasks.map(({ origin, task }) => {
              const isSelected = selectedTask?.task.id === task.id
              return (
                <button
                  key={task.id}
                  type="button"
                  className={`agent-task-card ${isSelected ? 'selected' : ''}`}
                  onClick={() => {
                    selectTask(task.id)
                    if (origin === 'analysis') {
                      onSelectExternalRole?.(task.role)
                    }
                  }}
                  aria-pressed={isSelected}
                >
                  <div className="agent-task-card-header">
                    <strong>{agentRoleLabel(task.role)}</strong>
                    <span className={`agent-task-status ${taskStatusTone(task.status)}`}>{taskStatusLabel(task.status)}</span>
                  </div>
                  <span className={`agent-task-origin ${origin}`}>{origin === 'live' ? '实时任务' : '分析快照'}</span>
                  <p>{task.sourceText.slice(0, 88)}{task.sourceText.length > 88 ? '…' : ''}</p>
                  <div className="agent-task-card-meta">
                    <span>{task.createdAt ? new Date(task.createdAt).toLocaleTimeString() : '当前分析结果'}</span>
                    <span>{task.events.length} 个事件</span>
                    {task.doneFrame?.token_count ? <span>{task.doneFrame.token_count} tok</span> : null}
                  </div>
                </button>
              )
            })}
          </div>

          {panelExpanded && selectedTask ? (
            <aside className="agent-task-center-detail" aria-label={`${agentRoleLabel(selectedTask.task.role)} 任务详情`}>
              <div className="agent-task-detail-header">
                <div>
                  <strong>{agentRoleLabel(selectedTask.task.role)}</strong>
                  <span className={`agent-task-status ${taskStatusTone(selectedTask.task.status)}`}>{taskStatusLabel(selectedTask.task.status)}</span>
                </div>
                {selectedTask.origin === 'live' ? (
                  <button type="button" className="btn-ghost" onClick={() => removeTask(selectedTask.task.id)}>
                    从列表移除
                  </button>
                ) : (
                  <span className="agent-task-origin analysis">只读快照</span>
                )}
              </div>
              <dl className="agent-task-detail-meta">
                <div>
                  <dt>创建时间</dt>
                  <dd>{formatTaskTimestamp(selectedTask.task.createdAt, '来自当前分析结果')}</dd>
                </div>
                <div>
                  <dt>最近更新</dt>
                  <dd>{formatTaskTimestamp(selectedTask.task.updatedAt, selectedTask.origin === 'analysis' ? '随 StoryAnalysis 刷新' : '—')}</dd>
                </div>
                <div>
                  <dt>Context keys</dt>
                  <dd>{selectedTask.task.contextKeys.length > 0 ? selectedTask.task.contextKeys.join(', ') : '无'}</dd>
                </div>
                <div>
                  <dt>Token</dt>
                  <dd>{selectedTask.task.doneFrame?.token_count ?? (selectedTask.origin === 'analysis' ? '由分析接口提供' : '—')}</dd>
                </div>
                {selectedTask.origin === 'live' ? (
                  <>
                    <div>
                      <dt>Run ID</dt>
                      <dd>{selectedTask.task.runId ?? '尚未建立'}</dd>
                    </div>
                    <div>
                      <dt>Sequence</dt>
                      <dd>{selectedTask.task.lastEventSequence ?? 0}</dd>
                    </div>
                  </>
                ) : null}
              </dl>
              {selectedTask.origin === 'live' && selectedTask.task.runId ? (
                <div className="agent-task-reconnect-row">
                  <button
                    type="button"
                    className="btn-ghost"
                    onClick={handleReconnect}
                    disabled={reconnectingTaskId === selectedTask.task.id}
                  >
                    {reconnectingTaskId === selectedTask.task.id ? '恢复中…' : '恢复连接'}
                  </button>
                  <small>从服务端 `run_id` + `last sequence` 拉取缺失事件，并在任务仍运行时继续追流。</small>
                </div>
              ) : null}
              {reconnectNotice ? <div className="agent-task-reconnect-notice">{reconnectNotice}</div> : null}
              {selectedTask.task.errorMessage ? (
                <div className="agent-task-detail-error" role="alert">
                  {selectedTask.task.errorMessage}
                </div>
              ) : null}
              <div className="agent-task-detail-block">
                <h3>结构化结果</h3>
                <AgentTaskResultCards task={selectedTask.task} />
              </div>
              <div className="agent-task-detail-block">
                <h3>原始输出</h3>
                <pre className="agent-task-detail-output">
                  {selectedTask.task.streamedText || '（等待首个 CONTENT 事件…）'}
                </pre>
              </div>
              <div className="agent-task-detail-block">
                <h3>事件时间线</h3>
                <div className="agent-task-event-list">
                  {selectedTask.task.events.length === 0 ? (
                    <p className="agent-task-empty-copy">
                      {selectedTask.origin === 'analysis' ? '分析快照不包含实时 SSE 时间线。' : '还没有事件。'}
                    </p>
                  ) : (
                    selectedTask.task.events.map((event, index) => (
                      <article key={`${selectedTask.task.id}-${index}`} className="agent-task-event-item">
                        <span className={`agent-task-event-badge type-${event.type.toLowerCase()}`}>{eventTypeLabel(event.type)}</span>
                        <div>
                          <strong>{event.role ? agentRoleLabel(event.role) : agentRoleLabel(selectedTask.task.role)}</strong>
                          <p>{eventPreview(event)}</p>
                        </div>
                      </article>
                    ))
                  )}
                </div>
              </div>
            </aside>
          ) : null}
        </div>
      )}
    </section>
  )
}
