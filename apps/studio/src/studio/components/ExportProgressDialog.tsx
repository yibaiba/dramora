import { X, CheckCircle2, AlertCircle } from 'lucide-react'
import { useState, useEffect } from 'react'
import type { Asset } from '../../api/types'
import { exportAssetsAsZip } from '../utils/export-utils'

type ExportState = 'preparing' | 'exporting' | 'success' | 'error'

export function ExportProgressDialog({
  selectedAssets,
  episodeId,
  onClose,
}: {
  selectedAssets: Asset[]
  episodeId: string
  onClose: () => void
}) {
  const [state, setState] = useState<ExportState>('preparing')
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const startExport = async () => {
      try {
        setState('exporting')
        await exportAssetsAsZip(selectedAssets, episodeId, (percent) => {
          setProgress(percent)
        })
        setState('success')
      } catch (err) {
        setError(err instanceof Error ? err.message : '导出失败')
        setState('error')
      }
    }

    startExport()
  }, [selectedAssets, episodeId])

  return (
    <div className="dialog-overlay" onClick={() => state === 'success' && onClose()}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <div className="dialog-header">
          <h2 className="dialog-title">导出素材包</h2>
          {(state === 'success' || state === 'error') && (
            <button
              onClick={onClose}
              className="dialog-close-btn"
              type="button"
              aria-label="关闭"
            >
              <X size={16} />
            </button>
          )}
        </div>

        <div className="dialog-body">
          {state === 'preparing' && (
            <p className="text-muted">正在准备 {selectedAssets.length} 个资产...</p>
          )}

          {state === 'exporting' && (
            <>
              <div className="export-progress-container">
                <div className="progress-bar">
                  <div className="progress-fill" style={{ width: `${progress}%` }} />
                </div>
                <div className="progress-text">{Math.round(progress)}%</div>
              </div>
              <p className="text-muted text-center">
                正在导出资产，请勿刷新页面...
              </p>
            </>
          )}

          {state === 'success' && (
            <div className="export-success">
              <CheckCircle2 size={32} className="icon-success" />
              <p>导出成功！已下载 {selectedAssets.length} 个资产。</p>
            </div>
          )}

          {state === 'error' && (
            <div className="export-error">
              <AlertCircle size={32} className="icon-error" />
              <p>导出失败：{error}</p>
            </div>
          )}
        </div>

        {state === 'success' && (
          <div className="dialog-footer">
            <button
              onClick={onClose}
              className="btn btn-primary"
              type="button"
            >
              完成
            </button>
          </div>
        )}

        {state === 'error' && (
          <div className="dialog-footer">
            <button
              onClick={onClose}
              className="btn btn-secondary"
              type="button"
            >
              关闭
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
