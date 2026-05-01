import { useMemo, useState } from 'react'
import { KeyRound, LogOut, RefreshCcw, ShieldCheck } from 'lucide-react'
import { useRevokeSession, useSessions } from '../../api/hooks'
import type { Session } from '../../api/types'
import { useAuthStore } from '../../state/authStore'

type SessionStatus = 'current' | 'active' | 'revoked' | 'expired'

function classifySession(session: Session, currentSessionId?: string): SessionStatus {
  if (session.revoked_at) return 'revoked'
  const expires = Date.parse(session.expires_at)
  if (!Number.isNaN(expires) && expires <= Date.now()) return 'expired'
  if (currentSessionId && session.id === currentSessionId) return 'current'
  return 'active'
}

function formatDateTime(iso?: string | null): string {
  if (!iso) return '—'
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return iso
  return new Date(ts).toLocaleString()
}

function statusLabel(status: SessionStatus): string {
  switch (status) {
    case 'current':
      return '本设备 · 活跃'
    case 'active':
      return '活跃'
    case 'revoked':
      return '已吊销'
    case 'expired':
      return '已过期'
  }
}

export function SessionsPage() {
  const session = useAuthStore((state) => state.session)
  const clearSession = useAuthStore((state) => state.clearSession)
  const sessionsQuery = useSessions(Boolean(session?.token))
  const revokeMutation = useRevokeSession()
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [bulkRunning, setBulkRunning] = useState(false)

  const currentSessionId = session?.current_session_id
  const sessions = sessionsQuery.data ?? []

  const decorated = useMemo(
    () =>
      sessions.map((row) => ({
        row,
        status: classifySession(row, currentSessionId),
      })),
    [sessions, currentSessionId],
  )

  const otherActive = useMemo(
    () => decorated.filter((entry) => entry.status === 'active'),
    [decorated],
  )

  const handleRevoke = async (sessionId: string, isCurrent: boolean) => {
    setError(null)
    setPendingId(sessionId)
    try {
      await revokeMutation.mutateAsync(sessionId)
      if (isCurrent) {
        clearSession()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '吊销失败')
    } finally {
      setPendingId(null)
    }
  }

  const handleRevokeAllOthers = async () => {
    if (otherActive.length === 0 || bulkRunning) return
    setError(null)
    setBulkRunning(true)
    try {
      for (const entry of otherActive) {
        try {
          await revokeMutation.mutateAsync(entry.row.id)
        } catch (err) {
          setError(err instanceof Error ? err.message : '部分会话吊销失败')
        }
      }
    } finally {
      setBulkRunning(false)
    }
  }

  return (
    <div className="sessions-page">
      <header className="page-header">
        <KeyRound size={20} aria-hidden="true" />
        <h1>登录会话</h1>
        <p className="page-subtitle">
          查看当前账号的所有 refresh-token 会话，必要时可吊销可疑或不再使用的设备。
        </p>
      </header>

      <section className="sessions-toolbar">
        <button
          type="button"
          className="btn-secondary"
          onClick={() => sessionsQuery.refetch()}
          disabled={sessionsQuery.isFetching}
        >
          <RefreshCcw size={14} aria-hidden="true" /> 刷新
        </button>
        <button
          type="button"
          className="btn-warning"
          onClick={handleRevokeAllOthers}
          disabled={bulkRunning || otherActive.length === 0}
          title="一次吊销除当前设备外的所有活跃会话"
        >
          <LogOut size={14} aria-hidden="true" />
          {bulkRunning ? '正在吊销...' : `吊销其他设备（${otherActive.length}）`}
        </button>
      </section>

      {error ? (
        <div className="form-error" role="alert">
          {error}
        </div>
      ) : null}

      {sessionsQuery.isLoading ? (
        <p className="muted">正在加载会话...</p>
      ) : sessions.length === 0 ? (
        <div className="empty-state">
          <ShieldCheck size={20} aria-hidden="true" />
          <strong>当前没有活跃会话</strong>
          <span>登录后才会在这里看到记录。</span>
        </div>
      ) : (
        <ul className="session-list">
          {decorated.map(({ row, status }) => {
            const isCurrent = status === 'current'
            const canRevoke = status === 'current' || status === 'active'
            return (
              <li key={row.id} className={`session-row session-row-${status}`}>
                <div className="session-meta">
                  <strong className="session-id">{row.id}</strong>
                  <span className={`session-status session-status-${status}`}>
                    {statusLabel(status)}
                  </span>
                  <span className="session-meta-line">
                    组织 <code>{row.organization_id}</code> · 角色 <code>{row.role}</code>
                  </span>
                  <span className="session-meta-line">
                    创建于 {formatDateTime(row.created_at)} · 到期 {formatDateTime(row.expires_at)}
                    {row.revoked_at ? ` · 已于 ${formatDateTime(row.revoked_at)} 吊销` : ''}
                  </span>
                </div>
                <div className="session-actions">
                  <button
                    type="button"
                    className="btn-warning"
                    disabled={!canRevoke || pendingId === row.id || revokeMutation.isPending}
                    onClick={() => handleRevoke(row.id, isCurrent)}
                  >
                    {pendingId === row.id
                      ? '正在吊销...'
                      : isCurrent
                        ? '吊销并退出'
                        : '吊销此会话'}
                  </button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
