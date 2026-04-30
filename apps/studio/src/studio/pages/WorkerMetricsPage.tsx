import { useMemo, useState } from 'react'
import { Activity, AlertTriangle, Copy, ExternalLink, RefreshCcw, ShieldCheck } from 'lucide-react'
import { useWorkerMetrics } from '../../api/hooks'
import { useAuthStore } from '../../state/authStore'

const ADMIN_ROLES = new Set(['owner', 'admin'])

function formatRelative(iso?: string): string {
  if (!iso) return '尚未发生'
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return iso
  const diffMs = Date.now() - ts
  if (diffMs < 0) return new Date(ts).toLocaleString()
  const sec = Math.floor(diffMs / 1000)
  if (sec < 60) return `${sec}s 前`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m 前`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h 前`
  return new Date(ts).toLocaleString()
}

function formatKind(kind?: string): string {
  if (!kind) return '—'
  if (kind === 'generation') return 'Generation 任务'
  if (kind === 'export') return 'Export 任务'
  return kind
}

export function WorkerMetricsPage() {
  const session = useAuthStore((state) => state.session)
  const isAdmin = Boolean(session && ADMIN_ROLES.has(session.role))
  const metricsQuery = useWorkerMetrics(isAdmin)
  const [copied, setCopied] = useState<'url' | 'snippet' | null>(null)

  const metricsUrl = useMemo(() => {
    if (typeof window === 'undefined') return '/metrics'
    return `${window.location.origin}/metrics`
  }, [])

  const scrapeSnippet = useMemo(
    () =>
      [
        'scrape_configs:',
        '  - job_name: dramora-worker',
        '    metrics_path: /metrics',
        '    static_configs:',
        `      - targets: ['${typeof window !== 'undefined' ? window.location.host : 'api.example.com'}']`,
      ].join('\n'),
    [],
  )

  const handleCopy = async (value: string, kind: 'url' | 'snippet') => {
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(value)
        setCopied(kind)
        window.setTimeout(() => setCopied(null), 1500)
      }
    } catch {
      setCopied(null)
    }
  }

  const totals = useMemo(() => {
    const data = metricsQuery.data
    if (!data) return { total: 0, generation: 0, export: 0 }
    return {
      total: data.generation_org_unresolved_skips + data.export_org_unresolved_skips,
      generation: data.generation_org_unresolved_skips,
      export: data.export_org_unresolved_skips,
    }
  }, [metricsQuery.data])

  if (!isAdmin) {
    return (
      <div className="admin-settings-page">
        <header className="page-header">
          <ShieldCheck size={20} aria-hidden="true" />
          <h1>Worker Metrics</h1>
          <p className="page-subtitle">
            仅 owner / admin 可查看 worker 健康指标。当前角色：{session?.role ?? '未知'}
          </p>
        </header>
      </div>
    )
  }

  const data = metricsQuery.data

  return (
    <div className="admin-settings-page worker-metrics-page">
      <header className="page-header">
        <Activity size={20} aria-hidden="true" />
        <h1>Worker Metrics</h1>
        <p className="page-subtitle">
          展示 worker 在解析 job 组织上下文时被跳过的次数。计数已持久化，进程重启后可恢复；多进程部署时优先返回跨进程聚合视图。
        </p>
      </header>

      <section className="provider-card" aria-label="刷新">
        <div className="provider-card-header">
          <RefreshCcw size={18} aria-hidden="true" />
          <h2>当前快照{data?.source ? `（${data.source === 'aggregated' ? '跨进程聚合' : '本地进程'}）` : ''}</h2>
          <button
            className="action-btn secondary"
            onClick={() => metricsQuery.refetch()}
            type="button"
            disabled={metricsQuery.isFetching}
          >
            {metricsQuery.isFetching ? '刷新中…' : '立即刷新'}
          </button>
        </div>

        {metricsQuery.isError ? (
          <p className="provider-card-body field-error">
            读取失败：{(metricsQuery.error as Error)?.message ?? '未知错误'}
          </p>
        ) : null}

        {data ? (
          <div className="worker-metrics-grid">
            <article className={`worker-metric-card${totals.total > 0 ? ' has-skips' : ''}`}>
              <span className="worker-metric-label">累计跳过</span>
              <strong className="worker-metric-value">{totals.total}</strong>
              <small className="worker-metric-hint">所有类型 skip 的总数</small>
            </article>
            <article className="worker-metric-card">
              <span className="worker-metric-label">Generation 跳过</span>
              <strong className="worker-metric-value">{data.generation_org_unresolved_skips}</strong>
              <small className="worker-metric-hint">解析 generation_jobs 组织上下文失败次数</small>
            </article>
            <article className="worker-metric-card">
              <span className="worker-metric-label">Export 跳过</span>
              <strong className="worker-metric-value">{data.export_org_unresolved_skips}</strong>
              <small className="worker-metric-hint">解析 exports 组织上下文失败次数</small>
            </article>
          </div>
        ) : null}
      </section>

      <section className="provider-card" aria-label="最近一次跳过">
        <div className="provider-card-header">
          <AlertTriangle size={18} aria-hidden="true" />
          <h2>最近一次跳过</h2>
        </div>
        <div className="provider-card-body worker-metrics-last">
          <dl>
            <div>
              <dt>类型</dt>
              <dd>{formatKind(data?.last_skip_kind)}</dd>
            </div>
            <div>
              <dt>原因</dt>
              <dd>{data?.last_skip_reason || '—'}</dd>
            </div>
            <div>
              <dt>时间</dt>
              <dd>{formatRelative(data?.last_skip_at)}</dd>
            </div>
          </dl>
          {totals.total === 0 ? (
            <p className="worker-metrics-empty-hint">
              进程启动以来 worker 未跳过任何 job，组织上下文链路稳定。
            </p>
          ) : (
            <p className="worker-metrics-warn-hint">
              如果跳过持续增长，说明部分 job 的 project / organization 关联缺失，建议排查 worker 日志和 jobs 表。
            </p>
          )}
        </div>
      </section>
      <section className="provider-card" aria-label="Prometheus 抓取">
        <div className="provider-card-header">
          <ExternalLink size={18} aria-hidden="true" />
          <h2>Prometheus 抓取</h2>
        </div>
        <div className="provider-card-body worker-metrics-prom">
          <p className="worker-metrics-prom-desc">
            服务端在 <code>/metrics</code> 暴露 Prometheus 0.0.4 text exposition：
            <code>dramora_worker_org_unresolved_skips_total</code> counter 与
            <code>dramora_worker_last_skip_*</code> gauge。该端点公开，建议在反向代理 / 网络层做访问控制。
          </p>
          <div className="worker-metrics-prom-row">
            <code className="worker-metrics-prom-url">{metricsUrl}</code>
            <a
              className="action-btn secondary"
              href="/metrics"
              target="_blank"
              rel="noreferrer noopener"
            >
              打开 /metrics
            </a>
            <button
              className="action-btn secondary"
              type="button"
              onClick={() => handleCopy(metricsUrl, 'url')}
            >
              <Copy size={14} aria-hidden="true" />
              {copied === 'url' ? '已复制' : '复制 URL'}
            </button>
          </div>
          <details className="worker-metrics-prom-snippet">
            <summary>查看 Prometheus scrape config 片段</summary>
            <pre>{scrapeSnippet}</pre>
            <button
              className="action-btn secondary"
              type="button"
              onClick={() => handleCopy(scrapeSnippet, 'snippet')}
            >
              <Copy size={14} aria-hidden="true" />
              {copied === 'snippet' ? '已复制' : '复制 snippet'}
            </button>
          </details>
        </div>
      </section>
    </div>
  )
}
