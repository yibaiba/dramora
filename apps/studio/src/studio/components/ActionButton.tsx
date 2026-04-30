import type { LucideIcon } from 'lucide-react'

type ActionButtonProps = {
  disabled?: boolean
  disabledReason?: string
  icon: LucideIcon
  label: string
  onClick: () => void
}

export function ActionButton({
  disabled,
  disabledReason,
  icon: Icon,
  label,
  onClick,
}: ActionButtonProps) {
  return (
    <button
      className="ghost-action"
      disabled={disabled}
      onClick={onClick}
      title={disabledReason}
      type="button"
    >
      <Icon aria-hidden="true" />
      {label}
    </button>
  )
}
