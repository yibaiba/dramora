import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Crown, RefreshCcw, ShieldCheck, UserMinus, Users } from 'lucide-react'
import { useOrganizationMembers, useRemoveOrganizationMember, useUpdateOrganizationMemberRole } from '../../api/hooks'
import type { OrganizationMember, OrganizationRole } from '../../api/types'
import { useAuthStore } from '../../state/authStore'
import { StatePlaceholder } from '../components/StatePlaceholder'
import { studioRoutePaths } from '../routes'

const ADMIN_ROLES = new Set(['owner', 'admin'])
const ROLE_PRIORITY: Record<OrganizationRole, number> = {
  owner: 0,
  admin: 1,
  editor: 2,
  viewer: 3,
}
const ROLE_OPTIONS: Array<{ value: OrganizationRole; label: string; hint: string }> = [
  { value: 'owner', label: 'Owner', hint: '全权所有者' },
  { value: 'admin', label: 'Admin', hint: '管理成员与配置' },
  { value: 'editor', label: 'Editor', hint: '可编辑生产内容' },
  { value: 'viewer', label: 'Viewer', hint: '只读观察' },
]

function formatDateTime(iso: string): string {
  const timestamp = Date.parse(iso)
  if (Number.isNaN(timestamp)) return iso
  return new Date(timestamp).toLocaleString()
}

