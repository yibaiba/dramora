import { useEffect, useRef, useState } from 'react'
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
import {
  RETURN_HISTORY_INITIAL_PAGE_SIZE,
  RETURN_HISTORY_PAGE_INCREMENT,
  sanitizeReturnedFollowUpHistory,
} from '../reviewPersistence'
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
  onImportHistoryEntries?: (entries: ReturnedFollowUpHistoryEntry[]) => number
  onRemoveHistoryEntry?: (entry: ReturnedFollowUpHistoryEntry) => void
  onRemoveHistoryEntries?: (entries: ReturnedFollowUpHistoryEntry[]) => void
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
  onImportHistoryEntries,
  onRemoveHistoryEntry,
  onRemoveHistoryEntries,
  onSelectAgent,
  onSelectHistoryEntry,
  returnedFollowUpHistory = [],
  returnedFollowUpSummary,
  selectedRole,
}: AgentFeedbackWorkspaceProps) {
  const counts = buildAgentFeedbackSummary(agents, feedbackByRole)
  const [historyFilter, setHistoryFilter] = useState<ReturnHistoryFilter>('all')
  const [historyFeedbackFilter, setHistoryFeedbackFilter] = useState<ReturnHistoryFeedbackFilter>('all')
  const [historyPageSize, setHistoryPageSize] = useState(RETURN_HISTORY_INITIAL_PAGE_SIZE)
  const [historySearch, setHistorySearch] = useState('')
  const [historySortOrder, setHistorySortOrder] = useState<'desc' | 'asc'>('desc')
  const [historyTimeRange, setHistoryTimeRange] = useState<'all' | '24h' | '7d' | '30d'>('all')
  const [historyRowCopyId, setHistoryRowCopyId] = useState<string | null>(null)
  const [historyCopyState, setHistoryCopyState] = useState<'idle' | 'copied' | 'error'>('idle')
  const [historyImportState, setHistoryImportState] = useState<
    | { kind: 'idle' }
    | { kind: 'imported'; count: number }
    | { kind: 'error'; message: string }
  >({ kind: 'idle' })
  const historyImportInputRef = useRef<HTMLInputElement | null>(null)
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
  const [historyNowMs, setHistoryNowMs] = useState<number>(() => Date.now())
  useEffect(() => {
    if (historyTimeRange === 'all') {
      return
    }
    const interval = window.setInterval(() => setHistoryNowMs(Date.now()), 60_000)
    return () => window.clearInterval(interval)
  }, [historyTimeRange])
  const filteredReturnHistory = returnedFollowUpHistory
    .filter((entry) => {
      const matchesSource = historyFilter === 'all' || entry.sourcePage === historyFilter
      const matchesFeedback =
        historyFeedbackFilter === 'all' || entry.feedback === historyFeedbackFilter
      const trimmedSearch = historySearch.trim().toLowerCase()
      const matchesSearch =
        trimmedSearch.length === 0 ||
        entry.agentLabel.toLowerCase().includes(trimmedSearch) ||
        entry.sourcePage.toLowerCase().includes(trimmedSearch) ||
        (entry.resultNote ?? '').toLowerCase().includes(trimmedSearch)
      let matchesTime = true
      if (historyTimeRange !== 'all') {
        const windowMs =
          historyTimeRange === '24h'
            ? 24 * 60 * 60 * 1000
            : historyTimeRange === '7d'
              ? 7 * 24 * 60 * 60 * 1000
              : 30 * 24 * 60 * 60 * 1000
        const entryTime = new Date(entry.createdAt).getTime()
        matchesTime = Number.isFinite(entryTime) && historyNowMs - entryTime <= windowMs
      }
      return matchesSource && matchesFeedback && matchesSearch && matchesTime
    })
    .slice()
    .sort((left, right) => {
      const leftTime = new Date(left.createdAt).getTime()
      const rightTime = new Date(right.createdAt).getTime()
      return historySortOrder === 'desc' ? rightTime - leftTime : leftTime - rightTime
    })
  const visibleReturnHistory = filteredReturnHistory.slice(0, historyPageSize)
  const hiddenReturnHistoryCount = filteredReturnHistory.length - visibleReturnHistory.length

  const historyFilterSignature = `${historyFilter}|${historyFeedbackFilter}|${historySearch}|${historySortOrder}|${historyTimeRange}`
  const [previousFilterSignature, setPreviousFilterSignature] = useState(historyFilterSignature)
  if (previousFilterSignature !== historyFilterSignature) {
    setPreviousFilterSignature(historyFilterSignature)
    setHistoryPageSize(RETURN_HISTORY_INITIAL_PAGE_SIZE)
  }

  const [historyCursor, setHistoryCursor] = useState<string | null>(null)
  const validCursor = historyCursor && visibleReturnHistory.some((entry) => entry.id === historyCursor)
  if (historyCursor && !validCursor) {
    setHistoryCursor(visibleReturnHistory[0]?.id ?? null)
  }
  const historyListRef = useRef<HTMLDivElement | null>(null)
  const scrollHistoryItemIntoView = (id: string) => {
    const root = historyListRef.current
    if (!root) return
    const node = root.querySelector<HTMLElement>(`[data-history-id="${id}"]`)
    node?.scrollIntoView({ block: 'nearest' })
  }
  useEffect(() => {
    if (visibleReturnHistory.length === 0) return
    const handler = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null
      if (target) {
        const tag = target.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable) {
          return
        }
      }
      if (event.metaKey || event.ctrlKey || event.altKey) return
      const key = event.key.toLowerCase()
      if (key !== 'j' && key !== 'k' && key !== 'enter' && key !== 'o') return
      const ids = visibleReturnHistory.map((entry) => entry.id)
      const currentIndex = historyCursor ? ids.indexOf(historyCursor) : -1
      if (key === 'j') {
        event.preventDefault()
        const next = currentIndex < 0 ? 0 : Math.min(currentIndex + 1, ids.length - 1)
        const nextId = ids[next]
        setHistoryCursor(nextId)
        scrollHistoryItemIntoView(nextId)
      } else if (key === 'k') {
        event.preventDefault()
        const prev = currentIndex < 0 ? 0 : Math.max(currentIndex - 1, 0)
        const prevId = ids[prev]
        setHistoryCursor(prevId)
        scrollHistoryItemIntoView(prevId)
      } else if (key === 'enter' && currentIndex >= 0) {
        event.preventDefault()
        onSelectHistoryEntry(visibleReturnHistory[currentIndex])
      } else if (key === 'o' && currentIndex >= 0) {
        event.preventDefault()
        onOpenHistorySource(visibleReturnHistory[currentIndex])
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [historyCursor, visibleReturnHistory, onSelectHistoryEntry, onOpenHistorySource])

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
      {returnedFollowUpHistory.length === 0 && onImportHistoryEntries ? (
        <div className="agent-feedback-history-toolbar">
          <small>无回传历史。可从他处导入：</small>
          <input
            ref={historyImportInputRef}
            type="file"
            accept="application/json,.json"
            hidden
            onChange={(event) => {
              const file = event.target.files?.[0]
              event.target.value = ''
              if (!file) return
              file
                .text()
                .then((raw) => {
                  let parsed: unknown
                  try {
                    parsed = JSON.parse(raw)
                  } catch {
                    throw new Error('JSON 格式无法解析')
                  }
                  const sanitized = sanitizeReturnedFollowUpHistory(parsed)
                  if (sanitized.length === 0) {
                    throw new Error('JSON 中没有可识别的回传记录')
                  }
                  const imported = onImportHistoryEntries(sanitized)
                  setHistoryImportState({ kind: 'imported', count: imported })
                })
                .catch((err: unknown) => {
                  const message = err instanceof Error ? err.message : '导入失败'
                  setHistoryImportState({ kind: 'error', message })
                })
                .finally(() => {
                  window.setTimeout(() => setHistoryImportState({ kind: 'idle' }), 2500)
                })
            }}
          />
          <button
            type="button"
            className="ghost-action"
            onClick={() => historyImportInputRef.current?.click()}
          >
            {historyImportState.kind === 'imported'
              ? `已导入 ${historyImportState.count} 条`
              : historyImportState.kind === 'error'
                ? `导入失败：${historyImportState.message}`
                : '导入 JSON'}
          </button>
        </div>
      ) : null}
      {returnedFollowUpHistory.length > 0 ? (
        <div className="agent-feedback-history">
          <div className="panel-title-row">
            <div>
              <span className="section-kicker">Return history</span>
              <strong>最近回传记录</strong>
            </div>
            <small>
              最近 {returnedFollowUpHistory.length} 条 · j/k 选择 · Enter 打开 Agent · O 回到来源
            </small>
          </div>
          <div className="agent-feedback-history-toolbar">
            <input
              type="search"
              className="agent-feedback-history-search"
              value={historySearch}
              onChange={(event) => setHistorySearch(event.target.value)}
              placeholder="搜索 agent / 来源 / 备注"
              aria-label="搜索回传历史"
            />
            <button
              type="button"
              className="ghost-action"
              onClick={() => setHistorySortOrder((prev) => (prev === 'desc' ? 'asc' : 'desc'))}
              title={historySortOrder === 'desc' ? '当前：新→旧，点击切换为旧→新' : '当前：旧→新，点击切换为新→旧'}
            >
              排序：{historySortOrder === 'desc' ? '新→旧' : '旧→新'}
            </button>
            <div className="agent-feedback-history-time-range" role="group" aria-label="时间范围">
              {(['all', '24h', '7d', '30d'] as const).map((range) => (
                <button
                  key={range}
                  type="button"
                  aria-pressed={historyTimeRange === range}
                  className={historyTimeRange === range ? 'asset-filter-chip active' : 'asset-filter-chip'}
                  onClick={() => setHistoryTimeRange(range)}
                  title={
                    range === 'all'
                      ? '全部时间'
                      : range === '24h'
                        ? '最近 24 小时'
                        : range === '7d'
                          ? '最近 7 天'
                          : '最近 30 天'
                  }
                >
                  {range === 'all' ? '全部' : range === '24h' ? '24h' : range === '7d' ? '7d' : '30d'}
                </button>
              ))}
            </div>
            {filteredReturnHistory.length > 0 ? (
              <button
                type="button"
                className="ghost-action"
                onClick={() => {
                  try {
                    const header = ['id', 'createdAt', 'agentLabel', 'agentRole', 'sourcePage', 'feedback', 'resultNote']
                    const escape = (value: unknown) => {
                      const str = value === undefined || value === null ? '' : String(value)
                      if (/[",\n]/.test(str)) {
                        return `"${str.replace(/"/g, '""')}"`
                      }
                      return str
                    }
                    const rows = [header.join(',')]
                    for (const entry of filteredReturnHistory) {
                      rows.push(
                        [
                          entry.id,
                          entry.createdAt,
                          entry.agentLabel,
                          entry.agentRole,
                          entry.sourcePage,
                          entry.feedback ?? '',
                          entry.resultNote ?? '',
                        ]
                          .map(escape)
                          .join(','),
                      )
                    }
                    const blob = new Blob(['\uFEFF' + rows.join('\n')], { type: 'text/csv;charset=utf-8' })
                    const url = URL.createObjectURL(blob)
                    const anchor = document.createElement('a')
                    anchor.href = url
                    const stamp = new Date().toISOString().replace(/[:.]/g, '-')
                    anchor.download = `return-history-${stamp}.csv`
                    document.body.appendChild(anchor)
                    anchor.click()
                    document.body.removeChild(anchor)
                    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
                  } catch {
                    /* swallow: download failure is non-fatal */
                  }
                }}
                title="按当前筛选导出为 CSV"
              >
                导出 CSV（{filteredReturnHistory.length}）
              </button>
            ) : null}
            {filteredReturnHistory.length > 0 && onRemoveHistoryEntries ? (
              <button
                type="button"
                className="ghost-action"
                onClick={() => {
                  onRemoveHistoryEntries(filteredReturnHistory)
                  setHistoryCopyState('idle')
                }}
              >
                批量移除（{filteredReturnHistory.length}）
              </button>
            ) : null}
            {filteredReturnHistory.length > 0 ? (
              <button
                type="button"
                className="ghost-action"
                onClick={() => {
                  try {
                    const payload = JSON.stringify(filteredReturnHistory, null, 2)
                    const blob = new Blob([payload], { type: 'application/json' })
                    const url = URL.createObjectURL(blob)
                    const anchor = document.createElement('a')
                    anchor.href = url
                    const stamp = new Date().toISOString().replace(/[:.]/g, '-')
                    anchor.download = `return-history-${stamp}.json`
                    document.body.appendChild(anchor)
                    anchor.click()
                    document.body.removeChild(anchor)
                    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
                  } catch {
                    /* swallow: download failure is non-fatal */
                  }
                }}
              >
                下载 JSON（{filteredReturnHistory.length}）
              </button>
            ) : null}
            {filteredReturnHistory.length > 0 ? (
              <button
                type="button"
                className="ghost-action"
                onClick={async () => {
                  try {
                    const payload = JSON.stringify(filteredReturnHistory, null, 2)
                    if (typeof navigator !== 'undefined' && navigator.clipboard) {
                      await navigator.clipboard.writeText(payload)
                      setHistoryCopyState('copied')
                    } else {
                      setHistoryCopyState('error')
                    }
                  } catch {
                    setHistoryCopyState('error')
                  }
                  window.setTimeout(() => setHistoryCopyState('idle'), 1500)
                }}
              >
                {historyCopyState === 'copied'
                  ? '已复制 JSON'
                  : historyCopyState === 'error'
                    ? '复制失败'
                    : `复制为 JSON（${filteredReturnHistory.length}）`}
              </button>
            ) : null}
            {onImportHistoryEntries ? (
              <>
                <input
                  ref={historyImportInputRef}
                  type="file"
                  accept="application/json,.json"
                  hidden
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    event.target.value = ''
                    if (!file) return
                    file
                      .text()
                      .then((raw) => {
                        let parsed: unknown
                        try {
                          parsed = JSON.parse(raw)
                        } catch {
                          throw new Error('JSON 格式无法解析')
                        }
                        const sanitized = sanitizeReturnedFollowUpHistory(parsed)
                        if (sanitized.length === 0) {
                          throw new Error('JSON 中没有可识别的回传记录')
                        }
                        const imported = onImportHistoryEntries(sanitized)
                        setHistoryImportState({ kind: 'imported', count: imported })
                      })
                      .catch((err: unknown) => {
                        const message = err instanceof Error ? err.message : '导入失败'
                        setHistoryImportState({ kind: 'error', message })
                      })
                      .finally(() => {
                        window.setTimeout(() => setHistoryImportState({ kind: 'idle' }), 2500)
                      })
                  }}
                />
                <button
                  type="button"
                  className="ghost-action"
                  onClick={() => historyImportInputRef.current?.click()}
                >
                  {historyImportState.kind === 'imported'
                    ? `已导入 ${historyImportState.count} 条`
                    : historyImportState.kind === 'error'
                      ? `导入失败：${historyImportState.message}`
                      : '导入 JSON'}
                </button>
              </>
            ) : null}
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
          <div className="agent-feedback-history-list" ref={historyListRef}>
            {(() => {
              const groups: { key: string; label: string; items: typeof visibleReturnHistory }[] = []
              const today = new Date()
              const todayKey = today.toDateString()
              const yesterdayKey = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1).toDateString()
              for (const item of visibleReturnHistory) {
                const created = new Date(item.createdAt)
                const dayKey = created.toDateString()
                let label = created.toLocaleDateString()
                if (dayKey === todayKey) label = `今天 · ${label}`
                else if (dayKey === yesterdayKey) label = `昨天 · ${label}`
                const last = groups[groups.length - 1]
                if (last && last.key === dayKey) {
                  last.items.push(item)
                } else {
                  groups.push({ key: dayKey, label, items: [item] })
                }
              }
              return groups.map((group) => (
                <div className="agent-feedback-history-group" key={group.key}>
                  <div className="agent-feedback-history-group-heading">
                    <span>{group.label}</span>
                    <small>{group.items.length} 条</small>
                  </div>
                  {group.items.map((entry) => (
                    <div
                      className={
                        entry.id === historyCursor
                          ? 'agent-feedback-history-item is-cursor'
                          : 'agent-feedback-history-item'
                      }
                      key={entry.id}
                      data-history-id={entry.id}
                      onMouseEnter={() => setHistoryCursor(entry.id)}
                    >
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
                        <button
                          type="button"
                          className="ghost-action"
                          onClick={async () => {
                            try {
                              const payload = JSON.stringify(entry, null, 2)
                              if (typeof navigator !== 'undefined' && navigator.clipboard) {
                                await navigator.clipboard.writeText(payload)
                                setHistoryRowCopyId(entry.id)
                                window.setTimeout(() => {
                                  setHistoryRowCopyId((current) => (current === entry.id ? null : current))
                                }, 1500)
                              }
                            } catch {
                              /* swallow */
                            }
                          }}
                          title="复制此条记录的 JSON"
                        >
                          {historyRowCopyId === entry.id ? '已复制 ✓' : '复制此条'}
                        </button>
                        <small>{new Date(entry.createdAt).toLocaleTimeString()}</small>
                      </div>
                    </div>
                  ))}
                </div>
              ))
            })()}
            {visibleReturnHistory.length === 0 ? (
              <small>当前筛选下暂无回传记录。</small>
            ) : null}
          </div>
          {filteredReturnHistory.length > RETURN_HISTORY_INITIAL_PAGE_SIZE ? (
            <div className="agent-feedback-history-pagination">
              {hiddenReturnHistoryCount > 0 ? (
                <button
                  type="button"
                  className="ghost-action"
                  onClick={() =>
                    setHistoryPageSize((current) =>
                      Math.min(current + RETURN_HISTORY_PAGE_INCREMENT, filteredReturnHistory.length),
                    )
                  }
                >
                  加载更多（剩余 {hiddenReturnHistoryCount}，每次 +{RETURN_HISTORY_PAGE_INCREMENT}）
                </button>
              ) : null}
              {hiddenReturnHistoryCount > 0 && filteredReturnHistory.length > historyPageSize ? (
                <button
                  type="button"
                  className="ghost-action"
                  onClick={() => setHistoryPageSize(filteredReturnHistory.length)}
                >
                  展开全部（{filteredReturnHistory.length}）
                </button>
              ) : null}
              {historyPageSize > RETURN_HISTORY_INITIAL_PAGE_SIZE ? (
                <button
                  type="button"
                  className="ghost-action"
                  onClick={() => setHistoryPageSize(RETURN_HISTORY_INITIAL_PAGE_SIZE)}
                >
                  收起回传记录
                </button>
              ) : null}
            </div>
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
