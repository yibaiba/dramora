import { X, CheckCircle2, AlertCircle } from 'lucide-react'
import { useState, useEffect } from 'react'
import type { Asset } from '../../api/types'
import { useDeleteAsset } from '../../api/hooks'

type DeleteState = 'deleting' | 'success' | 'partial-error' | 'error'

interface DeleteResult {
  successCount: number
  failureCount: number
  failures: Array<{ asset: Asset; error: string }>
}

export function DeleteProgressDialog({
  selectedAssets,
  episodeId,
  onClose,
}: {
  selectedAssets: Asset[]
  episodeId: string
  onClose: () => void
}) {
  const [state, setState] = useState<DeleteState>('deleting')
  const [progress, setProgress] = useState(0)
  const [result, setResult] = useState<DeleteResult | null>(null)
  const deleteAssetMutation = useDeleteAsset()

  useEffect(() => {
    const startDelete = async () => {
      const result: DeleteResult = {
        successCount: 0,
        failureCount: 0,
        failures: [],
      }

      const progressStep = 100 / selectedAssets.length

      for (let i = 0; i < selectedAssets.length; i++) {
        const asset = selectedAssets[i]
        try {
          await deleteAssetMutation.mutateAsync({
            assetId: asset.id,
            episodeId,
          })
          result.successCount++
        } catch (error) {
          result.failureCount++
          result.failures.push({
            asset,
            error: error instanceof Error ? error.message : '未知错误',
          })
        }
        setProgress(Math.round((i + 1) * progressStep))
      }

      setResult(result)
      if (result.failureCount === 0) {
        setState('success')
      } else if (result.successCount > 0) {
        setState('partial-error')
      } else {
        setState('error')
      }
    }

    startDelete()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedAssets, episodeId])

  return (
    <div className="dialog-overlay" onClick={() => (state !== 'deleting' ? onClose() : null)}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <div className="dialog-header">
          <h2 className="dialog-title">删除资产</h2>
          {state !== 'deleting' && (
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
          {state === 'deleting' && (
            <>
              <div className="export-progress-container">
                <div className="progress-bar">
                  <div className="progress-fill" style={{ width: `${progress}%` }} />
                </div>
                <div className="progress-text">{Math.round(progress)}%</div>
              </div>
              <p className="text-muted text-center">正在删除 {selectedAssets.length} 个资产...</p>
            </>
          )}

          {state === 'success' && result && (
            <div className="export-success">
              <CheckCircle2 size={32} className="icon-success" />
              <p>成功删除 {result.successCount} 个资产！</p>
            </div>
          )}

          {state === 'partial-error' && result && (
            <div className="export-error">
              <AlertCircle size={32} className="icon-error" />
              <p>
                部分删除成功：{result.successCount} 个成功，{result.failureCount} 个失败
              </p>
              {result.failures.length > 0 && (
                <div className="delete-failures">
                  <p className="text-muted" style={{ marginTop: '12px', marginBottom: '8px' }}>
                    失败的资产：
                  </p>
                  <ul className="failures-list">
                    {result.failures.slice(0, 3).map((item) => (
                      <li key={item.asset.id} className="failure-item">
                        <span className="failure-name">{item.asset.purpose}</span>
                        <span className="failure-reason">{item.error}</span>
                      </li>
                    ))}
                    {result.failures.length > 3 && (
                      <li className="failure-item-more">
                        还有 {result.failures.length - 3} 个失败
                      </li>
                    )}
                  </ul>
                </div>
              )}
            </div>
          )}

          {state === 'error' && result && (
            <div className="export-error">
              <AlertCircle size={32} className="icon-error" />
              <p>删除失败，所有 {result.failureCount} 个资产均未被删除。</p>
            </div>
          )}
        </div>

        {state !== 'deleting' && (
          <div className="dialog-footer">
            <button onClick={onClose} className="btn btn-primary" type="button">
              完成
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
