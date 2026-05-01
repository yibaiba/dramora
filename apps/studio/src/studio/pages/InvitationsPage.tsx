import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Mail, Plus, ShieldCheck, Copy, Check, Search, XCircle, RefreshCw } from 'lucide-react'
import { useCreateInvitation, useOrganizationInvitations, useRevokeInvitation, useResendInvitation, useInvitationAuditEvents } from '../../api/hooks'
import { useAuthStore } from '../../state/authStore'
import type { OrganizationInvitation } from '../../api/types'
import { StatePlaceholder } from '../components/StatePlaceholder'

const ADMIN_ROLES = new Set(['owner', 'admin'])

const ROLE_OPTIONS: Array<{ value: NonNullable<OrganizationInvitation['role']>; label: string }> = [
  { value: 'editor', label: 'Editor · 可编辑生产' },
  { value: 'viewer', label: 'Viewer · 只读观察' },
  { value: 'admin', label: 'Admin · 可配置端点' },
  { value: 'owner', label: 'Owner · 全权所有者' },
]

type StatusFilter = 'all' | 'pending' | 'accepted' | 'expired' | 'revoked'

const STATUS_FILTERS: Array<{ value: StatusFilter; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'pending', label: '待接受' },
  { value: 'accepted', label: '已接受' },
  { value: 'expired', label: '已失效' },
  { value: 'revoked', label: '已吊销' },
]

function isExpired(invitation: OrganizationInvitation): boolean {
  if (invitation.status !== 'pending') return false
  const expiresAt = Date.parse(invitation.expires_at)
  return Number.isFinite(expiresAt) && expiresAt < Date.now()
}

function buildInviteUrl(token: string): string {
  if (typeof window === 'undefined') return token
  return `${window.location.origin}${window.location.pathname}?invite=${encodeURIComponent(token)}`
}

