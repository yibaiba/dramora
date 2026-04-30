import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Mail, Plus, ShieldCheck, Copy, Check } from 'lucide-react'
import { useCreateInvitation, useOrganizationInvitations } from '../../api/hooks'
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

export function InvitationsPage() {
  const session = useAuthStore((state) => state.session)
  const isAdmin = Boolean(session && ADMIN_ROLES.has(session.role))

  const invitationsQuery = useOrganizationInvitations(isAdmin)
  const createMutation = useCreateInvitation()

  const [email, setEmail] = useState('')
  const [role, setRole] = useState<OrganizationInvitation['role']>('editor')
  const [errorMessage, setErrorMessage] = useState('')
  const [copiedToken, setCopiedToken] = useState<string | null>(null)

  const invitations = useMemo(
    () => (invitationsQuery.data ?? []).slice().sort((a, b) => b.created_at.localeCompare(a.created_at)),
    [invitationsQuery.data],
  )

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
    const url =
      typeof window !== 'undefined'
        ? `${window.location.origin}${window.location.pathname}?invite=${encodeURIComponent(invitation.token)}`
        : invitation.token
    try {
      await navigator.clipboard.writeText(url)
      setCopiedToken(invitation.id)
      window.setTimeout(() => setCopiedToken((current) => (current === invitation.id ? null : current)), 2000)
    } catch {
      // best-effort copy; clipboard may be unavailable in some sandboxes
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
          <h2>已发出邀请 · {invitations.length}</h2>
        </div>
        <div className="provider-card-body">
          {invitationsQuery.isLoading ? (
            <StatePlaceholder tone="loading" title="正在加载邀请列表" />
          ) : invitations.length === 0 ? (
            <StatePlaceholder
              tone="empty"
              title="暂无邀请记录"
              description="生成第一条邀请链接邀请协作者加入吧。"
            />
          ) : (
            <ul className="invitation-list">
              {invitations.map((invitation) => (
                <li key={invitation.id} className="invitation-row">
                  <div>
                    <strong>{invitation.email}</strong>
                    <small>
                      {invitation.role} · {invitation.status}
                      {invitation.status === 'pending'
                        ? ` · 失效于 ${new Date(invitation.expires_at).toLocaleString()}`
                        : invitation.accepted_at
                          ? ` · 已接受 ${new Date(invitation.accepted_at).toLocaleString()}`
                          : ''}
                    </small>
                  </div>
                  {invitation.status === 'pending' ? (
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
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>
    </div>
  )
}
