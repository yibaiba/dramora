import { useMemo, useState } from 'react'

export type RecoveryPanelEvent = {
  status: string
  message: string
  created_at: string
}

export type RecoveryPanelProps = {
  title: string
  subtitle?: string
  isLoading?: boolean
  isError?: boolean
  status?: string
  isTerminal?: boolean
  isRecoverable?: boolean
  statusEnteredAt?: string
  lastEventAt?: string
  totalEventCount?: number
  sameStatusCount?: number
  nextHint?: string
  events?: RecoveryPanelEvent[]
  emptyHint?: string
}

const MAX_VISIBLE_EVENTS = 4

export function RecoveryPanel({
  title,
  subtitle,
  isLoading,
  isError,
  status,
  isTerminal,
  isRecoverable,
  statusEnteredAt,
  lastEventAt,
  totalEventCount,
  sameStatusCount,
  nextHint,
  events,
  emptyHint = '暂无可显示的恢复轨迹',
}: RecoveryPanelProps) {
  const [expanded, setExpanded] = useState(false)
  const lifecycleLabel = useMemo(() => {
    if (isTerminal) return '已进入终态'
    if (isRecoverable) return '在线 / 可恢复'
    return '其他状态'
  }, [isTerminal, isRecoverable])

  const visibleEvents = useMemo(() => {
    if (!events || events.length === 0) return []
    if (expanded) return events
    return events.slice(0, MAX_VISIBLE_EVENTS)
  }, [events, expanded])

  const hasData = Boolean(status || statusEnteredAt || (events && events.length > 0))

  return (
    <section className="inspector-card recovery-card" aria-label={title}>
      <header>
        <strong>{title}</strong>
        {subtitle ? <span className="recovery-card-subtle">{subtitle}</span> : null}
      </header>
      {isLoading ? <p className="recovery-card-subtle">读取恢复轨迹…</p> : null}
      {isError ? <p className="recovery-card-subtle">暂时无法读取恢复轨迹</p> : null}
      {!isLoading && !isError && !hasData ? (
        <p className="recovery-card-subtle">{emptyHint}</p>
      ) : null}
      {hasData ? (
        <div className="recovery-card-body">
          <div className="recovery-card-row">
            {status ? (
              <span className={`recovery-badge recovery-badge-${status}`}>{status}</span>
            ) : null}
            <span className="recovery-card-subtle">{lifecycleLabel}</span>
          </div>
          <dl className="recovery-card-meta">
            <div>
              <dt>当前状态停留</dt>
              <dd>{relativeFromNow(statusEnteredAt)}</dd>
            </div>
            <div>
              <dt>最近事件</dt>
              <dd>{relativeFromNow(lastEventAt)}</dd>
            </div>
            <div>
              <dt>事件总数</dt>
              <dd>
                {typeof totalEventCount === 'number' ? totalEventCount : '—'}
                {typeof sameStatusCount === 'number' ? (
                  <span className="recovery-card-subtle"> · 同状态 {sameStatusCount} 次</span>
                ) : null}
              </dd>
            </div>
            <div>
              <dt>生命周期</dt>
              <dd>{lifecycleLabel}</dd>
            </div>
          </dl>
          {nextHint ? <p className="recovery-card-hint">下一步：{nextHint}</p> : null}
          {events && events.length > 0 ? (
            <ul className="recovery-card-events">
              {visibleEvents.map((event, index) => (
                <li key={`${event.created_at}-${index}`}>
                  <span className={`recovery-badge recovery-badge-${event.status}`}>
                    {event.status}
                  </span>
                  <span className="recovery-card-subtle">{relativeFromNow(event.created_at)}</span>
                  {event.message ? <p>{event.message}</p> : null}
                </li>
              ))}
              {events.length > MAX_VISIBLE_EVENTS ? (
                <li className="recovery-card-events-toggle">
                  <button type="button" onClick={() => setExpanded((value) => !value)}>
                    {expanded ? '收起事件' : `查看全部 ${events.length} 条事件`}
                  </button>
                </li>
              ) : null}
            </ul>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

function relativeFromNow(iso?: string): string {
  if (!iso) return '—'
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return '—'
  const diffMs = Date.now() - t
  const minutes = Math.floor(diffMs / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  return `${Math.floor(hours / 24)} 天前`
}
