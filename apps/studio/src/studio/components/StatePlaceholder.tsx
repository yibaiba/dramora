import type { LucideIcon } from 'lucide-react'
import { AlertTriangle, Inbox } from 'lucide-react'
import type { ReactNode } from 'react'

export type StatePlaceholderTone = 'loading' | 'empty' | 'error'

type StatePlaceholderProps = {
  tone: StatePlaceholderTone
  title: string
  description?: ReactNode
  icon?: LucideIcon
  action?: ReactNode
}

export function StatePlaceholder({
  tone,
  title,
  description,
  icon,
  action,
}: StatePlaceholderProps) {
  const Icon = icon ?? (tone === 'error' ? AlertTriangle : Inbox)
  return (
    <div
      className="state-placeholder"
      data-tone={tone}
      role={tone === 'error' ? 'alert' : 'status'}
    >
      {tone === 'loading' ? (
        <div className="state-placeholder-spinner" aria-hidden="true" />
      ) : (
        <div className="state-placeholder-icon">
          <Icon aria-hidden="true" size={20} />
        </div>
      )}
      <p className="state-placeholder-title">{title}</p>
      {description ? (
        <p className="state-placeholder-desc">{description}</p>
      ) : null}
      {action}
    </div>
  )
}
