import {
  Bell,
  Check,
  CheckCheck,
  Coins,
  Mail,
  RefreshCcw,
  Settings2,
  Sparkles,
} from 'lucide-react'
import { useMemo, useState } from 'react'

import { useMarkAllNotificationsAsRead, useMarkNotificationAsRead, useNotifications } from '../../api/hooks'
import type { Notification, NotificationKind } from '../../api/types'
import { StatePlaceholder } from '../components/StatePlaceholder'
import '../styles/NotificationsPage.css'

type NotificationFilter = 'all' | NotificationKind
type NotificationTone = 'billing' | 'invite' | 'config'
type NotificationChip = {
  key: string
  label: string
  value: string
}

const PAGE_SIZE = 30
const EMPTY_NOTIFICATIONS: Notification[] = []
const FILTER_OPTIONS: ReadonlyArray<{ key: NotificationFilter; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'wallet_credit', label: '充值' },
  { key: 'wallet_debit', label: '扣费' },
  { key: 'invitation_created', label: '邀请创建' },
  { key: 'invitation_resent', label: '邀请重发' },
  { key: 'provider_config_save', label: '提供商配置' },
]

const NOTIFICATION_KIND_META: Record<
  NotificationKind,
  {
    description: string
    label: string
    icon: typeof Coins
    tone: NotificationTone
  }
> = {
  invitation_created: {
    description: '组织协作邀请签发通知',
    icon: Mail,
    label: '邀请创建',
    tone: 'invite',
  },
  invitation_resent: {
    description: '已有邀请再次发送',
    icon: Mail,
    label: '邀请重发',
    tone: 'invite',
  },
  provider_config_save: {
    description: '模型与供应商参数已更新',
    icon: Settings2,
    label: '配置变更',
    tone: 'config',
  },
  wallet_credit: {
    description: '钱包积分增加',
    icon: Coins,
    label: '积分充值',
    tone: 'billing',
  },
  wallet_debit: {
    description: '钱包积分扣减',
    icon: Coins,
    label: '积分扣费',
    tone: 'billing',
  },
}

