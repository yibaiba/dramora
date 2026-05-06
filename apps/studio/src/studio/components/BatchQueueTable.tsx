import { ChevronLeft, ChevronRight, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import type { BatchSubmission } from '../../api/types'
import { StatePlaceholder } from './StatePlaceholder'

interface BatchQueueTableProps {
  batches: BatchSubmission[]
  isLoading?: boolean
  onCancel?: (batchId: string) => void
}

const PAGE_SIZE = 8

function statusLabel(status: BatchSubmission['status']) {
  switch (status) {
    case 'pending':
      return '待启动'
    case 'queued':
      return '已排队'
    case 'processing':
      return '处理中'
    case 'completed':
      return '已完成'
    case 'cancelled':
      return '已取消'
  }
}

function getProgressPercentage(batch: BatchSubmission) {
  if (batch.videoCount === 0) return 0
  const processed = batch.completedCount + batch.failedCount + batch.cancelledCount
  return Math.round((processed / batch.videoCount) * 100)
}

export default function BatchQueueTable({ batches, isLoading, onCancel }: BatchQueueTableProps) {
  const [pageIndex, setPageIndex] = useState(0)
  const pageCount = Math.max(1, Math.ceil(batches.length / PAGE_SIZE))

  const rows = useMemo(() => {
    const start = pageIndex * PAGE_SIZE
    return batches.slice(start, start + PAGE_SIZE)
  }, [batches, pageIndex])

  if (isLoading) {
    return <StatePlaceholder tone="loading" title="正在加载批量任务..." />
  }

  if (batches.length === 0) {
    return (
      <StatePlaceholder
        tone="empty"
        title="还没有批量任务"
        description="先创建几条视频记录，再批量提交到生成队列。"
      />
    )
  }

  return (
    <div className="short-video-batch-shell">
      <div className="short-video-batch-table" role="table" aria-label="批量提交任务表">
        <div className="short-video-batch-head" role="row">
          <span>批次</span>
          <span>状态</span>
          <span>视频数</span>
          <span>进度</span>
          <span>并发</span>
          <span>创建时间</span>
          <span>动作</span>
        </div>

        {rows.map((batch) => {
          const progress = getProgressPercentage(batch)
          const canCancel = batch.status !== 'completed' && batch.status !== 'cancelled'
          return (
            <div key={batch.id} className="short-video-batch-row" role="row">
              <span className="short-video-mono">{batch.id.slice(0, 8)}</span>
              <span className={`short-video-status-badge status-${batch.status}`}>{statusLabel(batch.status)}</span>
              <span>{batch.videoCount}</span>
              <span>
                <div className="short-video-inline-progress">
                  <div className="short-video-progress-track">
                    <div className="short-video-progress-fill" style={{ width: `${progress}%` }} />
                  </div>
                  <small>{progress}%</small>
                </div>
              </span>
              <span>{batch.concurrencyLimit}</span>
              <span>{new Date(batch.createdAt).toLocaleString('zh-CN')}</span>
              <span>
                {canCancel ? (
                  <button type="button" className="btn btn-danger" onClick={() => onCancel?.(batch.id)}>
                    <X size={14} aria-hidden="true" /> 取消
                  </button>
                ) : (
                  '—'
                )}
              </span>
            </div>
          )
        })}
      </div>

      <div className="short-video-batch-pagination">
        <span>
          第 {pageIndex + 1} / {pageCount} 页 · 共 {batches.length} 条任务
        </span>
        <div className="short-video-batch-page-actions">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => setPageIndex((current) => Math.max(0, current - 1))}
            disabled={pageIndex === 0}
          >
            <ChevronLeft size={16} aria-hidden="true" /> 上一页
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => setPageIndex((current) => Math.min(pageCount - 1, current + 1))}
            disabled={pageIndex >= pageCount - 1}
          >
            下一页 <ChevronRight size={16} aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  )
}
