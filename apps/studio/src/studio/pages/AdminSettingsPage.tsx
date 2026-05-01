import { useCallback, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Settings, Zap, Image, Video, Volume2, Activity, Download } from 'lucide-react'
import { useLLMTelemetry, useProviderAuditEvents, useProviderConfigs, useResetLLMTelemetry, useSaveProviderConfig, useTestProviderConfig } from '../../api/hooks'
import { downloadProviderAuditCSV } from '../../api/client'
import { AgentStreamSandbox } from '../components/AgentStreamSandbox'
import type {
  ProviderCapability,
  ProviderConfig,
  ProviderType,
  SaveProviderConfigRequest,
  TestProviderResult,
} from '../../api/types'

const CAPABILITIES: { key: ProviderCapability; label: string; icon: typeof Zap }[] = [
  { icon: Zap, key: 'chat', label: 'LLM 对话' },
  { icon: Image, key: 'image', label: '图像生成' },
  { icon: Video, key: 'video', label: '视频生成' },
  { icon: Volume2, key: 'audio', label: 'TTS 语音' },
]

type TelemetryAlertThresholds = {
  minTotal: number
  errorRateWarn: number
  errorRateCritical: number
  vendorErrorsWarn: number
  vendorErrorsCritical: number
  capabilityErrorsCritical: number
}

const DEFAULT_ALERT_THRESHOLDS: TelemetryAlertThresholds = {
  capabilityErrorsCritical: 5,
  errorRateCritical: 25,
  errorRateWarn: 10,
  minTotal: 10,
  vendorErrorsCritical: 5,
  vendorErrorsWarn: 3,
}

const ALERT_THRESHOLDS_STORAGE_KEY = 'dramora-llm-telemetry-alert-thresholds'

function loadAlertThresholds(): TelemetryAlertThresholds {
  if (typeof window === 'undefined') return DEFAULT_ALERT_THRESHOLDS
  try {
    const raw = window.localStorage.getItem(ALERT_THRESHOLDS_STORAGE_KEY)
    if (!raw) return DEFAULT_ALERT_THRESHOLDS
    const parsed = JSON.parse(raw) as Partial<TelemetryAlertThresholds>
    return { ...DEFAULT_ALERT_THRESHOLDS, ...parsed }
  } catch {
    return DEFAULT_ALERT_THRESHOLDS
  }
}

function saveAlertThresholds(value: TelemetryAlertThresholds) {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(ALERT_THRESHOLDS_STORAGE_KEY, JSON.stringify(value))
  } catch {
    /* swallow */
  }
}

const PROVIDER_TYPE_OPTIONS: { value: ProviderType; label: string; hint: string }[] = [
  { hint: 'OpenAI 兼容 /chat/completions 网关（DeepSeek / Moonshot / vLLM 等）', label: 'OpenAI 兼容', value: 'openai' },
  { hint: 'Anthropic /v1/messages，使用 x-api-key 与 system 顶层字段', label: 'Anthropic Claude', value: 'anthropic' },
  { hint: 'Volces ARK Seedance 视频生成 (POST /contents/generations/tasks)', label: 'Seedance (ARK)', value: 'seedance' },
  { hint: '本地 mock 适配器：不发网络请求，输出可解析的占位 JSON', label: 'Mock（离线）', value: 'mock' },
]

// Mirrors backend service.CapabilityProviderTypes; keep in sync when
// new vendor adapters are added.
const CAPABILITY_PROVIDER_TYPES: Record<ProviderCapability, ProviderType[]> = {
  audio: ['openai', 'mock'],
  chat: ['openai', 'anthropic', 'mock'],
  image: ['openai', 'mock'],
  video: ['seedance', 'mock'],
}

const CAPABILITY_DEFAULT_PROVIDER_TYPE: Record<ProviderCapability, ProviderType> = {
  audio: 'openai',
  chat: 'openai',
  image: 'openai',
  video: 'seedance',
}