export function NotificationsPage() {
  const [limit, setLimit] = useState(PAGE_SIZE)
  const [unreadOnly, setUnreadOnly] = useState(false)
  const [selectedKind, setSelectedKind] = useState<NotificationFilter>('all')
  const [actionError, setActionError] = useState<string | null>(null)
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [isMarkingAll, setIsMarkingAll] = useState(false)

  const notificationsQuery = useNotifications({ limit, unread_only: unreadOnly })
  const markAsRead = useMarkNotificationAsRead()
  const markAllAsRead = useMarkAllNotificationsAsRead()

  const unreadCount = notificationsQuery.data?.unread_count ?? 0
  const notifications = notificationsQuery.data?.notifications ?? EMPTY_NOTIFICATIONS

  const countsByKind = useMemo(() => {
    const counts: Record<NotificationKind, number> = {
      invitation_created: 0,
      invitation_resent: 0,
      provider_config_save: 0,
      wallet_credit: 0,
      wallet_debit: 0,
    }

    for (const notification of notifications) {
      counts[notification.kind] += 1
    }

    return counts
  }, [notifications])

  const visibleNotifications = useMemo(
    () => (selectedKind === 'all' ? notifications : notifications.filter((notification) => notification.kind === selectedKind)),
    [notifications, selectedKind],
  )

  const summary = useMemo(
    () => ({
      billing: countsByKind.wallet_credit + countsByKind.wallet_debit,
      loaded: notifications.length,
      providers: countsByKind.provider_config_save,
      unread: unreadCount,
    }),
    [countsByKind, notifications.length, unreadCount],
  )

  const clearFilters = () => {
    setSelectedKind('all')
    setUnreadOnly(false)
    setLimit(PAGE_SIZE)
  }

  const handleRefresh = async () => {
    setActionError(null)
    await notificationsQuery.refetch()
  }

  const handleMarkAsRead = async (notificationId: string) => {
    setActionError(null)
    setPendingId(notificationId)
    try {
      await markAsRead.mutateAsync(notificationId)
    } catch (error) {
      setActionError(error instanceof Error ? error.message : '标记通知失败')
    } finally {
      setPendingId(null)
    }
  }

  const handleMarkAllAsRead = async () => {
    setActionError(null)
    setIsMarkingAll(true)
    try {
      await markAllAsRead.mutateAsync()
    } catch (error) {
      setActionError(error instanceof Error ? error.message : '批量标记通知失败')
    } finally {
      setIsMarkingAll(false)
    }
  }

  return (
    <main className="notifications-page" aria-busy={notificationsQuery.isFetching}>
      <header className="page-header notifications-page-header">
        <div className="notifications-title-block">
          <div className="notifications-page-icon" aria-hidden="true">
            <Bell size={20} />
          </div>
          <div>
            <span className="section-kicker">活动中心</span>
            <h1>通知中心</h1>
            <p className="page-subtitle">统一查看钱包变动、邀请协作和提供商配置动作，按未读与事件类型快速筛选。</p>
          </div>
        </div>
        <div className="notifications-header-actions">
          <button type="button" className="btn btn-secondary" onClick={handleRefresh} disabled={notificationsQuery.isFetching}>
            <RefreshCcw size={14} aria-hidden="true" />
            {notificationsQuery.isFetching ? '刷新中...' : '刷新'}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleMarkAllAsRead}
            disabled={isMarkingAll || unreadCount === 0}
          >
            <CheckCheck size={14} aria-hidden="true" />
            {isMarkingAll ? '处理中...' : '全部标为已读'}
          </button>
        </div>
      </header>

      <section className="notifications-hero" aria-label="通知概览">
        <article className="notifications-stat-card">
          <span className="notifications-stat-label">未读通知</span>
          <strong>{summary.unread}</strong>
          <p>优先处理需要你确认或回看的事件。</p>
        </article>
        <article className="notifications-stat-card">
          <span className="notifications-stat-label">当前已加载</span>
          <strong>{summary.loaded}</strong>
          <p>单页最多加载 {limit} 条，支持继续展开历史记录。</p>
        </article>
        <article className="notifications-stat-card">
          <span className="notifications-stat-label">账单相关</span>
          <strong>{summary.billing}</strong>
          <p>包含积分充值、扣费与余额变化线索。</p>
        </article>
        <article className="notifications-stat-card">
          <span className="notifications-stat-label">配置事件</span>
          <strong>{summary.providers}</strong>
          <p>追踪模型、能力和供应商配置更新。</p>
        </article>
      </section>

      <section className="notifications-toolbar" aria-label="通知筛选">
        <div className="notifications-toolbar-group">
          <span className="notifications-toolbar-label">视图</span>
          <div className="notifications-toggle-group" role="tablist" aria-label="通知视图">
            <button
              type="button"
              className={!unreadOnly ? 'is-active' : ''}
              onClick={() => setUnreadOnly(false)}
              aria-pressed={!unreadOnly}
            >
              全部
            </button>
            <button
              type="button"
              className={unreadOnly ? 'is-active' : ''}
              onClick={() => setUnreadOnly(true)}
              aria-pressed={unreadOnly}
            >
              仅看未读
            </button>
          </div>
        </div>

        <div className="notifications-toolbar-group">
          <span className="notifications-toolbar-label">类型</span>
          <div className="notifications-filter-list" role="tablist" aria-label="通知类型筛选">
            {FILTER_OPTIONS.map((option) => {
              const count = option.key === 'all' ? notifications.length : countsByKind[option.key]
              return (
                <button
                  type="button"
                  key={option.key}
                  className={selectedKind === option.key ? 'is-active' : ''}
                  onClick={() => setSelectedKind(option.key)}
                  aria-pressed={selectedKind === option.key}
                >
                  <span>{option.label}</span>
                  <strong>{count}</strong>
                </button>
              )
            })}
          </div>
        </div>
      </section>

      {actionError ? (
        <div className="form-error" role="alert">
          {actionError}
        </div>
      ) : null}

      {notificationsQuery.isError ? (
        <StatePlaceholder
          tone="error"
          title="通知中心加载失败"
          description={notificationsQuery.error instanceof Error ? notificationsQuery.error.message : '请稍后重试。'}
          action={
            <button type="button" className="btn btn-secondary" onClick={handleRefresh}>
              重新加载
            </button>
          }
        />
      ) : notificationsQuery.isLoading ? (
        <StatePlaceholder tone="loading" title="正在加载通知..." description="读取最新的钱包、邀请与配置事件。" />
      ) : notifications.length === 0 ? (
        <StatePlaceholder
          tone="empty"
          title={unreadOnly ? '当前没有未读通知' : '还没有通知'}
          description={unreadOnly ? '所有通知都已处理完成，可以切回全部视图查看历史。' : '新的账单、邀请和配置动作会在这里出现。'}
          icon={Bell}
          action={
            unreadOnly ? (
              <button type="button" className="btn btn-secondary" onClick={() => setUnreadOnly(false)}>
                查看全部通知
              </button>
            ) : undefined
          }
        />
      ) : visibleNotifications.length === 0 ? (
        <StatePlaceholder
          tone="empty"
          title="当前筛选条件下没有结果"
          description="切换通知类型或显示全部，查看其他事件。"
          icon={Sparkles}
          action={
            <button type="button" className="btn btn-secondary" onClick={clearFilters}>
              清空筛选
            </button>
          }
        />
      ) : (
        <section className="notifications-feed" aria-label="通知列表">
          {visibleNotifications.map((notification) => (
            <NotificationCard
              key={notification.id}
              notification={notification}
              pending={pendingId === notification.id}
              onMarkAsRead={handleMarkAsRead}
            />
          ))}
        </section>
      )}

      {notificationsQuery.data?.has_more ? (
        <div className="notifications-footer">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => setLimit((current) => current + PAGE_SIZE)}
            disabled={notificationsQuery.isFetching}
          >
            加载更多历史通知
          </button>
        </div>
      ) : null}
    </main>
  )
}

