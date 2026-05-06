import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Copy, KeyRound, RefreshCcw, ShieldCheck } from 'lucide-react'
import {
  useChangePassword,
  useCreateUserAPIKey,
  useDeleteUserAPIKey,
  useToggleUserAPIKey,
  useUpdateUserAPIKey,
  useUserAPIKeys,
} from '../../api/hooks'
import type { APIKeyScope, UserAPIKey } from '../../api/types'
import { useAuthStore } from '../../state/authStore'

type TabKey = 'security' | 'apiKeys'

const API_KEY_SCOPE_OPTIONS: { value: APIKeyScope; label: string; hint: string }[] = [
  { value: 'read-only', label: 'Read-only', hint: '只读查询、自动化同步' },
  { value: 'write', label: 'Write', hint: '允许提交编辑与生成任务' },
  { value: 'admin', label: 'Admin', hint: '高权限运维与管理动作' },
]

const EMPTY_API_KEYS: UserAPIKey[] = []

function formatDateTime(value?: string | null) {
  if (!value) return '—'
  const ts = Date.parse(value)
  if (Number.isNaN(ts)) return value
  return new Date(ts).toLocaleString()
}

function toDateTimeLocal(value?: string | null) {
  if (!value) return ''
  const ts = Date.parse(value)
  if (Number.isNaN(ts)) return ''
  const date = new Date(ts)
  const offset = date.getTimezoneOffset()
  const local = new Date(date.getTime() - offset * 60_000)
  return local.toISOString().slice(0, 16)
}

function toISOOrNull(value: string) {
  if (!value) return null
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed.toISOString()
}

