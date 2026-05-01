import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Mail, Plus, ShieldCheck, Copy, Check, Search, XCircle, RefreshCw, Download } from 'lucide-react'
import { useCreateInvitation, useOrganizationInvitations, useRevokeInvitation, useResendInvitation, useInvitationAuditEvents } from '../../api/hooks'
import { downloadInvitationAuditExport } from '../../api/client'
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

  // Audit log filter state
  const AUDIT_PAGE_SIZE = 20
  const [auditActions, setAuditActions] = useState<string[]>([])
  const [auditEmail, setAuditEmail] = useState('')
  const [auditEmailInput, setAuditEmailInput] = useState('')
  const [auditOffset, setAuditOffset] = useState(0)
  const [auditSince, setAuditSince] = useState('')
  const [auditUntil, setAuditUntil] = useState('')
  const [exportingFormat, setExportingFormat] = useState<'csv' | 'json' | null>(null)
  const [exportError, setExportError] = useState('')

  // Convert datetime-local value (YYYY-MM-DDTHH:mm) to RFC3339 in local TZ.
  const toRFC3339 = (local: string): string | undefined => {
    if (!local) return undefined
    const d = new Date(local)
    if (Number.isNaN(d.getTime())) return undefined
    return d.toISOString()
  }

  const sinceISO = toRFC3339(auditSince)
  const untilISO = toRFC3339(auditUntil)

  const dateRangeInvalid =
    sinceISO !== undefined && untilISO !== undefined && new Date(sinceISO).getTime() > new Date(untilISO).getTime()

  const auditQuery = useInvitationAuditEvents(isAdmin && !dateRangeInvalid, {
    limit: AUDIT_PAGE_SIZE,
    offset: auditOffset,
    actions: auditActions,
    email: auditEmail,
    since: sinceISO,
    until: untilISO,
  })

  const handleExportAudit = async (format: 'csv' | 'json') => {
    setExportError('')
    if (dateRangeInvalid) {
      setExportError('开始时间不能晚于结束时间')
      return
    }
    setExportingFormat(format)
    try {
      await downloadInvitationAuditExport(format, {
        actions: auditActions,
        email: auditEmail,
        since: sinceISO,
        until: untilISO,
      })
    } catch (err) {
      setExportError(err instanceof Error ? err.message : '导出失败')
    } finally {
      setExportingFormat(null)
    }
  }

  const handleClearDateRange = () => {
    setAuditSince('')
    setAuditUntil('')
    setAuditOffset(0)
  }

  const handleSinceChange = (value: string) => {
    setAuditSince(value)
    setAuditOffset(0)
  }

  const handleUntilChange = (value: string) => {
    setAuditUntil(value)
    setAuditOffset(0)
  }

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
          <div className="invitation-audit-toolbar" role="group" aria-label="审计日志过滤器">
            <div className="invitation-audit-actions">
              {(['created', 'accepted', 'revoked', 'resent'] as const).map((action) => {
                const active = auditActions.includes(action)
                return (
                  <button
                    key={action}
                    type="button"
                    className={`chip${active ? ' is-active' : ''}`}
                    onClick={() => {
                      setAuditOffset(0)
                      setAuditActions((prev) =>
                        prev.includes(action) ? prev.filter((a) => a !== action) : [...prev, action],
                      )
                    }}
                    aria-pressed={active}
                  >
                    {action === 'created' && '已创建'}
                    {action === 'accepted' && '已接受'}
                    {action === 'revoked' && '已吊销'}
                    {action === 'resent' && '已重发'}
                  </button>
                )
              })}
              {auditActions.length > 0 ? (
                <button
                  type="button"
                  className="chip"
                  onClick={() => {
                    setAuditActions([])
                    setAuditOffset(0)
                  }}
                >
                  清空类型筛选
                </button>
              ) : null}
            </div>
            <form
              className="invitation-audit-search"
              onSubmit={(e) => {
                e.preventDefault()
                setAuditEmail(auditEmailInput.trim())
                setAuditOffset(0)
              }}
            >
              <input
                type="search"
                className="search-input"
                placeholder="按邀请邮箱过滤（子串匹配）"
                value={auditEmailInput}
                onChange={(e) => setAuditEmailInput(e.target.value)}
              />
              <button type="submit" className="action-btn">
                <Search size={14} aria-hidden="true" />
                筛选
              </button>
              {auditEmail ? (
                <button
                  type="button"
                  className="action-btn"
                  onClick={() => {
                    setAuditEmail('')
                    setAuditEmailInput('')
                    setAuditOffset(0)
                  }}
                >
                  清除
                </button>
              ) : null}
            </form>
            <div className="invitation-audit-date-range" role="group" aria-label="时间范围">
              <label className="invitation-audit-date-field">
                <span>起始</span>
                <input
                  type="datetime-local"
                  value={auditSince}
                  onChange={(e) => handleSinceChange(e.target.value)}
                  aria-invalid={dateRangeInvalid || undefined}
                />
              </label>
              <label className="invitation-audit-date-field">
                <span>截止</span>
                <input
                  type="datetime-local"
                  value={auditUntil}
                  onChange={(e) => handleUntilChange(e.target.value)}
                  aria-invalid={dateRangeInvalid || undefined}
                />
              </label>
              {(auditSince || auditUntil) ? (
                <button type="button" className="action-btn" onClick={handleClearDateRange}>
                  清除时间
                </button>
              ) : null}
              {dateRangeInvalid ? (
                <span className="invitation-audit-date-error" role="alert">
                  起始时间不能晚于截止时间
                </span>
              ) : null}
            </div>
          </div>
          {auditQuery.isLoading ? (
            <StatePlaceholder tone="loading" title="正在加载审计事件..." />
          ) : auditQuery.isError ? (
            <StatePlaceholder
              tone="error"
              title="加载审计日志失败"
              description={auditQuery.error instanceof Error ? auditQuery.error.message : ''}
            />
          ) : !auditQuery.data || auditQuery.data.events.length === 0 ? (
            <StatePlaceholder
              tone="empty"
              title="暂无审计事件"
              description={
                auditActions.length > 0 || auditEmail
                  ? '当前筛选条件下没有匹配事件，可清空筛选重试。'
                  : '邀请相关动作发生后会出现在这里。'
              }
            />
          ) : (
            <>
              <ul className="invitation-list">
                {auditQuery.data.events.map((event) => (
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
              <div className="invitation-audit-pager">
                <span className="invitation-audit-pager-info">
                  第 {auditOffset + 1}–{auditOffset + auditQuery.data.events.length} 条
                  {auditQuery.data.has_more ? '（还有更多）' : ''}
                </span>
                <div className="invitation-audit-pager-actions">
                  <button
                    type="button"
                    className="action-btn"
                    disabled={auditOffset === 0}
                    onClick={() => setAuditOffset(Math.max(0, auditOffset - AUDIT_PAGE_SIZE))}
                  >
                    上一页
                  </button>
                  <button
                    type="button"
                    className="action-btn"
                    disabled={!auditQuery.data.has_more}
                    onClick={() => setAuditOffset(auditOffset + AUDIT_PAGE_SIZE)}
                  >
                    下一页
                  </button>
                  <button
                    type="button"
                    className="action-btn"
                    onClick={() => auditQuery.refetch()}
                    aria-label="刷新审计日志"
                  >
                    <RefreshCw size={14} aria-hidden="true" />
                    刷新
                  </button>
                  <button
                    type="button"
                    className="action-btn"
                    onClick={() => handleExportAudit('csv')}
                    disabled={exportingFormat !== null}
                    aria-label="导出 CSV"
                  >
                    <Download size={14} aria-hidden="true" />
                    {exportingFormat === 'csv' ? '导出中…' : '导出 CSV'}
                  </button>
                  <button
                    type="button"
                    className="action-btn"
                    onClick={() => handleExportAudit('json')}
                    disabled={exportingFormat !== null}
                    aria-label="导出 JSON"
                  >
                    <Download size={14} aria-hidden="true" />
                    {exportingFormat === 'json' ? '导出中…' : '导出 JSON'}
                  </button>
                </div>
                {exportError ? (
                  <p className="invitation-audit-export-error" role="alert">
                    导出失败：{exportError}
                  </p>
                ) : null}
              </div>
            </>
          )}
        </div>
      </section>
    </div>
  )
}
