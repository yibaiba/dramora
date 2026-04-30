import { useState } from 'react'
import { ListFilter } from 'lucide-react'
import type { StoryAgentOutput } from '../../api/types'
import type {
  AgentFeedbackFilter,
  AgentFollowUpFeedback,
  ReturnedFollowUpHistoryEntry,
  ReturnedFollowUpSummary,
} from '../agentOutput'
import {
  buildAgentFollowUpTarget,
  buildAgentFeedbackSummary,
  agentFeedbackFilterLabel,
  agentFollowUpFeedbackLabel,
  matchesAgentFeedbackFilter,
} from '../agentOutput'
import { agentRoleLabel, agentStatusLabel } from '../utils'

type AgentFeedbackWorkspaceProps = {
  agents: StoryAgentOutput[]
  feedbackByRole: Partial<Record<string, AgentFollowUpFeedback>>
  filter: AgentFeedbackFilter
  closureNotice?: string | null
  onApplyBatchFeedback: (feedback: AgentFollowUpFeedback) => void
  onCloseReviewCycle: () => void
  onClearFeedback: () => void
  onClearFilteredFeedback: () => void
  onFilterChange: (filter: AgentFeedbackFilter) => void
  onInsertIntoSource: (agent: StoryAgentOutput) => void
  onOpenNextFollowUp: () => void
  onOpenFollowUpTarget: (agent: StoryAgentOutput) => void
  onOpenHistorySource: (entry: ReturnedFollowUpHistoryEntry) => void
  onRemoveHistoryEntry?: (entry: ReturnedFollowUpHistoryEntry) => void
  onSelectAgent: (agent: StoryAgentOutput) => void
  onSelectHistoryEntry: (entry: ReturnedFollowUpHistoryEntry) => void
  returnedFollowUpHistory?: ReturnedFollowUpHistoryEntry[]
  returnedFollowUpSummary?: ReturnedFollowUpSummary | null
  selectedRole?: string
}

type ReturnHistoryFeedbackFilter = 'all' | AgentFollowUpFeedback

const feedbackFilters: AgentFeedbackFilter[] = [
  'all',
  'adopted',
  'needs_follow_up',
  'unmarked',
]
type ReturnHistoryFilter = 'all' | 'Storyboard' | 'Assets / Graph'

