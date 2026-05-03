import { AlertTriangle, X } from 'lucide-react'

export function ConfirmDeleteDialog({
  selectedCount,
  onConfirm,
  onCancel,
  isLoading,
}: {
  selectedCount: number
  onConfirm: () => void
  onCancel: () => void
  isLoading: boolean
}) {
  return (
    <div className="dialog-overlay" onClick={onCancel}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <div className="dialog-header">
          <div className="dialog-title-row">
            <AlertTriangle size={20} className="dialog-icon-warning" />
            <h2 className="dialog-title">确认删除</h2>
          </div>
          <button
            onClick={onCancel}
            className="dialog-close-btn"
            type="button"
            aria-label="关闭"
          >
            <X size={16} />
          </button>
        </div>

        <div className="dialog-body">
          <p>
            您即将删除 <strong>{selectedCount} 个资产</strong>。此操作无法撤销。
          </p>
          <p className="text-muted">删除后，这些资产将从库中永久移除。</p>
        </div>

        <div className="dialog-footer">
          <button
            onClick={onCancel}
            className="btn btn-secondary"
            type="button"
            disabled={isLoading}
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            className="btn btn-danger"
            type="button"
            disabled={isLoading}
          >
            {isLoading ? '删除中...' : '确认删除'}
          </button>
        </div>
      </div>
    </div>
  )
}