function NotificationCard(props: {
  notification: Notification
  pending: boolean
  onMarkAsRead: (notificationId: string) => Promise<void>
}) {
  const { notification, pending, onMarkAsRead } = props
  const meta = NOTIFICATION_KIND_META[notification.kind]
  const Icon = meta.icon
  const chips = formatNotificationChips(notification)
  const isUnread = !notification.read_at

  return (
    <article className={`notification-entry ${isUnread ? 'is-unread' : 'is-read'}`} data-tone={meta.tone}>
      <div className="notification-entry-icon" aria-hidden="true">
        <Icon size={18} />
      </div>

      <div className="notification-entry-body">
        <div className="notification-entry-topline">
          <div className="notification-entry-kind-group">
            <span className="notification-kind-pill">{meta.label}</span>
            <span className="notification-kind-description">{meta.description}</span>
          </div>
          <span className={`notification-read-state ${isUnread ? 'is-unread' : 'is-read'}`}>{isUnread ? '未读' : '已读'}</span>
        </div>

        <h2>{notification.title}</h2>
        <p>{notification.body}</p>

        <div className="notification-time-row">
          <span>{formatRelativeTime(notification.created_at)}</span>
          <span>{formatAbsoluteTime(notification.created_at)}</span>
        </div>

        {chips.length > 0 ? (
          <ul className="notification-chip-list" aria-label="通知详情">
            {chips.map((chip) => (
              <li key={chip.key}>
                <span>{chip.label}</span>
                <strong>{chip.value}</strong>
              </li>
            ))}
          </ul>
        ) : null}
      </div>

      <div className="notification-entry-actions">
        {isUnread ? (
          <button type="button" className="btn btn-primary" disabled={pending} onClick={() => void onMarkAsRead(notification.id)}>
            <Check size={14} aria-hidden="true" />
            {pending ? '处理中...' : '标记已读'}
          </button>
        ) : (
          <span className="notification-read-badge">已处理</span>
        )}
      </div>
    </article>
  )
}

function formatNotificationChips(notification: Notification): NotificationChip[] {
  const metadata = notification.metadata ?? {}

  switch (notification.kind) {
    case 'wallet_credit':
    case 'wallet_debit':
      return compactChips([
        toChip('amount', '变动', formatCredits(metadata.amount)),
        toChip('balance_after', '余额', formatCredits(metadata.balance_after)),
        toChip('reason', '原因', toShortText(metadata.reason)),
        toChip('transaction_id', '交易', toShortID(metadata.transaction_id)),
      ])
    case 'provider_config_save':
      return compactChips([
        toChip('provider_type', '提供商', toShortText(metadata.provider_type)),
        toChip('capability', '能力', toShortText(metadata.capability)),
        toChip('model', '模型', toShortText(metadata.model)),
      ])
    case 'invitation_created':
    case 'invitation_resent':
      return compactChips([
        toChip('email', '邮箱', toShortText(metadata.email)),
        toChip('role', '角色', toShortText(metadata.role)),
        toChip('invitation_id', '邀请', toShortID(metadata.invitation_id)),
      ])
    default:
      return []
  }
}

function compactChips(chips: Array<NotificationChip | null>): NotificationChip[] {
  return chips.filter((chip): chip is NotificationChip => Boolean(chip)).slice(0, 4)
}

function toChip(key: string, label: string, value: string | null): NotificationChip | null {
  if (!value) return null
  return { key, label, value }
}

function formatCredits(value: unknown): string | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return `${value.toLocaleString('zh-CN')} 积分`
  }

  if (typeof value === 'string' && value.trim() !== '') {
    return `${value.trim()} 积分`
  }

  return null
}

function toShortID(value: unknown): string | null {
  if (typeof value !== 'string' || value.trim() === '') {
    return null
  }

  const trimmed = value.trim()
  return trimmed.length > 14 ? `${trimmed.slice(0, 6)}…${trimmed.slice(-4)}` : trimmed
}

function toShortText(value: unknown): string | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }

  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  if (!trimmed) return null
  return trimmed.length > 36 ? `${trimmed.slice(0, 33)}…` : trimmed
}

function formatRelativeTime(isoString: string): string {
  const createdAt = new Date(isoString)
  const diffMs = Date.now() - createdAt.getTime()
  const diffMinutes = Math.floor(diffMs / 60_000)
  const diffHours = Math.floor(diffMs / 3_600_000)
  const diffDays = Math.floor(diffMs / 86_400_000)

  if (diffMinutes < 1) return '刚刚'
  if (diffMinutes < 60) return `${diffMinutes} 分钟前`
  if (diffHours < 24) return `${diffHours} 小时前`
  if (diffDays < 7) return `${diffDays} 天前`

  return createdAt.toLocaleDateString('zh-CN', {
    day: 'numeric',
    month: 'short',
  })
}

function formatAbsoluteTime(isoString: string): string {
  return new Date(isoString).toLocaleString('zh-CN', {
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}