export function AccountSettingsPage() {
  const session = useAuthStore((state) => state.session)
  const apiKeysQuery = useUserAPIKeys(Boolean(session?.token))
  const changePasswordMutation = useChangePassword()
  const createAPIKeyMutation = useCreateUserAPIKey()
  const updateAPIKeyMutation = useUpdateUserAPIKey()
  const toggleAPIKeyMutation = useToggleUserAPIKey()
  const deleteAPIKeyMutation = useDeleteUserAPIKey()

  const [tab, setTab] = useState<TabKey>('security')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [passwordNotice, setPasswordNotice] = useState<string | null>(null)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [apiKeyError, setAPIKeyError] = useState<string | null>(null)
  const [createdToken, setCreatedToken] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [scope, setScope] = useState<APIKeyScope>('write')
  const [expiresAt, setExpiresAt] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)

  const apiKeys = apiKeysQuery.data ?? EMPTY_API_KEYS
  const activeCount = useMemo(() => apiKeys.filter((item) => item.is_active).length, [apiKeys])

  const handleChangePassword = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPasswordError(null)
    setPasswordNotice(null)
    try {
      await changePasswordMutation.mutateAsync({
        current_password: currentPassword,
        new_password: newPassword,
      })
      setCurrentPassword('')
      setNewPassword('')
      setPasswordNotice('密码已更新，当前登录会话保持有效。')
    } catch (error) {
      setPasswordError(error instanceof Error ? error.message : '更新密码失败')
    }
  }

  const handleCreateAPIKey = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setAPIKeyError(null)
    setCreatedToken(null)
    try {
      const created = await createAPIKeyMutation.mutateAsync({
        name,
        scope,
        expires_at: toISOOrNull(expiresAt),
      })
      setName('')
      setScope('write')
      setExpiresAt('')
      setCreatedToken(created.token)
      setTab('apiKeys')
    } catch (error) {
      setAPIKeyError(error instanceof Error ? error.message : '创建 API Key 失败')
    }
  }

  const handleToggleAPIKey = async (item: UserAPIKey) => {
    setAPIKeyError(null)
    try {
      await toggleAPIKeyMutation.mutateAsync({
        keyId: item.id,
        request: { is_active: !item.is_active },
      })
    } catch (error) {
      setAPIKeyError(error instanceof Error ? error.message : '更新 API Key 状态失败')
    }
  }

  const handleDeleteAPIKey = async (item: UserAPIKey) => {
    if (!window.confirm(`确认删除 API Key「${item.name}」吗？此操作不可恢复。`)) return
    setAPIKeyError(null)
    try {
      await deleteAPIKeyMutation.mutateAsync(item.id)
      if (editingId === item.id) setEditingId(null)
    } catch (error) {
      setAPIKeyError(error instanceof Error ? error.message : '删除 API Key 失败')
    }
  }

  return (
    <div className="account-settings-page">
      <header className="page-header">
        <ShieldCheck size={20} aria-hidden="true" />
        <h1>Account Settings</h1>
        <p className="page-subtitle">管理账号安全、个人 API Key 与自动化接入权限。</p>
      </header>

      <section className="account-settings-hero">
        <div className="account-settings-summary">
          <span className="section-kicker">Profile</span>
          <strong>{session?.user.display_name ?? '当前用户'}</strong>
          <span>{session?.user.email}</span>
        </div>
        <div className="account-settings-summary">
          <span className="section-kicker">Security</span>
          <strong>{activeCount}</strong>
          <span>个活跃 API Key</span>
        </div>
      </section>

      <div className="account-settings-tabs" role="tablist" aria-label="Account settings tabs">
        <button type="button" className={tab === 'security' ? 'is-active' : ''} onClick={() => setTab('security')}>
          <ShieldCheck size={16} aria-hidden="true" /> Security
        </button>
        <button type="button" className={tab === 'apiKeys' ? 'is-active' : ''} onClick={() => setTab('apiKeys')}>
          <KeyRound size={16} aria-hidden="true" /> API Keys
        </button>
      </div>

      {tab === 'security' ? (
        <section className="account-settings-grid">
          <article className="provider-card">
            <header className="provider-card-header">
              <ShieldCheck size={18} aria-hidden="true" />
              <h2>修改密码</h2>
            </header>
            <form className="account-settings-form" onSubmit={handleChangePassword}>
              <label>
                <span>当前密码</span>
                <input type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} />
              </label>
              <label>
                <span>新密码</span>
                <input type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} minLength={8} />
              </label>
              {passwordError ? <div className="form-error">{passwordError}</div> : null}
              {passwordNotice ? <div className="account-settings-success">{passwordNotice}</div> : null}
              <button type="submit" className="btn-primary" disabled={changePasswordMutation.isPending}>
                {changePasswordMutation.isPending ? '更新中...' : '更新密码'}
              </button>
            </form>
          </article>

          <article className="provider-card">
            <header className="provider-card-header">
              <RefreshCcw size={18} aria-hidden="true" />
              <h2>安全提示</h2>
            </header>
            <ul className="account-settings-checklist">
              <li>修改密码不会强制下线当前会话，若怀疑泄露请前往 Sessions 手动吊销。</li>
              <li>API Key 仅在创建时回显一次，建议立即复制到密码管理器。</li>
              <li>高权限自动化建议设置过期时间，并定期轮换 admin scope 密钥。</li>
            </ul>
          </article>
        </section>
      ) : (
        <section className="account-settings-grid">
          <article className="provider-card">
            <header className="provider-card-header">
              <KeyRound size={18} aria-hidden="true" />
              <h2>创建 API Key</h2>
            </header>
            <form className="account-settings-form" onSubmit={handleCreateAPIKey}>
              <label>
                <span>名称</span>
                <input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：Studio automation" />
              </label>
              <label>
                <span>权限范围</span>
                <select value={scope} onChange={(event) => setScope(event.target.value as APIKeyScope)}>
                  {API_KEY_SCOPE_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label} · {option.hint}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                <span>过期时间</span>
                <input type="datetime-local" value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} />
              </label>
              {apiKeyError ? <div className="form-error">{apiKeyError}</div> : null}
              {createdToken ? (
                <div className="account-token-preview">
                  <strong>仅显示一次</strong>
                  <code>{createdToken}</code>
                  <button type="button" className="btn-secondary" onClick={() => navigator.clipboard?.writeText(createdToken)}>
                    <Copy size={14} aria-hidden="true" /> 复制
                  </button>
                </div>
              ) : null}
              <button type="submit" className="btn-primary" disabled={createAPIKeyMutation.isPending}>
                {createAPIKeyMutation.isPending ? '创建中...' : '创建 API Key'}
              </button>
            </form>
          </article>

          <article className="provider-card">
            <header className="provider-card-header">
              <RefreshCcw size={18} aria-hidden="true" />
              <h2>已签发 Key</h2>
            </header>
            {apiKeysQuery.isLoading ? (
              <p className="muted">正在加载 API Keys...</p>
            ) : apiKeys.length === 0 ? (
              <div className="empty-state">
                <KeyRound size={18} aria-hidden="true" />
                <strong>还没有 API Key</strong>
                <span>创建后可用于 CLI、自动化脚本或外部集成。</span>
              </div>
            ) : (
              <div className="account-key-list">
                {apiKeys.map((item) => (
                  <APIKeyCard
                    key={item.id}
                    item={item}
                    editing={editingId === item.id}
                    pending={toggleAPIKeyMutation.isPending || updateAPIKeyMutation.isPending || deleteAPIKeyMutation.isPending}
                    onDelete={() => handleDeleteAPIKey(item)}
                    onEditToggle={() => setEditingId((current) => (current === item.id ? null : item.id))}
                    onSave={async (next) => {
                      setAPIKeyError(null)
                      try {
                        await updateAPIKeyMutation.mutateAsync({ keyId: item.id, request: next })
                        setEditingId(null)
                      } catch (error) {
                        setAPIKeyError(error instanceof Error ? error.message : '更新 API Key 失败')
                      }
                    }}
                    onToggle={() => handleToggleAPIKey(item)}
                  />
                ))}
              </div>
            )}
          </article>
        </section>
      )}
    </div>
  )
}

