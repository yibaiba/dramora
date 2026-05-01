import { useNotifications, useMarkNotificationAsRead, useMarkAllNotificationsAsRead } from '../../api/hooks'
import type { Notification } from '../../api/types'
import '../styles/NotificationsPage.css'

export function NotificationsPage() {
  const { data: notifData, isLoading } = useNotifications({ limit: 50 })
  const markAsRead = useMarkNotificationAsRead()
  const markAllAsRead = useMarkAllNotificationsAsRead()

  const unreadCount = notifData?.unread_count ?? 0
  const notifications = notifData?.notifications ?? []

  const handleMarkAsRead = (notificationId: string) => {
    markAsRead.mutate(notificationId)
  }

  const handleMarkAllAsRead = () => {
    markAllAsRead.mutate()
  }

  const notificationKindLabel: Record<string, string> = {
    wallet_credit: '积分充值',
    wallet_debit: '积分扣费',
    invitation_created: '邀请已发送',
    invitation_resent: '邀请已重新发送',
    provider_config_save: '配置已保存',
  }

  return (
    <div className="notifications-page">
      <div className="notifications-header">
        <div>
          <h1>通知中心</h1>
          <p>管理您的所有通知</p>
        </div>
        {unreadCount > 0 && (
          <button className="mark-all-btn" onClick={handleMarkAllAsRead} disabled={markAllAsRead.isPending}>
            全部标记为已读
          </button>
        )}
      </div>

      {isLoading ? (
        <div className="notifications-loading">加载通知中...</div>
      ) : notifications.length === 0 ? (
        <div className="notifications-empty">
          <div className="empty-icon">📭</div>
          <p>暂无通知</p>
        </div>
      ) : (
        <div className="notifications-list">
          {notifications.map((notif: Notification) => (
            <NotificationItemCard
              key={notif.id}
              notification={notif}
              onMarkAsRead={handleMarkAsRead}
              kindLabel={notificationKindLabel[notif.kind] || notif.kind}
              isMarking={markAsRead.isPending}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function NotificationItemCard({
  notification,
  onMarkAsRead,
  kindLabel,
  isMarking,
}: {
  notification: Notification
  onMarkAsRead: (id: string) => void
  kindLabel: string
  isMarking: boolean
}) {
  const isUnread = !notification.read_at
  const createdDate = new Date(notification.created_at)

  return (
    <div className={`notification-card ${isUnread ? 'unread' : 'read'}`}>
      <div className="notification-badge">{kindLabel}</div>

      <div className="notification-body">
        <div className="notification-title">{notification.title}</div>
        <div className="notification-description">{notification.body}</div>
        <div className="notification-meta">
          <span className="notification-time">{createdDate.toLocaleString('zh-CN')}</span>
          {notification.metadata && Object.keys(notification.metadata).length > 0 && (
            <span className="notification-metadata">
              {JSON.stringify(notification.metadata).substring(0, 50)}
            </span>
          )}
        </div>
      </div>

      {isUnread && (
        <button
          className="notification-action"
          onClick={() => onMarkAsRead(notification.id)}
          disabled={isMarking}
          aria-label="标记为已读"
        >
          标记已读
        </button>
      )}
    </div>
  )
}