export function AgentFeedbackWorkspace({
  agents,
  closureNotice,
  feedbackByRole,
  filter,
  onApplyBatchFeedback,
  onCloseReviewCycle,
  onClearFeedback,
  onClearFilteredFeedback,
  onFilterChange,
  onInsertIntoSource,
  onOpenNextFollowUp,
  onOpenFollowUpTarget,
  onOpenHistorySource,
  onRemoveHistoryEntry,
  onSelectAgent,
  onSelectHistoryEntry,
  returnedFollowUpHistory = [],
  returnedFollowUpSummary,
  selectedRole,
}: AgentFeedbackWorkspaceProps) {
  const counts = buildAgentFeedbackSummary(agents, feedbackByRole)
  const [historyFilter, setHistoryFilter] = useState<ReturnHistoryFilter>('all')
  const [historyFeedbackFilter, setHistoryFeedbackFilter] = useState<ReturnHistoryFeedbackFilter>('all')
  const [historyExpanded, setHistoryExpanded] = useState(false)
  const filteredAgents = agents
    .filter((agent) => matchesAgentFeedbackFilter(agent.role, feedbackByRole, filter))
    .sort((left, right) => {
      const leftPriority = feedbackPriority(feedbackByRole[left.role])
      const rightPriority = feedbackPriority(feedbackByRole[right.role])
      if (leftPriority !== rightPriority) {
        return leftPriority - rightPriority
      }
      return agentRoleLabel(left.role).localeCompare(agentRoleLabel(right.role), 'zh-Hans-CN')
    })
  const hasFilteredFeedback = filteredAgents.some((agent) => feedbackByRole[agent.role] !== undefined)
  const hasNeedsFollowUp = filteredAgents.some(
    (agent) => feedbackByRole[agent.role] === 'needs_follow_up',
  )
  const reviewedCount = counts.adopted + counts.needs_follow_up
  const reviewProgress = counts.total === 0 ? 0 : Math.round((reviewedCount / counts.total) * 100)
  const canCloseReviewCycle =
    counts.total > 0 && counts.needs_follow_up === 0 && counts.unmarked === 0
  const filteredReturnHistory = returnedFollowUpHistory.filter((entry) => {
    const matchesSource = historyFilter === 'all' || entry.sourcePage === historyFilter
    const matchesFeedback =
      historyFeedbackFilter === 'all' || entry.feedback === historyFeedbackFilter
    return matchesSource && matchesFeedback
  })
  const visibleReturnHistory = historyExpanded
    ? filteredReturnHistory
    : filteredReturnHistory.slice(0, 3)
  const hiddenReturnHistoryCount = filteredReturnHistory.length - visibleReturnHistory.length

  return (
    <section className="surface-card agent-feedback-workspace" aria-labelledby="agent-feedback-workspace-title">
      <div className="panel-title-row">
        <div>
          <span className="section-kicker">Feedback workspace</span>
          <strong id="agent-feedback-workspace-title">Agent 反馈工作台</strong>
        </div>
        <small>
          已处理 {reviewedCount}/{counts.total} · 待跟进 {counts.needs_follow_up}
        </small>
      </div>
      <div className="agent-feedback-progress" aria-label={`反馈处理进度 ${reviewProgress}%`}>
        <div className="agent-feedback-progress-bar">
          <span
            className="agent-feedback-progress-fill"
            style={{ width: `${reviewProgress}%` }}
          />
        </div>
        <small>
          已采纳 {counts.adopted} · 未标记 {counts.unmarked} · 进度 {reviewProgress}%
        </small>
      </div>
      {canCloseReviewCycle ? (
        <div className="board-notice agent-feedback-empty">
          {returnedFollowUpSummary
            ? `最近回传（${returnedFollowUpSummary.sourcePage} · ${returnedFollowUpSummary.agentLabel} · ${agentFollowUpFeedbackLabel(returnedFollowUpSummary.feedback)}）已让本轮队列满足收口条件。`
            : returnedFollowUpHistory.length > 0
              ? `当前 review queue 已没有待跟进项，且本轮已累计回传 ${returnedFollowUpHistory.length} 条。`
              : '当前 review queue 已没有待跟进项，且所有 Agent 都已完成标记。'}
          <button type="button" className="ghost-action" onClick={onCloseReviewCycle}>
            收口本轮反馈
          </button>
        </div>
      ) : returnedFollowUpSummary ? (
        <div className="board-notice agent-feedback-empty">
          最近回传：{returnedFollowUpSummary.sourcePage} · {returnedFollowUpSummary.agentLabel} ·{' '}
          {agentFollowUpFeedbackLabel(returnedFollowUpSummary.feedback)}
          {returnedFollowUpSummary.resultNote ? ` · ${returnedFollowUpSummary.resultNote}` : ''}
        </div>
      ) : closureNotice ? (
        <div className="board-notice agent-feedback-empty">{closureNotice}</div>
      ) : null}
      {returnedFollowUpHistory.length > 0 ? (
        <div className="agent-feedback-history">
          <div className="panel-title-row">
            <div>
              <span className="section-kicker">Return history</span>
              <strong>最近回传记录</strong>
            </div>
            <small>最近 {returnedFollowUpHistory.length} 条</small>
          </div>
          <div className="asset-filter-row" role="toolbar" aria-label="回传历史筛选">
            {(['all', 'Storyboard', 'Assets / Graph'] as ReturnHistoryFilter[]).map((entry) => {
              const count =
                entry === 'all'
                  ? returnedFollowUpHistory.length
                  : returnedFollowUpHistory.filter((item) => item.sourcePage === entry).length
              return (
                <button
                  key={entry}
                  type="button"
                  aria-pressed={historyFilter === entry}
                  className={historyFilter === entry ? 'asset-filter-chip active' : 'asset-filter-chip'}
                  onClick={() => setHistoryFilter(entry)}
                >
                  {entry === 'all' ? '全部' : entry} {count}
                </button>
              )
            })}
          </div>
          <div className="asset-filter-row" role="toolbar" aria-label="回传反馈类型筛选">
            {(['all', 'adopted', 'needs_follow_up'] as ReturnHistoryFeedbackFilter[]).map((entry) => {
              const count =
                entry === 'all'
                  ? returnedFollowUpHistory.length
                  : returnedFollowUpHistory.filter((item) => item.feedback === entry).length
              const label =
                entry === 'all' ? '全部反馈' : agentFollowUpFeedbackLabel(entry)
              return (
                <button
                  key={entry}
                  type="button"
                  aria-pressed={historyFeedbackFilter === entry}
                  className={
                    historyFeedbackFilter === entry ? 'asset-filter-chip active' : 'asset-filter-chip'
                  }
                  onClick={() => setHistoryFeedbackFilter(entry)}
                >
                  {label} {count}
                </button>
              )
            })}
          </div>
          <div className="agent-feedback-history-list">
            {visibleReturnHistory.map((entry) => (
              <div className="agent-feedback-history-item" key={entry.id}>
                <strong>
                  {entry.sourcePage} · {entry.agentLabel}
                </strong>
                <span>{agentFollowUpFeedbackLabel(entry.feedback)}</span>
                <small>{entry.resultNote ?? '已从下游页回传结果。'}</small>
                <div className="agent-feedback-history-actions">
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={() => onSelectHistoryEntry(entry)}
                  >
                    打开对应 Agent
                  </button>
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={() => onOpenHistorySource(entry)}
                  >
                    回到来源页
                  </button>
                  {onRemoveHistoryEntry ? (
                    <button
                      type="button"
                      className="ghost-action"
                      onClick={() => onRemoveHistoryEntry(entry)}
                    >
                      移除此条
                    </button>
                  ) : null}
                  <small>{new Date(entry.createdAt).toLocaleString()}</small>
                </div>
              </div>
            ))}
            {visibleReturnHistory.length === 0 ? (
              <small>当前筛选下暂无回传记录。</small>
            ) : null}
          </div>
          {filteredReturnHistory.length > 3 ? (
            <button
              type="button"
              className="ghost-action"
              onClick={() => setHistoryExpanded((current) => !current)}
            >
              {historyExpanded
                ? '收起回传记录'
                : `展开全部回传记录（还有 ${hiddenReturnHistoryCount} 条）`}
            </button>
          ) : null}
        </div>
      ) : null}

      <div className="asset-filter-toolbar">
        <div className="asset-filter-row" role="toolbar" aria-label="Agent 反馈筛选">
          {feedbackFilters.map((entry) => {
            const count =
              entry === 'all'
                ? counts.total
                : entry === 'adopted'
                  ? counts.adopted
                  : entry === 'needs_follow_up'
                    ? counts.needs_follow_up
                    : counts.unmarked
            return (
              <button
                key={entry}
                type="button"
                aria-pressed={filter === entry}
                className={filter === entry ? 'asset-filter-chip active' : 'asset-filter-chip'}
                onClick={() => onFilterChange(entry)}
              >
                {agentFeedbackFilterLabel(entry)} {count}
              </button>
            )
          })}
        </div>
        {filteredAgents.length > 0 && (
          <div className="agent-feedback-batch-actions" aria-label="Agent 批量反馈动作">
            {hasNeedsFollowUp && (
              <button
                type="button"
                className="ghost-action"
                onClick={onOpenNextFollowUp}
              >
                处理下一条待跟进
              </button>
            )}
            <button
              type="button"
              className="ghost-action"
              onClick={() => onApplyBatchFeedback('adopted')}
            >
              当前筛选标为已采纳
            </button>
            <button
              type="button"
              className="ghost-action"
              onClick={() => onApplyBatchFeedback('needs_follow_up')}
            >
              当前筛选标为待跟进
            </button>
            {hasFilteredFeedback && filter !== 'unmarked' && (
              <button
                type="button"
                className="ghost-action"
                onClick={onClearFilteredFeedback}
              >
                {filter === 'all' ? '清空当前分析反馈' : '清空当前筛选反馈'}
              </button>
            )}
          </div>
        )}
        <div className="agent-feedback-summary">
          <ListFilter aria-hidden="true" size={14} />
          <span>
            当前筛选：{agentFeedbackFilterLabel(filter)}
            {filter === 'all'
              ? ' · 标记结果会记住到当前浏览器，刷新后仍可继续 follow-up。'
              : ` · 当前命中 ${filteredAgents.length} 个 Agent。`}
          </span>
          {(counts.adopted > 0 || counts.needs_follow_up > 0) && (
            <button type="button" className="ghost-action" onClick={onClearFeedback}>
              清空反馈
            </button>
          )}
        </div>
      </div>

      {filteredAgents.length === 0 ? (
        <div className="board-notice agent-feedback-empty">
          当前没有“{agentFeedbackFilterLabel(filter)}”的 Agent。先在输出面板里做反馈标记，工作台会自动刷新。
          {filter !== 'all' && (
            <button type="button" className="ghost-action" onClick={() => onFilterChange('all')}>
              返回全部
            </button>
          )}
        </div>
      ) : (
        <div className="agent-feedback-list">
          {filteredAgents.map((agent) => {
            const followUpTarget = buildAgentFollowUpTarget(agent)
            return (
              <article
                className={
                  selectedRole === agent.role
                    ? 'agent-feedback-card active'
                    : 'agent-feedback-card'
                }
                key={agent.role}
              >
                <button
                  type="button"
                  className="agent-feedback-card-main"
                  onClick={() => onSelectAgent(agent)}
                >
                  <div className="agent-feedback-card-head">
                    <div>
                      <strong>{agentRoleLabel(agent.role)}</strong>
                      <span>{agentStatusLabel(agent.status)}</span>
                    </div>
                    <span
                      className={
                        feedbackByRole[agent.role] === 'adopted'
                          ? 'asset-status-chip ready'
                          : feedbackByRole[agent.role] === 'needs_follow_up'
                            ? 'asset-status-chip draft'
                            : 'asset-status-chip draft'
                      }
                    >
                      {agentFollowUpFeedbackLabel(feedbackByRole[agent.role])}
                    </span>
                  </div>
                  <p className="agent-feedback-copy">
                    {agent.highlights[0] ||
                      agent.output.slice(0, 96) ||
                      '打开输出面板后，可继续复制摘要、回填和跳转下游工作台。'}
                  </p>
                  <div className="agent-highlights">
                    {agent.highlights.slice(0, 2).map((highlight) => (
                      <span key={highlight} className="agent-highlight-tag">
                        {highlight}
                      </span>
                    ))}
                  </div>
                  <span className="agent-feedback-card-meta">
                    {selectedRole === agent.role ? '当前打开' : '查看输出'} · 重点 {agent.highlights.length}
                  </span>
                </button>
                <div className="agent-feedback-card-actions">
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={() => onInsertIntoSource(agent)}
                  >
                    回填摘要
                  </button>
                  {followUpTarget && (
                    <button
                      type="button"
                      className="ghost-action"
                      onClick={() => onOpenFollowUpTarget(agent)}
                    >
                      {followUpTarget.actionLabel}
                    </button>
                  )}
                </div>
              </article>
            )
          })}
        </div>
      )}
    </section>
  )
}

function feedbackPriority(feedback: AgentFollowUpFeedback | undefined): number {
  if (feedback === 'needs_follow_up') return 0
  if (feedback === undefined) return 1
  return 2
}