export function InvitationsPage() {
  const session = useAuthStore((state) => state.session)
  const isAdmin = Boolean(session && ADMIN_ROLES.has(session.role))

  const invitationsQuery = useOrganizationInvitations(isAdmin)
  const createMutation = useCreateInvitation()
  const revokeMutation = useRevokeInvitation()
  const resendMutation = useResendInvitation()
  const auditQuery = useInvitationAuditEvents(isAdmin, 50)

  const [email, setEmail] = useState('')
  const [role, setRole] = useState<OrganizationInvitation['role']>('editor')
  const [errorMessage, setErrorMessage] = useState('')
  const [copiedToken, setCopiedToken] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [searchInput, setSearchInput] = useState('')
  const [revokeError, setRevokeError] = useState('')
  const [revokingId, setRevokingId] = useState<string | null>(null)
  const [resendError, setResendError] = useState('')
  const [resendingId, setResendingId] = useState<string | null>(null)

  const allInvitations = useMemo(
    () => (invitationsQuery.data ?? []).slice().sort((a, b) => b.created_at.localeCompare(a.created_at)),
    [invitationsQuery.data],
  )

  const counts = useMemo(() => {
    const pending = allInvitations.filter((inv) => inv.status === 'pending' && !isExpired(inv)).length
    const accepted = allInvitations.filter((inv) => inv.status === 'accepted').length
    const expired = allInvitations.filter((inv) => isExpired(inv)).length
    const revoked = allInvitations.filter((inv) => inv.status === 'revoked').length
    return { all: allInvitations.length, pending, accepted, expired, revoked }
  }, [allInvitations])

  const filteredInvitations = useMemo(() => {
    const search = searchInput.trim().toLowerCase()
    return allInvitations.filter((invitation) => {
      const expired = isExpired(invitation)
      let effectiveStatus: StatusFilter
      if (invitation.status === 'accepted') effectiveStatus = 'accepted'
      else if (invitation.status === 'revoked') effectiveStatus = 'revoked'
      else if (expired) effectiveStatus = 'expired'
      else effectiveStatus = 'pending'
      if (statusFilter !== 'all' && effectiveStatus !== statusFilter) {
        return false
      }
      if (search && !invitation.email.toLowerCase().includes(search)) {
        return false
      }
      return true
    })
  }, [allInvitations, statusFilter, searchInput])

  if (!isAdmin) {
    return (
      <div className="admin-settings-page">
        <header className="page-header">
          <ShieldCheck size={20} aria-hidden="true" />
          <h1>组织邀请</h1>
          <p className="page-subtitle">仅 owner / admin 可发起组织邀请。当前角色：{session?.role ?? '未知'}</p>
        </header>
      </div>
    )
  }

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setErrorMessage('')
    createMutation.mutate(
      { email, role },
      {
        onError: (error) => setErrorMessage(error.message),
        onSuccess: () => setEmail(''),
      },
    )
  }

  async function handleCopy(invitation: OrganizationInvitation) {
    const url = buildInviteUrl(invitation.token)
    try {
      await navigator.clipboard.writeText(url)
      setCopiedToken(invitation.id)
      window.setTimeout(() => setCopiedToken((current) => (current === invitation.id ? null : current)), 2000)
    } catch {
      // best-effort copy; clipboard may be unavailable in some sandboxes
    }
  }

  async function handleRevoke(invitation: OrganizationInvitation) {
    if (revokingId || resendingId) return
    if (typeof window !== 'undefined') {
      const ok = window.confirm(`确认吊销发往 ${invitation.email} 的邀请？吊销后该 token 将不可再用于注册。`)
      if (!ok) return
    }
    setRevokeError('')
    setRevokingId(invitation.id)
    try {
      await revokeMutation.mutateAsync(invitation.id)
    } catch (error) {
      setRevokeError(error instanceof Error ? error.message : '吊销失败')
    } finally {
      setRevokingId(null)
    }
  }

  async function handleResend(invitation: OrganizationInvitation) {
    if (revokingId || resendingId) return
    setResendError('')
    setResendingId(invitation.id)
    try {
      const next = await resendMutation.mutateAsync(invitation.id)
      // 自动复制新链接，方便协作者直接拿去用。
      try {
        await navigator.clipboard.writeText(buildInviteUrl(next.token))
        setCopiedToken(next.id)
        window.setTimeout(
          () => setCopiedToken((current) => (current === next.id ? null : current)),
          2000,
        )
      } catch {
        // best-effort copy
      }
    } catch (error) {
      setResendError(error instanceof Error ? error.message : '重发失败')
    } finally {
      setResendingId(null)
    }
  }

  return (
    <div className="admin-settings-page">
      <header className="page-header">
        <Mail size={20} aria-hidden="true" />
        <h1>组织邀请</h1>
        <p className="page-subtitle">
          为协作者签发邀请链接；接受邀请的成员会按指定角色加入当前组织，而不是自动新建工作台。
        </p>
      </header>

      <section className="provider-card" aria-label="发起邀请">
        <div className="provider-card-header">
          <Plus size={18} aria-hidden="true" />
          <h2>新邀请</h2>
        </div>
        <form className="provider-card-body" onSubmit={handleCreate}>
          <label className="field-label">
            协作者 Email
            <input
              type="email"
              required
              placeholder="teammate@example.com"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </label>
          <label className="field-label">
            角色
            <select value={role} onChange={(event) => setRole(event.target.value as OrganizationInvitation['role'])}>
              {ROLE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <div className="provider-actions">
            <button type="submit" className="action-btn primary" disabled={createMutation.isPending}>
              {createMutation.isPending ? '生成中...' : '生成邀请链接'}
            </button>
          </div>
          {errorMessage ? <div className="test-result failure">创建失败: {errorMessage}</div> : null}
        </form>
      </section>

      <section className="provider-card" aria-label="邀请列表" style={{ marginTop: 16 }}>
        <div className="provider-card-header">
          <ShieldCheck size={18} aria-hidden="true" />
          <h2>已发出邀请 · {filteredInvitations.length}/{counts.all}</h2>
        </div>
        <div className="provider-card-body">
          <div className="invitation-toolbar">
            <div className="invitation-filter-chips" role="tablist">
              {STATUS_FILTERS.map((option) => {
                const count = counts[option.value]
                const isActive = statusFilter === option.value
                return (
                  <button
                    key={option.value}
                    type="button"
                    role="tab"
                    aria-selected={isActive}
                    className={`invitation-filter-chip${isActive ? ' is-active' : ''}`}
                    onClick={() => setStatusFilter(option.value)}
                  >
                    {option.label}
                    <span className="invitation-filter-chip-count">{count}</span>
                  </button>
                )
              })}
            </div>
            <label className="invitation-search">
              <Search size={14} aria-hidden="true" />
              <input
                type="search"
                placeholder="按 Email 搜索"
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
              />
            </label>
          </div>

          {invitationsQuery.isLoading ? (
            <StatePlaceholder tone="loading" title="正在加载邀请列表" />
          ) : filteredInvitations.length === 0 ? (
            <StatePlaceholder
              tone="empty"
              title={allInvitations.length === 0 ? '暂无邀请记录' : '当前筛选下没有邀请'}
              description={
                allInvitations.length === 0
                  ? '生成第一条邀请链接邀请协作者加入吧。'
                  : '尝试切换筛选条件或清空搜索框。'
              }
            />
          ) : (
            <ul className="invitation-list">
              {filteredInvitations.map((invitation) => {
                const expired = isExpired(invitation)
                const isRevoked = invitation.status === 'revoked'
                const statusKey = isRevoked ? 'revoked' : expired ? 'expired' : invitation.status
                const statusLabel =
                  invitation.status === 'accepted'
                    ? '已接受'
                    : isRevoked
                      ? '已吊销'
                      : expired
                        ? '已失效'
                        : '待接受'
                const canRevoke = invitation.status === 'pending' && !expired
                return (
                  <li key={invitation.id} className="invitation-row">
                    <div>
                      <strong>{invitation.email}</strong>
                      <small>
                        <span className={`invitation-status invitation-status-${statusKey}`}>
                          {statusLabel}
                        </span>
                        · {invitation.role}
                        {invitation.status === 'pending' && !expired
                          ? ` · 失效于 ${new Date(invitation.expires_at).toLocaleString()}`
                          : invitation.accepted_at
                            ? ` · 已接受 ${new Date(invitation.accepted_at).toLocaleString()}`
                            : expired
                              ? ` · 失效于 ${new Date(invitation.expires_at).toLocaleString()}`
                              : isRevoked
                                ? ' · 已被吊销'
                                : ''}
                      </small>
                    </div>
                    {canRevoke ? (
                      <div className="invitation-row-actions">
                        <button
                          type="button"
                          className="action-btn secondary"
                          onClick={() => handleCopy(invitation)}
                        >
                          {copiedToken === invitation.id ? (
                            <>
                              <Check size={14} aria-hidden="true" /> 已复制
                            </>
                          ) : (
                            <>
                              <Copy size={14} aria-hidden="true" /> 复制邀请链接
                            </>
                          )}
                        </button>
                        <button
                          type="button"
                          className="action-btn secondary"
                          onClick={() => handleResend(invitation)}
                          disabled={resendingId === invitation.id || revokingId === invitation.id}
                          title="生成新 token 并自动吊销旧链接"
                        >
                          <RefreshCw size={14} aria-hidden="true" />
                          {resendingId === invitation.id ? '重发中...' : '重发'}
                        </button>
                        <button
                          type="button"
                          className="action-btn warning"
                          onClick={() => handleRevoke(invitation)}
                          disabled={revokingId === invitation.id || resendingId === invitation.id}
                        >
                          <XCircle size={14} aria-hidden="true" />
                          {revokingId === invitation.id ? '吊销中...' : '吊销'}
                        </button>
                      </div>
                    ) : null}
                  </li>
                )
              })}
            </ul>
          )}
          {revokeError ? (
            <div className="test-result failure" role="alert">
              吊销失败：{revokeError}
            </div>
          ) : null}
          {resendError ? (
            <div className="test-result failure" role="alert">
              重发失败：{resendError}
            </div>
          ) : null}
        </div>
      </section>

      <section className="page-section">
        <header className="page-section-header">
          <h2>邀请审计日志</h2>
          <p>记录最近的「创建 / 接受 / 吊销 / 重发」动作，便于回溯责任与跨组织审计。</p>
        </header>
        <div className="page-section-body">
          {auditQuery.isLoading ? (
            <StatePlaceholder tone="loading" title="正在加载审计事件..." />
          ) : auditQuery.isError ? (
            <StatePlaceholder
              tone="error"
              title="加载审计日志失败"
              description={auditQuery.error instanceof Error ? auditQuery.error.message : ''}
            />
          ) : !auditQuery.data || auditQuery.data.length === 0 ? (
            <StatePlaceholder tone="empty" title="暂无审计事件" description="邀请相关动作发生后会出现在这里。" />
          ) : (
            <ul className="invitation-list">
              {auditQuery.data.map((event) => (
                <li key={event.id} className="invitation-row">
                  <div className="invitation-row-main">
                    <span className={`invitation-status invitation-status-${event.action}`}>
                      {event.action === 'created' && '已创建'}
                      {event.action === 'accepted' && '已接受'}
                      {event.action === 'revoked' && '已吊销'}
                      {event.action === 'resent' && '已重发'}
                    </span>
                    <span className="invitation-email">{event.email}</span>
                    <span className="invitation-role">{event.role}</span>
                  </div>
                  <div className="invitation-row-meta">
                    {event.actor_email ? <span>操作者：{event.actor_email}</span> : null}
                    {event.actor_user_id && !event.actor_email ? (
                      <span>操作者 ID：{event.actor_user_id.slice(0, 8)}…</span>
                    ) : null}
                    <span title={event.created_at}>{new Date(event.created_at).toLocaleString()}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>
    </div>
  )
}