function APIKeyCard(props: {
  item: UserAPIKey
  editing: boolean
  pending: boolean
  onEditToggle: () => void
  onToggle: () => void
  onDelete: () => void
  onSave: (request: { name: string; scope: APIKeyScope; expires_at?: string | null }) => Promise<void>
}) {
  const { item, editing, pending, onDelete, onEditToggle, onSave, onToggle } = props
  const [name, setName] = useState(item.name)
  const [scope, setScope] = useState<APIKeyScope>(item.scope)
  const [expiresAt, setExpiresAt] = useState(toDateTimeLocal(item.expires_at))

  return (
    <div className="account-key-card">
      <div className="account-key-header">
        <div>
          <strong>{item.name}</strong>
          <div className="account-key-meta">
            <span>{item.token_preview}</span>
            <span>{item.scope}</span>
            <span>{item.is_active ? 'active' : 'disabled'}</span>
          </div>
        </div>
        <div className="account-key-actions">
          <button type="button" className="btn-secondary" disabled={pending} onClick={onToggle}>
            {item.is_active ? '禁用' : '启用'}
          </button>
          <button type="button" className="btn-secondary" disabled={pending} onClick={onEditToggle}>
            {editing ? '收起' : '编辑'}
          </button>
          <button type="button" className="btn-danger" disabled={pending} onClick={onDelete}>
            删除
          </button>
        </div>
      </div>
      <div className="account-key-meta">
        <span>创建于 {formatDateTime(item.created_at)}</span>
        <span>过期 {formatDateTime(item.expires_at)}</span>
        <span>最后使用 {formatDateTime(item.last_used_at)}</span>
      </div>
      {editing ? (
        <form
          className="account-key-editor"
          onSubmit={(event) => {
            event.preventDefault()
            void onSave({ expires_at: toISOOrNull(expiresAt), name, scope })
          }}
        >
          <input value={name} onChange={(event) => setName(event.target.value)} />
          <select value={scope} onChange={(event) => setScope(event.target.value as APIKeyScope)}>
            {API_KEY_SCOPE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <input type="datetime-local" value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} />
          <button type="submit" className="btn-primary" disabled={pending}>
            保存
          </button>
        </form>
      ) : null}
    </div>
  )
}
