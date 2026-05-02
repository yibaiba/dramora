import { Clock, X, RotateCw } from 'lucide-react'
import { useMemo, useState, useCallback } from 'react'
import { useGenerationJobs } from '../../api/hooks'
import type { GenerationJob, GenerationJobStatus } from '../../api/types'
import { StatePlaceholder } from '../components/StatePlaceholder'

type FilterStatus = 'all' | 'queued' | 'rendering' | 'succeeded' | 'failed'
type TimeRangeFilter = 'all' | '1h' | '24h'

const JOB_STATUS_MAP: Record<GenerationJobStatus, 'queued' | 'rendering' | 'succeeded' | 'failed' | 'other'> = {
  draft: 'other',
  preflight: 'queued',
  queued: 'queued',
  submitting: 'queued',
  submitted: 'queued',
  polling: 'rendering',
  downloading: 'rendering',
  postprocessing: 'rendering',
  needs_review: 'rendering',
  succeeded: 'succeeded',
  blocked: 'other',
  failed: 'failed',
  timed_out: 'failed',
  canceling: 'rendering',
  canceled: 'other',
}

const STATUS_LABELS: Record<GenerationJobStatus, string> = {
  draft: '草稿',
  preflight: '预检中',
  queued: '等待中',
  submitting: '提交中',
  submitted: '已提交',
  polling: '生成中',
  downloading: '下载中',
  postprocessing: '后处理中',
  needs_review: '待审批',
  succeeded: '成功',
  blocked: '已阻止',
  failed: '失败',
  timed_out: '超时',
  canceling: '取消中',
  canceled: '已取消',
}

