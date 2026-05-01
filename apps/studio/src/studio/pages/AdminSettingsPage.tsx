import { useState } from 'react'
import type { FormEvent } from 'react'
import { Settings, Zap, Image, Video, Volume2, Activity } from 'lucide-react'
import { useLLMTelemetry, useProviderConfigs, useSaveProviderConfig, useTestProviderConfig } from '../../api/hooks'
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
    </div>
  )
}

function LLMTelemetryPanel() {
  const { data, isLoading, isError, error } = useLLMTelemetry()
  return (
    <section className="provider-card" style={{ marginTop: 24 }}>
      <header className="provider-card-header">
        <Activity size={18} aria-hidden="true" />
        <h2>LLM 调用 Telemetry</h2>
        <span className="provider-card-hint">每 10s 刷新 · 进程内最近 50 条</span>
      </header>
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
                sub={`${count} 次`}
              />
            ))}
          </div>
          {data.recent_events && data.recent_events.length > 0 ? (
            <div style={{ overflowX: 'auto' }}>
              <table className="telemetry-table" style={{ width: '100%', fontSize: 12 }}>
                <thead>
                  <tr>
                    <th style={{ textAlign: 'left' }}>时间</th>
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
                ? `连接成功 · ${testResult.model} · ${testResult.latency_ms}ms`
                : `失败: ${testResult.error}`}
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
