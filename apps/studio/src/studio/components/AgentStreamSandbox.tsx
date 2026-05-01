import { useRef, useState } from 'react'
import { Sparkles, Plus, Trash2 } from 'lucide-react'
import { streamAgentRun, type AgentStreamDoneFrame } from '../../api/agentStream'

const ROLE_OPTIONS = [
  { value: 'story_analyst', label: '故事分析师' },
  { value: 'outline_planner', label: '大纲规划师' },
  { value: 'character_analyst', label: '角色分析师' },
  { value: 'scene_analyst', label: '场景分析师' },
  { value: 'prop_analyst', label: '道具分析师' },
  { value: 'screenwriter', label: '编剧' },
  { value: 'director', label: '导演' },
  { value: 'cinematographer', label: '摄影指导' },
  { value: 'voice_subtitle', label: '配音导演' },
]

type ContextEntry = { key: string; value: string }

const CONTEXT_PRESETS: { label: string; entries: ContextEntry[] }[] = [
  {
    label: 'outline_planner 依赖 story_analyst',
    entries: [
      { key: 'story_analyst_output', value: '主线：少年送伞与陌生人结成短暂同行；冲突：陌生人身份悬疑。' },
    ],
  },
  {
    label: 'screenwriter 依赖 outline_planner',
    entries: [
      {
        key: 'outline_planner_output',
        value: '幕一：雨夜偶遇；幕二：陌生人讲述自身故事；幕三：分别与回望。',
      },
    ],
  },
]