function formatRelativeTime(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000)

    if (seconds < 60) return '刚刚'
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}分钟前`
    const hours = Math.floor(minutes / 60)
    if (hours < 24) return `${hours}小时前`
    const days = Math.floor(hours / 24)
    if (days < 7) return `${days}天前`
    return date.toLocaleDateString('zh-CN')
  } catch {
    return '未知'
  }
}

function getStatusTone(status: string): 'neutral' | 'success' | 'error' {
  if (status === 'succeeded') return 'success'
  if (status === 'failed' || status === 'timed_out' || status === 'blocked') return 'error'
  return 'neutral'
}

function isCancelable(status: GenerationJobStatus): boolean {
  return status === 'queued' || status === 'submitting' || status === 'submitted' || status === 'polling' || status === 'downloading' || status === 'postprocessing' || status === 'canceling'
}

function getUniqueTaskTypes(jobs: GenerationJob[]): string[] {
  const types = new Set(jobs.map((j) => j.task_type))
  return Array.from(types).sort()
}

function isWithinTimeRange(dateStr: string, range: TimeRangeFilter): boolean {
  if (range === 'all') return true
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    if (range === '1h') return diffMs <= 60 * 60 * 1000
    if (range === '24h') return diffMs <= 24 * 60 * 60 * 1000
    return true
  } catch {
    return false
  }
}

export function QueuePage() {
  const { data: jobs = [], isLoading, refetch } = useGenerationJobs({ refetchInterval: 2000 })
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('all')
  const [filterTaskType, setFilterTaskType] = useState<string>('all')
  const [filterTimeRange, setFilterTimeRange] = useState<TimeRangeFilter>('all')
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const uniqueTaskTypes = useMemo(() => getUniqueTaskTypes(jobs), [jobs])

  const stats = useMemo(() => {
    const all = jobs.length
    const queued = jobs.filter((j) => JOB_STATUS_MAP[j.status] === 'queued').length
    const rendering = jobs.filter((j) => JOB_STATUS_MAP[j.status] === 'rendering').length
    const succeeded = jobs.filter((j) => JOB_STATUS_MAP[j.status] === 'succeeded').length
    const failed = jobs.filter((j) => JOB_STATUS_MAP[j.status] === 'failed').length

    return { all, queued, rendering, succeeded, failed }
  }, [jobs])

  const filtered = useMemo(() => {
    let result = jobs

    if (filterStatus !== 'all') {
      const statusCategory = (filterStatus === 'rendering' ? 'rendering' : filterStatus) as
        | 'queued'
        | 'rendering'
        | 'succeeded'
        | 'failed'
      result = result.filter((job) => JOB_STATUS_MAP[job.status] === statusCategory)
    }

    if (filterTaskType !== 'all') {
      result = result.filter((job) => job.task_type === filterTaskType)
    }

    if (filterTimeRange !== 'all') {
      result = result.filter((job) => isWithinTimeRange(job.created_at, filterTimeRange))
    }

    return result.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  }, [jobs, filterStatus, filterTaskType, filterTimeRange])

  const handleManualRefresh = useCallback(async () => {
    setIsRefreshing(true)
    try {
      await refetch()
    } finally {
      setIsRefreshing(false)
    }
  }, [refetch])

  const clearFilters = () => {
    setFilterStatus('all')
    setFilterTaskType('all')
    setFilterTimeRange('all')
  }

  const activeFilterCount = (filterStatus !== 'all' ? 1 : 0) + (filterTaskType !== 'all' ? 1 : 0) + (filterTimeRange !== 'all' ? 1 : 0)

  const selectedJob = selectedJobId ? jobs.find((j) => j.id === selectedJobId) : null

  return (
    <section className="studio-page queue-page" aria-labelledby="queue-title">
      <div className="board-header">
        <div>
          <h1 id="queue-title">生成队列</h1>
          <span>实时监控所有生成任务的进度</span>
        </div>
      </div>

      <div className="dashboard-grid">
        <article className="surface-card">
          <span className="section-kicker">Total jobs</span>
          <strong>{stats.all} 个任务</strong>
          <p>队列中的全部生成任务。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">In progress</span>
          <strong>{stats.rendering} 个生成中</strong>
          <p>正在处理的生成任务数。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Succeeded</span>
          <strong>{stats.succeeded} 个成功</strong>
          <p>已完成的生成任务数。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Failed</span>
          <strong>{stats.failed} 个失败</strong>
          <p>失败的生成任务数。</p>
        </article>
      </div>

      <article className="surface-card">
        <div className="panel-title-row">
          <div>
            <span>Queue</span>
            <strong>生成队列 · {filtered.length} 个结果</strong>
          </div>
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            {activeFilterCount > 0 && (
              <button
                className="gallery-clear-filters"
                onClick={clearFilters}
                type="button"
                title="清空所有筛选条件"
              >
                <X size={16} aria-hidden="true" />
                清空筛选
              </button>
            )}
            <button
              className="gallery-refresh-btn"
              onClick={handleManualRefresh}
              disabled={isRefreshing}
              type="button"
              title="立即刷新队列"
            >
              <RotateCw size={16} aria-hidden="true" style={{ animation: isRefreshing ? 'spin 1s linear infinite' : 'none' }} />
              刷新
            </button>
          </div>
        </div>

        <div className="gallery-filters">
          <div className="filter-group">
            <span className="filter-label">状态</span>
            <div className="filter-buttons">
              <button
                className={filterStatus === 'all' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('all')}
                type="button"
              >
                全部 ({stats.all})
              </button>
              <button
                className={filterStatus === 'queued' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('queued')}
                type="button"
              >
                等待中 ({stats.queued})
              </button>
              <button
                className={filterStatus === 'rendering' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('rendering')}
                type="button"
              >
                生成中 ({stats.rendering})
              </button>
              <button
                className={filterStatus === 'succeeded' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('succeeded')}
                type="button"
              >
                成功 ({stats.succeeded})
              </button>
              <button
                className={filterStatus === 'failed' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('failed')}
                type="button"
              >
                失败 ({stats.failed})
              </button>
            </div>
          </div>

          {uniqueTaskTypes.length > 0 && (
            <div className="filter-group">
              <span className="filter-label">生成类型</span>
              <div className="filter-buttons">
                <button
                  className={filterTaskType === 'all' ? 'filter-btn active' : 'filter-btn'}
                  onClick={() => setFilterTaskType('all')}
                  type="button"
                >
                  全部
                </button>
                {uniqueTaskTypes.map((type) => (
                  <button
                    key={type}
                    className={filterTaskType === type ? 'filter-btn active' : 'filter-btn'}
                    onClick={() => setFilterTaskType(type)}
                    type="button"
                  >
                    {type}
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className="filter-group">
            <span className="filter-label">时间范围</span>
            <div className="filter-buttons">
              <button
                className={filterTimeRange === 'all' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterTimeRange('all')}
                type="button"
              >
                全部时间
              </button>
              <button
                className={filterTimeRange === '1h' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterTimeRange('1h')}
                type="button"
              >
                最近 1 小时
              </button>
              <button
                className={filterTimeRange === '24h' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterTimeRange('24h')}
                type="button"
              >
                最近 24 小时
              </button>
            </div>
          </div>
        </div>

        {isLoading ? (
          <StatePlaceholder tone="loading" title="加载中..." description="正在获取生成队列。" />
        ) : jobs.length === 0 ? (
          <StatePlaceholder
            tone="empty"
            icon={Clock}
            title="队列为空"
            description="当前没有生成任务。开始生成分镜或素材以查看队列。"
          />
        ) : filtered.length === 0 ? (
          <StatePlaceholder tone="empty" title="当前筛选条件下没有任务" description="请调整筛选条件。" />
        ) : (
          <>
            <div className="queue-grid">
              {filtered.map((job) => (
                <QueueJobCard key={job.id} job={job} isSelected={selectedJobId === job.id} onSelect={() => setSelectedJobId(job.id)} />
              ))}
            </div>
            {selectedJob && (
              <JobDetailPanel job={selectedJob} onClose={() => setSelectedJobId(null)} />
            )}
          </>
        )}
      </article>
    </section>
  )
}

function QueueJobCard({ job, isSelected, onSelect }: { job: GenerationJob; isSelected: boolean; onSelect: () => void }) {
  const tone = getStatusTone(job.status)
  const canCancel = isCancelable(job.status)
  const shortId = job.id.substring(0, 6)

  return (
    <article
      className={`queue-job-card tone-${tone} ${isSelected ? 'selected' : ''}`}
      onClick={onSelect}
      style={{ cursor: 'pointer' }}
    >
      <div className="job-header">
        <div className="job-title">
          <span className="job-id">{shortId}…</span>
          <h3>{job.task_type}</h3>
        </div>
        <span className={`job-status-badge status-${job.status}`}>{STATUS_LABELS[job.status]}</span>
      </div>

      <div className="job-meta">
        <div className="job-meta-row">
          <span className="meta-label">Model</span>
          <span className="meta-value">{job.model}</span>
        </div>
        <div className="job-meta-row">
          <span className="meta-label">Created</span>
          <span className="meta-value">{formatRelativeTime(job.created_at)}</span>
        </div>
      </div>

      {canCancel && (
        <button className="queue-cancel-btn" type="button" title="取消此任务" disabled>
          <X size={14} aria-hidden="true" />
          取消
        </button>
      )}
    </article>
  )
}

function JobDetailPanel({ job, onClose }: { job: GenerationJob; onClose: () => void }) {
  return (
    <div className="job-detail-panel-overlay" onClick={onClose}>
      <div className="job-detail-panel" onClick={(e) => e.stopPropagation()}>
        <div className="job-detail-header">
          <h2>Job 详情</h2>
          <button
            className="job-detail-close"
            onClick={onClose}
            type="button"
            title="关闭"
            aria-label="关闭 Job 详情面板"
          >
            <X size={20} aria-hidden="true" />
          </button>
        </div>

        <div className="job-detail-content">
          <div className="detail-section">
            <h3>基本信息</h3>
            <div className="detail-row">
              <span className="detail-label">Job ID</span>
              <span className="detail-value">{job.id}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">状态</span>
              <span className={`detail-status status-${job.status}`}>{STATUS_LABELS[job.status]}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">任务类型</span>
              <span className="detail-value">{job.task_type}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">模型</span>
              <span className="detail-value">{job.model}</span>
            </div>
          </div>

          <div className="detail-section">
            <h3>时间戳</h3>
            <div className="detail-row">
              <span className="detail-label">创建时间</span>
              <span className="detail-value">{new Date(job.created_at).toLocaleString('zh-CN')}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">更新时间</span>
              <span className="detail-value">{new Date(job.updated_at).toLocaleString('zh-CN')}</span>
            </div>
          </div>

          <div className="detail-section">
            <h3>项目关联</h3>
            <div className="detail-row">
              <span className="detail-label">Project ID</span>
              <span className="detail-value">{job.project_id}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Episode ID</span>
              <span className="detail-value">{job.episode_id}</span>
            </div>
            {job.workflow_run_id && (
              <div className="detail-row">
                <span className="detail-label">Workflow Run ID</span>
                <span className="detail-value">{job.workflow_run_id}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