export function AdminSettingsPage() {
  const { data: configs = [] } = useProviderConfigs()

  return (
    <div className="admin-settings-page">
      <header className="page-header">
        <Settings size={20} aria-hidden="true" />
        <h1>端点配置</h1>
        <p className="page-subtitle">配置 AI 能力端点，所有 Agent 共享这些端点</p>
      </header>
      <div className="provider-grid">
        {CAPABILITIES.map((cap) => {
          const existing = configs.find((c) => c.capability === cap.key)
          return (
            <ProviderCard
              key={cap.key}
              capability={cap.key}
              icon={cap.icon}
              label={cap.label}
              config={existing}
            />
          )
        })}
      </div>
      <AgentStreamSandbox />
      <LLMTelemetryPanel />
      <ProviderAuditPanel />
    </div>
  )
}

function ProviderAuditPanel() {
  const [actionFilter, setActionFilter] = useState<'all' | 'save' | 'test'>('all')
  const [capabilityFilter, setCapabilityFilter] = useState<'all' | 'chat' | 'image' | 'video' | 'audio'>('all')
  const [sinceMinutes, setSinceMinutes] = useState<'all' | '15' | '60' | '1440' | '10080'>('all')
  const [actorInput, setActorInput] = useState('')
  const [actorFilter, setActorFilter] = useState('')

  const filter = useMemo(
    () => ({
      action: actionFilter === 'all' ? undefined : actionFilter,
      capability: capabilityFilter === 'all' ? undefined : capabilityFilter,
      sinceMinutes: sinceMinutes === 'all' ? undefined : Number(sinceMinutes),
      actor: actorFilter.trim() === '' ? undefined : actorFilter.trim(),
      limit: 100,
    }),
    [actionFilter, capabilityFilter, sinceMinutes, actorFilter],
  )

  const { data, isLoading, isError, error } = useProviderAuditEvents(filter)
  const [exportError, setExportError] = useState<string | null>(null)
  const [isExporting, setIsExporting] = useState(false)

  const onReset = () => {
    setActionFilter('all')
    setCapabilityFilter('all')
    setSinceMinutes('all')
    setActorInput('')
    setActorFilter('')
  }

  const onExport = async () => {
    setExportError(null)
    setIsExporting(true)
    try {
      const blob = await downloadProviderAuditCSV(filter)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      const ts = new Date().toISOString().replace(/[:.]/g, '-')
      a.download = `provider-audit-${ts}.csv`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch (err) {
      setExportError(err instanceof Error ? err.message : '导出失败')
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <section className="provider-card" style={{ marginTop: 24 }}>
      <header className="provider-card-header">
        <Activity size={18} aria-hidden="true" />
        <h2>Provider 配置审计</h2>
        <span className="provider-card-hint">
          每 15s 刷新 · {data ? `当前 ${data.events.length} 条${data.has_more ? ' (还有更多)' : ''}` : '加载中…'}
        </span>
      </header>
      <div
        style={{
          display: 'flex',
          gap: 12,
          flexWrap: 'wrap',
          alignItems: 'center',
          margin: '8px 0 12px',
          fontSize: 12,
        }}
      >
        <label style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          动作：
          <select value={actionFilter} onChange={(e) => setActionFilter(e.target.value as typeof actionFilter)}>
            <option value="all">全部</option>
            <option value="save">save</option>
            <option value="test">test</option>
          </select>
        </label>
        <label style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          Capability：
          <select
            value={capabilityFilter}
            onChange={(e) => setCapabilityFilter(e.target.value as typeof capabilityFilter)}
          >
            <option value="all">全部</option>
            <option value="chat">chat</option>
            <option value="image">image</option>
            <option value="video">video</option>
            <option value="audio">audio</option>
          </select>
        </label>
        <label style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          时间窗口：
          <select value={sinceMinutes} onChange={(e) => setSinceMinutes(e.target.value as typeof sinceMinutes)}>
            <option value="all">全部</option>
            <option value="15">最近 15 分钟</option>
            <option value="60">最近 1 小时</option>
            <option value="1440">最近 24 小时</option>
            <option value="10080">最近 7 天</option>
          </select>
        </label>
        <label style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          Actor：
          <input
            type="text"
            value={actorInput}
            onChange={(e) => setActorInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                setActorFilter(actorInput)
              }
            }}
            onBlur={() => setActorFilter(actorInput)}
            placeholder="email，多个用逗号分隔"
            style={{
              padding: '4px 8px',
              fontSize: 12,
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.2)',
              background: 'rgba(0,0,0,0.2)',
              color: 'inherit',
              minWidth: 200,
            }}
          />
        </label>
        <button type="button" onClick={onReset} className="btn-ghost" style={{ fontSize: 12 }}>
          重置筛选
        </button>
        <button
          type="button"
          onClick={onExport}
          disabled={isExporting || !data || data.events.length === 0}
          className="btn-ghost"
          style={{ fontSize: 12, display: 'inline-flex', alignItems: 'center', gap: 4 }}
          title="按当前筛选导出 CSV"
        >
          <Download size={12} />
          {isExporting ? '导出中…' : '导出 CSV'}
        </button>
      </div>
      {exportError && (
        <p className="error" style={{ fontSize: 12, marginTop: 0 }}>
          导出失败：{exportError}
        </p>
      )}
      {isLoading ? (
        <p className="muted">加载中…</p>
      ) : isError ? (
        <p className="error">无法加载（需要 owner/admin 权限）：{(error as Error)?.message ?? 'unknown'}</p>
      ) : !data || data.events.length === 0 ? (
        <p className="muted">当前筛选下无审计记录</p>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="telemetry-table" style={{ width: '100%', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={{ textAlign: 'left' }}>时间</th>
                <th style={{ textAlign: 'left' }}>动作</th>
                <th style={{ textAlign: 'left' }}>Capability</th>
                <th style={{ textAlign: 'left' }}>Vendor</th>
                <th style={{ textAlign: 'left' }}>Model</th>
                <th style={{ textAlign: 'left' }}>操作人</th>
                <th style={{ textAlign: 'left' }}>状态</th>
                <th style={{ textAlign: 'left' }}>备注</th>
              </tr>
            </thead>
            <tbody>
              {data.events.map((ev) => (
                <tr key={ev.id}>
                  <td>{ev.created_at ? new Date(ev.created_at).toLocaleString() : '-'}</td>
                  <td>{ev.action}</td>
                  <td>{ev.capability}</td>
                  <td>{ev.provider_type}</td>
                  <td>{ev.model || '-'}</td>
                  <td>{ev.actor_email || ev.actor_user_id || '-'}</td>
                  <td style={{ color: ev.success ? '#3ddc84' : '#ff6b6b' }}>{ev.success ? 'OK' : 'FAIL'}</td>
                  <td style={{ maxWidth: 360, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {ev.message || '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

function LLMTelemetryPanel() {
  const { data, isLoading, isError, error } = useLLMTelemetry()
  const resetMutation = useResetLLMTelemetry()

  const handleReset = useCallback(() => {
    if (resetMutation.isPending) return
    if (typeof window !== 'undefined' && !window.confirm('确定要重置 LLM Telemetry 计数吗？此操作会清空所有 vendor / capability 累计数据，且不可撤销。')) {
      return
    }
    resetMutation.mutate()
  }, [resetMutation])

  const [thresholds, setThresholds] = useState<TelemetryAlertThresholds>(() =>
    loadAlertThresholds(),
  )
  const updateThreshold = useCallback(
    (key: keyof TelemetryAlertThresholds, raw: string) => {
      const next = Number.parseFloat(raw)
      if (!Number.isFinite(next) || next < 0) return
      setThresholds((prev) => {
        const updated = { ...prev, [key]: next }
        saveAlertThresholds(updated)
        return updated
      })
    },
    [],
  )
  const resetThresholds = useCallback(() => {
    setThresholds(DEFAULT_ALERT_THRESHOLDS)
    saveAlertThresholds(DEFAULT_ALERT_THRESHOLDS)
  }, [])
  const [thresholdsOpen, setThresholdsOpen] = useState(false)

  const alerts = useMemo(() => {
    if (!data) return [] as { tone: 'warn' | 'critical'; text: string }[]
    const out: { tone: 'warn' | 'critical'; text: string }[] = []
    const total = data.total_calls ?? 0
    const errs = data.error_calls ?? 0
    if (total >= thresholds.minTotal && errs > 0) {
      const rate = (errs / total) * 100
      if (rate >= thresholds.errorRateCritical) {
        out.push({ tone: 'critical', text: `整体失败率 ${rate.toFixed(1)}%（${errs}/${total}），建议立即排查 provider 配置或额度。` })
      } else if (rate >= thresholds.errorRateWarn) {
        out.push({ tone: 'warn', text: `整体失败率 ${rate.toFixed(1)}%（${errs}/${total}），建议关注最近事件中的错误。` })
      }
    }
    Object.entries(data.errors_by_vendor ?? {}).forEach(([vendor, n]) => {
      if (n >= thresholds.vendorErrorsCritical) {
        out.push({ tone: 'critical', text: `${vendor} 已累计 ${n} 次失败，可能 vendor 异常。` })
      } else if (n >= thresholds.vendorErrorsWarn) {
        out.push({ tone: 'warn', text: `${vendor} 已累计 ${n} 次失败，建议检查 API key / 网络。` })
      }
    })
    Object.entries(data.errors_by_capability ?? {}).forEach(([cap, n]) => {
      if (n >= thresholds.capabilityErrorsCritical) {
        out.push({ tone: 'critical', text: `${cap} capability 累计 ${n} 次失败。` })
      }
    })
    return out
  }, [data, thresholds])

  return (
    <section className="provider-card" style={{ marginTop: 24 }}>
      <header className="provider-card-header">
        <Activity size={18} aria-hidden="true" />
        <h2>LLM 调用 Telemetry</h2>
        <span className="provider-card-hint">每 10s 刷新 · 进程内最近 50 条</span>
        <button
          type="button"
          onClick={handleReset}
          disabled={resetMutation.isPending}
          style={{
            marginLeft: 'auto',
            padding: '4px 10px',
            fontSize: 12,
            borderRadius: 6,
            border: '1px solid rgba(255,196,87,0.4)',
            background: 'rgba(255,196,87,0.08)',
            color: '#ffc457',
            cursor: resetMutation.isPending ? 'wait' : 'pointer',
          }}
          title="清空所有 vendor / capability 累计计数与最近事件，并删除持久化聚合行"
        >
          {resetMutation.isPending ? '重置中…' : '重置统计'}
        </button>
      </header>
      {resetMutation.isError && (
        <p className="error" style={{ marginTop: 4 }}>
          重置失败：{(resetMutation.error as Error)?.message ?? 'unknown'}
        </p>
      )}
      <div style={{ marginBottom: 8 }}>
        <button
          type="button"
          onClick={() => setThresholdsOpen((prev) => !prev)}
          style={{
            padding: '4px 10px',
            fontSize: 12,
            borderRadius: 6,
            border: '1px solid rgba(255,255,255,0.18)',
            background: 'rgba(255,255,255,0.04)',
            color: '#cbd5f5',
            cursor: 'pointer',
          }}
          title="编辑触发告警的阈值（仅本浏览器持久化）"
        >
          {thresholdsOpen ? '收起阈值设置' : '调整告警阈值'}
        </button>
        {thresholdsOpen && (
          <div
            style={{
              marginTop: 8,
              padding: 10,
              borderRadius: 8,
              border: '1px solid rgba(255,255,255,0.08)',
              background: 'rgba(255,255,255,0.02)',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
              gap: 8,
              fontSize: 12,
            }}
          >
            {(
              [
                ['minTotal', '失败率最小样本数'],
                ['errorRateWarn', '失败率 warn (%)'],
                ['errorRateCritical', '失败率 critical (%)'],
                ['vendorErrorsWarn', '单 vendor 失败 warn'],
                ['vendorErrorsCritical', '单 vendor 失败 critical'],
                ['capabilityErrorsCritical', '单 capability 失败 critical'],
              ] as [keyof TelemetryAlertThresholds, string][]
            ).map(([key, label]) => (
              <label key={key} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                <span className="muted" style={{ fontSize: 11 }}>{label}</span>
                <input
                  type="number"
                  min={0}
                  step={key.startsWith('errorRate') ? 0.5 : 1}
                  value={thresholds[key]}
                  onChange={(event) => updateThreshold(key, event.target.value)}
                  style={{
                    padding: '4px 6px',
                    borderRadius: 4,
                    border: '1px solid rgba(255,255,255,0.12)',
                    background: 'rgba(0,0,0,0.25)',
                    color: '#e6edf3',
                  }}
                />
              </label>
            ))}
            <button
              type="button"
              onClick={resetThresholds}
              style={{
                alignSelf: 'flex-end',
                padding: '4px 10px',
                fontSize: 12,
                borderRadius: 6,
                border: '1px solid rgba(255,255,255,0.18)',
                background: 'transparent',
                color: '#cbd5f5',
                cursor: 'pointer',
              }}
              title="恢复默认阈值"
            >
              恢复默认
            </button>
          </div>
        )}
      </div>
      {alerts.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 12 }}>
          {alerts.map((a, i) => (
            <div
              key={i}
              role="alert"
              style={{
                padding: '8px 12px',
                borderRadius: 6,
                fontSize: 12,
                background: a.tone === 'critical' ? 'rgba(255,107,107,0.12)' : 'rgba(255,196,87,0.12)',
                border: `1px solid ${a.tone === 'critical' ? 'rgba(255,107,107,0.4)' : 'rgba(255,196,87,0.4)'}`,
                color: a.tone === 'critical' ? '#ff6b6b' : '#ffc457',
              }}
            >
              {a.tone === 'critical' ? '⚠ 严重：' : '⚠ 提示：'}
              {a.text}
            </div>
          ))}
        </div>
      )}
      {isLoading ? (
        <p className="muted">加载中…</p>
      ) : isError ? (
        <p className="error">无法加载（需要 owner/admin 权限）：{(error as Error)?.message ?? 'unknown'}</p>
      ) : !data ? (
        <p className="muted">暂无数据</p>
      ) : (
        <>
          <div className="telemetry-summary" style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12 }}>
            <Stat label="总调用" value={data.total_calls} />
            <Stat label="成功" value={data.success_calls} tone="ok" />
            <Stat label="失败" value={data.error_calls} tone={data.error_calls > 0 ? 'warn' : undefined} />
            {Object.entries(data.by_vendor ?? {}).map(([vendor, count]) => (
              <Stat
                key={vendor}
                label={`${vendor} · 平均`}
                value={`${data.avg_duration_ms_by_vendor?.[vendor] ?? 0}ms`}
                sub={`${count} 次${data.errors_by_vendor?.[vendor] ? ` · 失败 ${data.errors_by_vendor[vendor]}` : ''}`}
                tone={data.errors_by_vendor?.[vendor] ? 'warn' : undefined}
              />
            ))}
          </div>
          {data.by_capability && Object.keys(data.by_capability).length > 0 ? (
            <div
              className="telemetry-by-capability"
              style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 12 }}
            >
              <span className="muted" style={{ fontSize: 12, alignSelf: 'center' }}>
                按能力：
              </span>
              {Object.entries(data.by_capability).map(([cap, count]) => (
                <Stat
                  key={cap}
                  label={cap}
                  value={count}
                  sub={`${data.avg_duration_ms_by_capability?.[cap] ?? 0}ms${
                    data.errors_by_capability?.[cap] ? ` · 失败 ${data.errors_by_capability[cap]}` : ''
                  }`}
                  tone={data.errors_by_capability?.[cap] ? 'warn' : undefined}
                />
              ))}
            </div>
          ) : null}
          {data.window ? (
            <div
              className="telemetry-window"
              style={{
                display: 'flex',
                gap: 12,
                flexWrap: 'wrap',
                marginBottom: 12,
                padding: '8px 12px',
                border: '1px solid rgba(255,255,255,0.06)',
                borderRadius: 6,
                background: 'rgba(255,255,255,0.02)',
              }}
            >
              <span className="muted" style={{ fontSize: 12, alignSelf: 'center' }}>
                最近 {data.window.days} 天（自 {data.window.since_day_utc} UTC）：
              </span>
              <Stat label="窗口调用" value={data.window.total_calls} />
              <Stat
                label="窗口失败"
                value={data.window.error_calls}
                tone={data.window.error_calls > 0 ? 'warn' : undefined}
              />
              {Object.entries(data.window.by_vendor ?? {}).map(([vendor, count]) => (
                <Stat
                  key={`win-v-${vendor}`}
                  label={`${vendor} · ${count}`}
                  value={`${data.window?.avg_duration_ms_by_vendor?.[vendor] ?? 0}ms`}
                  sub={
                    data.window?.errors_by_vendor?.[vendor]
                      ? `失败 ${data.window.errors_by_vendor[vendor]}`
                      : undefined
                  }
                  tone={data.window?.errors_by_vendor?.[vendor] ? 'warn' : undefined}
                />
              ))}
            </div>
          ) : null}
          {data.recent_events && data.recent_events.length > 0 ? (
            <div style={{ overflowX: 'auto' }}>
              <table className="telemetry-table" style={{ width: '100%', fontSize: 12 }}>
                <thead>
                  <tr>
                    <th style={{ textAlign: 'left' }}>时间</th>
                    <th style={{ textAlign: 'left' }}>Capability</th>
                    <th style={{ textAlign: 'left' }}>Vendor</th>
                    <th style={{ textAlign: 'left' }}>Model</th>
                    <th style={{ textAlign: 'left' }}>Role</th>
                    <th style={{ textAlign: 'left' }}>Mode</th>
                    <th style={{ textAlign: 'right' }}>耗时</th>
                    <th style={{ textAlign: 'right' }}>Tokens</th>
                    <th style={{ textAlign: 'left' }}>状态</th>
                  </tr>
                </thead>
                <tbody>
                  {data.recent_events.slice(0, 10).map((ev, idx) => (
                    <tr key={`${ev.started_at}-${idx}`}>
                      <td>{ev.started_at ? new Date(ev.started_at).toLocaleTimeString() : '-'}</td>
                      <td>{ev.capability || 'chat'}</td>
                      <td>{ev.vendor}</td>
                      <td>{ev.model || '-'}</td>
                      <td>{ev.role || '-'}</td>
                      <td>{ev.mode}</td>
                      <td style={{ textAlign: 'right' }}>{ev.duration_ms}ms</td>
                      <td style={{ textAlign: 'right' }}>{ev.token_count || '-'}</td>
                      <td style={{ color: ev.success ? '#3ddc84' : '#ff6b6b' }}>
                        {ev.success ? 'OK' : ev.error_message || 'ERR'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="muted">尚无 LLM 调用记录</p>
          )}
        </>
      )}
    </section>
  )
}

function Stat({ label, value, sub, tone }: { label: string; value: string | number; sub?: string; tone?: 'ok' | 'warn' }) {
  const color = tone === 'ok' ? '#3ddc84' : tone === 'warn' ? '#ff8c42' : undefined
  return (
    <div style={{ minWidth: 120 }}>
      <div style={{ fontSize: 11, opacity: 0.6 }}>{label}</div>
      <div style={{ fontSize: 18, fontWeight: 600, color }}>{value}</div>
      {sub ? <div style={{ fontSize: 10, opacity: 0.5 }}>{sub}</div> : null}
    </div>
  )
}

function ProviderCard({
  capability,
  config,
  icon: Icon,
  label,
}: {
  capability: ProviderCapability
  config?: ProviderConfig
  icon: typeof Zap
  label: string
}) {
  const saveMutation = useSaveProviderConfig()
  const testMutation = useTestProviderConfig()
  const [editing, setEditing] = useState(!config)
  const allowedTypes = CAPABILITY_PROVIDER_TYPES[capability]
  const defaultType = CAPABILITY_DEFAULT_PROVIDER_TYPE[capability]
  const initialType: ProviderType = config?.provider_type && allowedTypes.includes(config.provider_type)
    ? config.provider_type
    : defaultType
  const [providerType, setProviderType] = useState<ProviderType>(initialType)
  const [baseUrl, setBaseUrl] = useState(config?.base_url ?? '')
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState(config?.model ?? '')
  const [creditsPerUnit, setCreditsPerUnit] = useState(config?.credits_per_unit ?? 5)
  const [testResult, setTestResult] = useState<TestProviderResult | null>(null)

  function handleSave(e: FormEvent) {
    e.preventDefault()
    const request: SaveProviderConfigRequest = {
      api_key: apiKey || config?.api_key || '',
      base_url: baseUrl,
      capability,
      credit_unit: 'per_call',
      credits_per_unit: creditsPerUnit,
      max_retries: 3,
      model,
      provider_type: providerType,
      timeout_ms: 120000,
    }
    saveMutation.mutate(request, {
      onSuccess: () => setEditing(false),
    })
  }

  function handleTest() {
    setTestResult(null)
    testMutation.mutate(capability, {
      onError: () => setTestResult({ error: '请求失败', latency_ms: 0, model: '', ok: false }),
      onSuccess: (r) => setTestResult(r),
    })
  }

  return (
    <section className="provider-card" aria-label={label}>
      <div className="provider-card-header">
        <Icon size={18} aria-hidden="true" />
        <h2>{label}</h2>
        <span className={`provider-status ${config ? 'configured' : 'unconfigured'}`}>
          {config ? '已配置' : '未配置'}
        </span>
      </div>

      {!editing && config ? (
        <div className="provider-card-body">
          <dl className="provider-fields">
            <dt>适配器</dt>
            <dd>{providerTypeLabel(config.provider_type)}</dd>
            <dt>Base URL</dt>
            <dd>{config.base_url}</dd>
            <dt>API Key</dt>
            <dd>{config.api_key}</dd>
            <dt>Model</dt>
            <dd>{config.model}</dd>
            <dt>积分/次</dt>
            <dd>{config.credits_per_unit}</dd>
          </dl>
          <div className="provider-actions">
            <button type="button" className="action-btn secondary" onClick={() => setEditing(true)}>
              编辑
            </button>
            <button type="button" className="action-btn secondary" onClick={handleTest} disabled={testMutation.isPending}>
              {testMutation.isPending ? '测试中...' : '测试连接'}
            </button>
          </div>
          {testResult && (
            <div className={`test-result ${testResult.ok ? 'success' : 'failure'}`}>
              {testResult.ok
                ? `连接成功 · ${testResult.capability ?? ''}${testResult.provider_type ? ' / ' + testResult.provider_type : ''} · ${testResult.model} · ${testResult.latency_ms}ms${testResult.probe ? ' · ' + testResult.probe : ''}`
                : `失败: ${testResult.error}${testResult.probe ? '（probe: ' + testResult.probe + '）' : ''}`}
            </div>
          )}
        </div>
      ) : (
        <form className="provider-card-body" onSubmit={handleSave}>
          <label className="field-label">
            适配器
            <select
              value={providerType}
              onChange={(e) => setProviderType(e.target.value as ProviderType)}
            >
              {PROVIDER_TYPE_OPTIONS.filter((opt) => allowedTypes.includes(opt.value)).map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            <span className="field-hint">
              {PROVIDER_TYPE_OPTIONS.find((opt) => opt.value === providerType)?.hint}
            </span>
          </label>
          <label className="field-label">
            Base URL
            <input type="url" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} required placeholder="https://api.example.com/v1" />
          </label>
          <label className="field-label">
            API Key
            <input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder={config ? '留空保持不变' : 'sk-...'} required={!config} />
          </label>
          <label className="field-label">
            Model
            <input type="text" value={model} onChange={(e) => setModel(e.target.value)} required placeholder="deepseek-chat" />
          </label>
          <label className="field-label">
            积分/次
            <input type="number" value={creditsPerUnit} onChange={(e) => setCreditsPerUnit(Number(e.target.value))} min={0} />
          </label>
          <div className="provider-actions">
            <button type="submit" className="action-btn primary" disabled={saveMutation.isPending}>
              {saveMutation.isPending ? '保存中...' : '保存'}
            </button>
            {config && (
              <button type="button" className="action-btn secondary" onClick={() => setEditing(false)}>
                取消
              </button>
            )}
          </div>
          {saveMutation.isError && (
            <div className="test-result failure">保存失败: {saveMutation.error.message}</div>
          )}
        </form>
      )}
    </section>
  )
}

function providerTypeLabel(type: ProviderType): string {
  return PROVIDER_TYPE_OPTIONS.find((opt) => opt.value === type)?.label ?? type
}