function formatRelative(iso: string): string {
  const timestamp = Date.parse(iso)
  if (Number.isNaN(timestamp)) return iso
  const diffMs = Date.now() - timestamp
  if (diffMs < 60_000) return '刚刚活跃'
  const diffMinutes = Math.floor(diffMs / 60_000)
  if (diffMinutes < 60) return `${diffMinutes} 分钟前`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours} 小时前`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 30) return `${diffDays} 天前`
  return new Date(timestamp).toLocaleDateString()
}

function sortMembers(members: OrganizationMember[]): OrganizationMember[] {
  return [...members].sort((left, right) => {
    const roleDiff = ROLE_PRIORITY[left.role] - ROLE_PRIORITY[right.role]
    if (roleDiff !== 0) return roleDiff
    const joinedDiff = Date.parse(left.joined_at) - Date.parse(right.joined_at)
    if (!Number.isNaN(joinedDiff) && joinedDiff !== 0) return joinedDiff
    return left.display_name.localeCompare(right.display_name, 'zh-CN')
  })
}

type RemoveDialogProps = {
  member: OrganizationMember | null
  isSelf: boolean
  pending: boolean
  onCancel: () => void
  onConfirm: () => void
}

function RemoveMemberDialog({ member, isSelf, pending, onCancel, onConfirm }: RemoveDialogProps) {
  useEffect(() => {
    if (!member) return
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape' && !pending) {
        onCancel()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [member, onCancel, pending])

  if (!member) return null

  return (
    <div
      className="dialog-overlay"
      onClick={() => {
        if (!pending) onCancel()
      }}
      role="presentation"
    >
      <div
        aria-describedby="remove-member-description"
        aria-labelledby="remove-member-title"
        aria-modal="true"
        className="dialog-content"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
      >
        <div className="dialog-header">
          <div className="dialog-title-row">
            <UserMinus className="dialog-icon-warning" size={18} aria-hidden="true" />
            <div>
              <h2 className="dialog-title" id="remove-member-title">
                移除组织成员
              </h2>
              <p className="text-muted" id="remove-member-description">
                此操作会立即撤销该成员对当前组织的访问权限。
              </p>
            </div>
          </div>
        </div>

        <div className="dialog-body admin-users-dialog-body">
          <p>
            即将移除 <strong>{member.display_name || member.email}</strong>。
          </p>
          <p className="text-muted">
            邮箱：{member.email}
            <br />
            角色：{member.role}
            {isSelf ? <><br />你正在移除自己的组织访问，确认后当前会话会刷新。</> : null}
          </p>
        </div>

        <div className="dialog-footer">
          <button className="btn btn-secondary" disabled={pending} onClick={onCancel} type="button">
            取消
          </button>
          <button className="btn btn-danger" disabled={pending} onClick={onConfirm} type="button">
            {pending ? '移除中...' : '确认移除'}
          </button>
        </div>
      </div>
    </div>
  )
}

export function AdminUsersPage() {
  const session = useAuthStore((state) => state.session)
  const isAdmin = Boolean(session && ADMIN_ROLES.has(session.role))
  const membersQuery = useOrganizationMembers(isAdmin)
  const updateRoleMutation = useUpdateOrganizationMemberRole()
  const removeMemberMutation = useRemoveOrganizationMember()

  const [feedback, setFeedback] = useState<string | null>(null)
  const [pendingRoleUserId, setPendingRoleUserId] = useState<string | null>(null)
  const [removeTarget, setRemoveTarget] = useState<OrganizationMember | null>(null)

  const members = useMemo(() => sortMembers(membersQuery.data ?? []), [membersQuery.data])
  const counts = useMemo(
    () => ({
      total: members.length,
      owner: members.filter((member) => member.role === 'owner').length,
      admin: members.filter((member) => member.role === 'admin').length,
      editor: members.filter((member) => member.role === 'editor').length,
      viewer: members.filter((member) => member.role === 'viewer').length,
    }),
    [members],
  )

  if (!isAdmin) {
    return (
      <div className="admin-settings-page">
        <header className="page-header">
          <ShieldCheck size={20} aria-hidden="true" />
          <h1>组织成员</h1>
          <p className="page-subtitle">仅 owner / admin 可管理组织成员。当前角色：{session?.role ?? '未知'}</p>
        </header>
      </div>
    )
  }

  const handleRoleChange = async (member: OrganizationMember, role: OrganizationRole) => {
    if (member.role === role) return
    setFeedback(null)
    setPendingRoleUserId(member.user_id)
    try {
      await updateRoleMutation.mutateAsync({ role, userId: member.user_id })
      setFeedback(`${member.display_name || member.email} 已更新为 ${role}`)
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : '角色更新失败')
    } finally {
      setPendingRoleUserId(null)
    }
  }

  const handleConfirmRemove = async () => {
    if (!removeTarget) return
    setFeedback(null)
    try {
      await removeMemberMutation.mutateAsync(removeTarget.user_id)
      setFeedback(`${removeTarget.display_name || removeTarget.email} 已从组织中移除`)
      setRemoveTarget(null)
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : '移除成员失败')
    }
  }

  return (
    <div className="admin-settings-page admin-users-page">
      <header className="page-header">
        <Users size={20} aria-hidden="true" />
        <h1>组织成员</h1>
        <p className="page-subtitle">
          管理组织成员、角色与访问权限。角色变更会立即刷新成员列表；若修改到当前账号，会同步刷新当前会话。
        </p>
      </header>

      <section className="admin-users-toolbar">
        <div className="admin-users-summary-grid" aria-label="成员统计">
          <article className="admin-users-summary-card">
            <span className="admin-users-summary-label">总成员</span>
            <strong className="admin-users-summary-value">{counts.total}</strong>
            <small className="admin-users-summary-hint">当前组织已加入账号</small>
          </article>
          <article className="admin-users-summary-card">
            <span className="admin-users-summary-label">Owner / Admin</span>
            <strong className="admin-users-summary-value">{counts.owner + counts.admin}</strong>
            <small className="admin-users-summary-hint">具备管理权限的账号</small>
          </article>
          <article className="admin-users-summary-card">
            <span className="admin-users-summary-label">Editor</span>
            <strong className="admin-users-summary-value">{counts.editor}</strong>
            <small className="admin-users-summary-hint">可编辑生产内容</small>
          </article>
          <article className="admin-users-summary-card">
            <span className="admin-users-summary-label">Viewer</span>
            <strong className="admin-users-summary-value">{counts.viewer}</strong>
            <small className="admin-users-summary-hint">仅可查看状态与结果</small>
          </article>
        </div>

        <div className="admin-users-toolbar-actions">
          <button
            className="btn btn-secondary"
            disabled={membersQuery.isFetching}
            onClick={() => membersQuery.refetch()}
            type="button"
          >
            <RefreshCcw size={14} aria-hidden="true" />
            {membersQuery.isFetching ? '刷新中...' : '刷新成员'}
          </button>
          <Link className="btn btn-primary" to={studioRoutePaths.organizationInvitations}>
            邀请新成员
          </Link>
        </div>
      </section>

      {feedback ? (
        <p
          className={`admin-users-feedback${
            updateRoleMutation.isError || removeMemberMutation.isError ? ' is-error' : ''
          }`}
          role={updateRoleMutation.isError || removeMemberMutation.isError ? 'alert' : 'status'}
        >
          {feedback}
        </p>
      ) : null}

      <section className="provider-card" aria-label="成员列表">
        <div className="provider-card-header admin-users-section-header">
          <div className="admin-users-section-title">
            <Users size={18} aria-hidden="true" />
            <h2>成员卡片</h2>
          </div>
          <span className="admin-users-section-hint">
            {members.length} 位成员 · Owner 不可降级/移除至 0
          </span>
        </div>

        <div className="provider-card-body">
          {membersQuery.isLoading ? (
            <StatePlaceholder tone="loading" title="正在加载组织成员" description="读取成员身份与最近活跃时间。" />
          ) : membersQuery.isError ? (
            <StatePlaceholder
              action={
                <button className="state-placeholder-action" onClick={() => membersQuery.refetch()} type="button">
                  重新加载
                </button>
              }
              description={(membersQuery.error as Error)?.message ?? '无法获取组织成员。'}
              title="成员列表加载失败"
              tone="error"
            />
          ) : members.length === 0 ? (
            <StatePlaceholder
              action={
                <Link className="state-placeholder-action" to={studioRoutePaths.organizationInvitations}>
                  去邀请成员
                </Link>
              }
              description="当前组织还没有可管理的成员。"
              title="暂无成员"
              tone="empty"
            />
          ) : (
            <div className="admin-users-grid">
              {members.map((member) => {
                const isSelf = session?.user.id === member.user_id
                const rolePending = pendingRoleUserId === member.user_id && updateRoleMutation.isPending
                const removePending = removeTarget?.user_id === member.user_id && removeMemberMutation.isPending
                return (
                  <article
                    className={`admin-user-card${isSelf ? ' is-self' : ''}${member.role === 'owner' ? ' is-owner' : ''}`}
                    key={member.user_id}
                  >
                    <div className="admin-user-card-head">
                      <div className="admin-user-identity">
                        <strong className="admin-user-name">{member.display_name || member.email}</strong>
                        <span className="admin-user-email">{member.email}</span>
                      </div>
                      <div className="admin-user-badges">
                        <span className={`admin-user-role admin-user-role-${member.role}`}>
                          {member.role === 'owner' ? <Crown size={12} aria-hidden="true" /> : null}
                          {member.role}
                        </span>
                        {isSelf ? <span className="admin-user-self-badge">You</span> : null}
                      </div>
                    </div>

                    <dl className="admin-user-meta">
                      <div className="admin-user-meta-row">
                        <dt>User ID</dt>
                        <dd>
                          <code>{member.user_id}</code>
                        </dd>
                      </div>
                      <div className="admin-user-meta-row">
                        <dt>加入时间</dt>
                        <dd>{formatDateTime(member.joined_at)}</dd>
                      </div>
                      <div className="admin-user-meta-row">
                        <dt>最近活跃</dt>
                        <dd>
                          {formatRelative(member.last_activity_at)}
                          <span className="admin-user-meta-subtle">{formatDateTime(member.last_activity_at)}</span>
                        </dd>
                      </div>
                    </dl>

                    <div className="admin-user-actions">
                      <label className="admin-user-role-field">
                        <span>角色</span>
                        <select
                          aria-label={`修改 ${member.display_name || member.email} 的角色`}
                          className="admin-user-role-select"
                          disabled={rolePending || removePending}
                          onChange={(event) => handleRoleChange(member, event.target.value as OrganizationRole)}
                          value={member.role}
                        >
                          {ROLE_OPTIONS.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label} · {option.hint}
                            </option>
                          ))}
                        </select>
                      </label>
                      <button
                        className="btn btn-danger"
                        disabled={rolePending || removePending}
                        onClick={() => setRemoveTarget(member)}
                        type="button"
                      >
                        <UserMinus size={14} aria-hidden="true" />
                        {removePending ? '移除中...' : '移除'}
                      </button>
                    </div>

                    {isSelf ? (
                      <p className="admin-user-note">你修改自己的角色或成员身份后，当前会话会自动刷新。</p>
                    ) : null}
                  </article>
                )
              })}
            </div>
          )}
        </div>
      </section>

      <RemoveMemberDialog
        isSelf={session?.user.id === removeTarget?.user_id}
        member={removeTarget}
        onCancel={() => setRemoveTarget(null)}
        onConfirm={handleConfirmRemove}
        pending={removeMemberMutation.isPending}
      />
    </div>
  )
}