export function AgentStreamSandbox() {
  const [role, setRole] = useState('story_analyst')
  const [sourceText, setSourceText] = useState('小镇雨夜，少年送伞给陌生人。')
  const [contextEntries, setContextEntries] = useState<ContextEntry[]>([])
  const [streamedText, setStreamedText] = useState('')
  const [doneFrame, setDoneFrame] = useState<AgentStreamDoneFrame | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  const updateEntry = (idx: number, patch: Partial<ContextEntry>) => {
    setContextEntries((prev) => prev.map((entry, i) => (i === idx ? { ...entry, ...patch } : entry)))
  }
  const addEntry = () => setContextEntries((prev) => [...prev, { key: '', value: '' }])
  const removeEntry = (idx: number) =>
    setContextEntries((prev) => prev.filter((_, i) => i !== idx))
  const applyPreset = (entries: ContextEntry[]) => setContextEntries(entries)

  const buildContextMap = (): Record<string, string> | undefined => {
    const trimmed = contextEntries
      .map((entry) => ({ key: entry.key.trim(), value: entry.value }))
      .filter((entry) => entry.key !== '')
    if (trimmed.length === 0) return undefined
    const map: Record<string, string> = {}
    for (const entry of trimmed) {
      map[entry.key] = entry.value
    }
    return map
  }

  const handleRun = async () => {
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setStreamedText('')
    setDoneFrame(null)
    setErrorMessage(null)
    setIsStreaming(true)
    try {
      await streamAgentRun(
        { role, source_text: sourceText, context: buildContextMap() },
        {
          onDelta: (chunk) => setStreamedText((prev) => prev + chunk),
          onDone: (frame) => setDoneFrame(frame),
          onError: (message) => setErrorMessage(message),
        },
        controller.signal,
      )
    } catch (err) {
      if (controller.signal.aborted) return
      setErrorMessage(err instanceof Error ? err.message : 'stream failed')
    } finally {
      setIsStreaming(false)
    }
  }

  const handleStop = () => {
    abortRef.current?.abort()
    setIsStreaming(false)
  }

  return (
    <section className="agent-stream-sandbox" aria-label="Agent 流式调试">
      <header className="sandbox-header">
        <Sparkles size={18} aria-hidden="true" />
        <h2>Agent 流式调试</h2>
        <p className="page-subtitle">
          直接调用 <code>POST /api/v1/agents/stream</code>，验证已配置 chat 端点的 LLM 响应与流式输出。
        </p>
      </header>
      <div className="sandbox-controls">
        <label>
          <span>Agent 角色</span>
          <select value={role} onChange={(e) => setRole(e.target.value)} disabled={isStreaming}>
            {ROLE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>故事源文本</span>
          <textarea
            value={sourceText}
            onChange={(e) => setSourceText(e.target.value)}
            rows={4}
            disabled={isStreaming}
          />
        </label>
        <fieldset
          style={{
            border: '1px solid rgba(255,255,255,0.08)',
            borderRadius: 8,
            padding: 12,
            display: 'flex',
            flexDirection: 'column',
            gap: 8,
          }}
        >
          <legend style={{ padding: '0 6px', fontSize: 12, opacity: 0.7 }}>
            上游 Agent 输出（context map） · 可选
          </legend>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {CONTEXT_PRESETS.map((preset) => (
              <button
                key={preset.label}
                type="button"
                onClick={() => applyPreset(preset.entries)}
                disabled={isStreaming}
                className="btn-ghost"
                style={{ fontSize: 11 }}
              >
                填入：{preset.label}
              </button>
            ))}
            {contextEntries.length > 0 && (
              <button
                type="button"
                onClick={() => setContextEntries([])}
                disabled={isStreaming}
                className="btn-ghost"
                style={{ fontSize: 11 }}
              >
                清空
              </button>
            )}
          </div>
          {contextEntries.length === 0 ? (
            <p className="muted" style={{ fontSize: 12, margin: 0 }}>
              还未填入任何上游输出。例如 <code>outline_planner</code> 等需要 <code>story_analyst_output</code> 时，
              在此添加 key=<code>story_analyst_output</code>、value=前序 Agent 的输出文本。
            </p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {contextEntries.map((entry, idx) => (
                <div key={idx} style={{ display: 'flex', gap: 6, alignItems: 'flex-start' }}>
                  <input
                    type="text"
                    placeholder="key 例如 story_analyst_output"
                    value={entry.key}
                    onChange={(e) => updateEntry(idx, { key: e.target.value })}
                    disabled={isStreaming}
                    style={{ flex: '0 0 220px', fontSize: 12 }}
                  />
                  <textarea
                    placeholder="上游 Agent 的输出文本"
                    value={entry.value}
                    onChange={(e) => updateEntry(idx, { value: e.target.value })}
                    disabled={isStreaming}
                    rows={2}
                    style={{ flex: 1, fontSize: 12 }}
                  />
                  <button
                    type="button"
                    onClick={() => removeEntry(idx)}
                    disabled={isStreaming}
                    aria-label="移除该条 context"
                    className="btn-ghost"
                    style={{ padding: 6 }}
                  >
                    <Trash2 size={14} aria-hidden="true" />
                  </button>
                </div>
              ))}
            </div>
          )}
          <button
            type="button"
            onClick={addEntry}
            disabled={isStreaming}
            className="btn-ghost"
            style={{ alignSelf: 'flex-start', fontSize: 12, display: 'inline-flex', alignItems: 'center', gap: 4 }}
          >
            <Plus size={14} aria-hidden="true" />
            新增 context 项
          </button>
        </fieldset>
        <div className="sandbox-actions">
          <button
            type="button"
            className="primary"
            onClick={handleRun}
            disabled={isStreaming || !sourceText.trim()}
          >
            {isStreaming ? '流式输出中…' : '运行并流式输出'}
          </button>
          {isStreaming && (
            <button type="button" onClick={handleStop}>
              中止
            </button>
          )}
        </div>
      </div>
      {errorMessage && <div className="test-result failure">错误：{errorMessage}</div>}
      {(streamedText || doneFrame) && (
        <div className="sandbox-output">
          <h3>实时输出</h3>
          <pre className="stream-text">{streamedText || '（等待第一个 token…）'}</pre>
          {doneFrame && (
            <dl className="stream-meta">
              <div>
                <dt>耗时</dt>
                <dd>{doneFrame.duration_ms} ms</dd>
              </div>
              <div>
                <dt>Token</dt>
                <dd>{doneFrame.token_count}</dd>
              </div>
              {doneFrame.highlights?.length > 0 && (
                <div>
                  <dt>Highlights</dt>
                  <dd>{doneFrame.highlights.join('、')}</dd>
                </div>
              )}
            </dl>
          )}
        </div>
      )}
    </section>
  )
}
