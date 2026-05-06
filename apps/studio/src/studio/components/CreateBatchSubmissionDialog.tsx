import { Loader2, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import type { CreateBatchSubmissionRequest } from '../../api/types'

interface CreateBatchSubmissionDialogProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (req: CreateBatchSubmissionRequest) => Promise<void>
  availableVideoIds: string[]
}

export default function CreateBatchSubmissionDialog({
  isOpen,
  onClose,
  onSubmit,
  availableVideoIds,
}: CreateBatchSubmissionDialogProps) {
  const [selectedVideoIds, setSelectedVideoIds] = useState<string[]>([])
  const [concurrencyLimit, setConcurrencyLimit] = useState(4)
  const [retryLimit, setRetryLimit] = useState(2)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const allSelected = useMemo(
    () => availableVideoIds.length > 0 && selectedVideoIds.length === availableVideoIds.length,
    [availableVideoIds.length, selectedVideoIds.length],
  )

  if (!isOpen) return null

  const resetForm = () => {
    setSelectedVideoIds([])
    setConcurrencyLimit(4)
    setRetryLimit(2)
    setError(null)
  }

  const toggleVideo = (videoId: string) => {
    setSelectedVideoIds((current) =>
      current.includes(videoId) ? current.filter((item) => item !== videoId) : [...current, videoId],
    )
  }

  const handleSubmit = async () => {
    if (selectedVideoIds.length === 0) {
      setError('至少选择一个视频后才能提交批量任务。')
      return
    }

    setIsSubmitting(true)
    setError(null)
    try {
      await onSubmit({
        videoIds: selectedVideoIds,
        concurrencyLimit,
        retryLimit,
        parameters: {},
      })
      resetForm()
      onClose()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : '创建批量任务失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="batch-dialog-backdrop" role="presentation">
      <div className="batch-dialog" role="dialog" aria-modal="true" aria-labelledby="batch-dialog-title">
        <header className="batch-dialog-header">
          <div>
            <h2 id="batch-dialog-title">创建批量任务</h2>
            <p>选择多个视频并设置并发与重试策略。</p>
          </div>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => {
              resetForm()
              onClose()
            }}
            disabled={isSubmitting}
            aria-label="关闭对话框"
          >
            <X size={16} aria-hidden="true" />
          </button>
        </header>

        <div className="batch-dialog-body">
          <section className="batch-dialog-section">
            <div className="batch-dialog-section-title">
              <strong>选择视频</strong>
              <button
                type="button"
                className="btn btn-secondary"
                onClick={() => setSelectedVideoIds(allSelected ? [] : availableVideoIds)}
                disabled={availableVideoIds.length === 0}
              >
                {allSelected ? '清空' : '全选'}
              </button>
            </div>
            <div className="batch-dialog-video-list">
              {availableVideoIds.length > 0 ? (
                availableVideoIds.map((videoId) => (
                  <label key={videoId} className="batch-dialog-video-item">
                    <input
                      type="checkbox"
                      checked={selectedVideoIds.includes(videoId)}
                      onChange={() => toggleVideo(videoId)}
                    />
                    <span className="short-video-mono">{videoId.slice(0, 12)}</span>
                  </label>
                ))
              ) : (
                <p className="batch-dialog-empty">暂无可供批量处理的视频。</p>
              )}
            </div>
          </section>

          <section className="batch-dialog-settings">
            <label>
              <span>并发上限</span>
              <input
                type="number"
                min={1}
                max={8}
                value={concurrencyLimit}
                onChange={(event) => setConcurrencyLimit(Math.min(8, Math.max(1, Number(event.target.value) || 1)))}
              />
            </label>
            <label>
              <span>失败重试</span>
              <input
                type="number"
                min={0}
                max={5}
                value={retryLimit}
                onChange={(event) => setRetryLimit(Math.min(5, Math.max(0, Number(event.target.value) || 0)))}
              />
            </label>
          </section>

          {error ? <div className="form-error">{error}</div> : null}
        </div>

        <footer className="batch-dialog-footer">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => {
              resetForm()
              onClose()
            }}
            disabled={isSubmitting}
          >
            取消
          </button>
          <button type="button" className="btn btn-primary" onClick={() => void handleSubmit()} disabled={isSubmitting}>
            {isSubmitting ? <Loader2 size={16} className="animate-spin" /> : null}
            {isSubmitting ? '提交中...' : '创建批量任务'}
          </button>
        </footer>
      </div>
    </div>
  )
}
