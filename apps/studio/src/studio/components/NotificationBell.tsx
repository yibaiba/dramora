import { Bell, Check } from 'lucide-react'
import type { Notification } from '../../api/types'
import { useEffect, useRef, useState } from 'react'
import { useMarkAllNotificationsAsRead, useMarkNotificationAsRead, useNotifications } from '../../api/hooks'
import './NotificationBell.css'

export function NotificationBell() {
  const { data: notifData, isLoading } = useNotifications({ limit: 10, unread_only: false })
  const markAsRead = useMarkNotificationAsRead()
  const markAllAsRead = useMarkAllNotificationsAsRead()
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const unreadCount = notifData?.unread_count ?? 0
  const notifications = notifData?.notifications ?? []

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleMarkAsRead = (e: React.MouseEvent, notificationId: string) => {
    e.stopPropagation()
    markAsRead.mutate(notificationId)
  }

  const handleMarkAllAsRead = (e: React.MouseEvent) => {
    e.stopPropagation()
    markAllAsRead.mutate()
  }

  return (
    <div className="notification-bell" ref={dropdownRef}>
      <button
        className="notification-bell-button"
        onClick={() => setIsOpen(!isOpen)}
        aria-label={`通知 (${unreadCount} 条未读)`}
        title={`${unreadCount} 条未读通知`}
      >
        <Bell size={20} />
        {unreadCount > 0 && <span className="notification-badge">{unreadCount > 99 ? '99+' : unreadCount}</span>}
      </button>

      {isOpen && (
        <div className="notification-dropdown">
          <div className="notification-header">
            <span className="notification-title">通知</span>
            {unreadCount > 0 && (
              <button
                className="notification-mark-all"
                onClick={handleMarkAllAsRead}
                disabled={markAllAsRead.isPending}
              >
                全部已读
              </button>
            )}
          </div>

          <div className="notification-list">
            {isLoading ? (
              <div className="notification-loading">加载中...</div>
            ) : notifications.length === 0 ? (
              <div className="notification-empty">暂无通知</div>
            ) : (
              notifications.map((notif: Notification) => (
                <div
                  key={notif.id}
                  className={`notification-item ${notif.read_at ? 'read' : 'unread'}`}
                >
                  <div className="notification-content">
                    <div className="notification-item-title">{notif.title}</div>
                    <div className="notification-item-body">{notif.body}</div>
                    <div className="notification-item-time">{formatTime(notif.created_at)}</div>
                  </div>
                  {!notif.read_at && (
                    <button
                      className="notification-mark-btn"
                      onClick={(e) => handleMarkAsRead(e, notif.id)}
                      disabled={markAsRead.isPending}
                      aria-label="标记为已读"
                    >
                      <Check size={16} />
                    </button>
                  )}
                </div>
              ))
            )}
          </div>

          {notifications.length > 0 && (
            <div className="notification-footer">
              <a href="/notifications" className="notification-view-all">
                查看全部通知
              </a>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function formatTime(isoString: string): string {
  const date = new Date(isoString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins}分钟前`
  if (diffHours < 24) return `${diffHours}小时前`
  if (diffDays < 7) return `${diffDays}天前`
  
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
