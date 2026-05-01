import { useEffect, useMemo, useRef, useState } from 'react'
import { Sparkles, Plus, Trash2, Save, X } from 'lucide-react'
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

const SAVED_PRESETS_KEY = 'dramora-sandbox-context-presets'
const SAVED_HISTORY_KEY = 'dramora-sandbox-run-history'
const MAX_HISTORY_ENTRIES = 10

type SavedPreset = { id: string; label: string; entries: ContextEntry[] }

type RunHistoryEntry = {
  id: string
  role: string
  sourceText: string
  contextEntries: ContextEntry[]
  output: string
  durationMs?: number
  tokenCount?: number
  timestamp: string
}

function loadRunHistory(): RunHistoryEntry[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(SAVED_HISTORY_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((entry): entry is RunHistoryEntry => {
        if (!entry || typeof entry !== 'object') return false
        const obj = entry as Record<string, unknown>
        return (
          typeof obj.id === 'string' &&
          typeof obj.role === 'string' &&
          typeof obj.sourceText === 'string' &&
          Array.isArray(obj.contextEntries) &&
          typeof obj.output === 'string' &&
          typeof obj.timestamp === 'string'
        )
      })
      .map((entry) => ({
        id: entry.id,
        role: entry.role,
        sourceText: entry.sourceText,
        contextEntries: entry.contextEntries
          .filter((e): e is ContextEntry =>
            !!e && typeof e === 'object' && typeof (e as ContextEntry).key === 'string',
          )
          .map((e) => ({ key: String(e.key), value: String(e.value ?? '') })),
        output: entry.output,
        durationMs: typeof entry.durationMs === 'number' ? entry.durationMs : undefined,
        tokenCount: typeof entry.tokenCount === 'number' ? entry.tokenCount : undefined,
        timestamp: entry.timestamp,
      }))
      .slice(0, MAX_HISTORY_ENTRIES)
  } catch {
    return []
  }
}

function persistRunHistory(history: RunHistoryEntry[]) {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(SAVED_HISTORY_KEY, JSON.stringify(history))
  } catch {
    /* quota exceeded — ignore */
  }
}

function loadSavedPresets(): SavedPreset[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(SAVED_PRESETS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((p): p is SavedPreset => {
        if (!p || typeof p !== 'object') return false
        const obj = p as Record<string, unknown>
        return (
          typeof obj.id === 'string' &&
          typeof obj.label === 'string' &&
          Array.isArray(obj.entries)
        )
      })
      .map((p) => ({
        id: p.id,
        label: p.label,
        entries: p.entries
          .filter((e): e is ContextEntry =>
            !!e && typeof e === 'object' && typeof (e as ContextEntry).key === 'string',
          )
          .map((e) => ({ key: String(e.key), value: String(e.value ?? '') })),
      }))
  } catch {
    return []
  }
}

function persistSavedPresets(presets: SavedPreset[]) {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(SAVED_PRESETS_KEY, JSON.stringify(presets))
  } catch {
    /* quota exceeded or storage disabled — ignore */
  }
}

type DiffLine = { kind: 'eq' | 'add' | 'del'; left?: string; right?: string }

function computeLineDiff(a: string, b: string): DiffLine[] {
  const al = a.split('\n')
  const bl = b.split('\n')
  const m = al.length
  const n = bl.length
  const dp: number[][] = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0))
  for (let i = m - 1; i >= 0; i--) {
    for (let j = n - 1; j >= 0; j--) {
      dp[i][j] = al[i] === bl[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const out: DiffLine[] = []
  let i = 0
  let j = 0
  while (i < m && j < n) {
    if (al[i] === bl[j]) {
      out.push({ kind: 'eq', left: al[i], right: bl[j] })
      i++
      j++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      out.push({ kind: 'del', left: al[i] })
      i++
    } else {
      out.push({ kind: 'add', right: bl[j] })
      j++
    }
  }
  while (i < m) {
    out.push({ kind: 'del', left: al[i++] })
  }
  while (j < n) {
    out.push({ kind: 'add', right: bl[j++] })
  }
  return out
}

export function AgentStreamSandbox() {
  const [role, setRole] = useState('story_analyst')
  const [sourceText, setSourceText] = useState('小镇雨夜，少年送伞给陌生人。')
  const [contextEntries, setContextEntries] = useState<ContextEntry[]>([])
  const [streamedText, setStreamedText] = useState('')
  const [doneFrame, setDoneFrame] = useState<AgentStreamDoneFrame | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const [savedPresets, setSavedPresets] = useState<SavedPreset[]>(() => loadSavedPresets())
  const [isSavingPreset, setIsSavingPreset] = useState(false)
  const [newPresetLabel, setNewPresetLabel] = useState('')
  const [history, setHistory] = useState<RunHistoryEntry[]>(() => loadRunHistory())
  const [isHistoryOpen, setIsHistoryOpen] = useState(false)
  const [diffSelection, setDiffSelection] = useState<string[]>([])
  const abortRef = useRef<AbortController | null>(null)

  const toggleDiffSelection = (id: string) => {
    setDiffSelection((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id)
      if (prev.length >= 2) return [prev[1], id]
      return [...prev, id]
    })
  }

  const diffEntries = useMemo(
    () => diffSelection.map((id) => history.find((h) => h.id === id)).filter((x): x is RunHistoryEntry => Boolean(x)),
    [diffSelection, history],
  )

  useEffect(() => {
    persistSavedPresets(savedPresets)
  }, [savedPresets])

  useEffect(() => {
    persistRunHistory(history)
  }, [history])

  const saveCurrentAsPreset = () => {
    const label = newPresetLabel.trim()
    if (!label) return
    const entries = contextEntries
      .map((e) => ({ key: e.key.trim(), value: e.value }))
      .filter((e) => e.key !== '')
    if (entries.length === 0) return
    setSavedPresets((prev) => [
      ...prev.filter((p) => p.label !== label),
      { id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`, label, entries },
    ])
    setNewPresetLabel('')
    setIsSavingPreset(false)
  }
  const removeSavedPreset = (id: string) =>
    setSavedPresets((prev) => prev.filter((p) => p.id !== id))

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
    let accumulated = ''
    const snapshotRole = role
    const snapshotSourceText = sourceText
    const snapshotEntries = contextEntries.map((e) => ({ key: e.key, value: e.value }))
    try {
      await streamAgentRun(
        { role, source_text: sourceText, context: buildContextMap() },
        {
          onDelta: (chunk) => {
            accumulated += chunk
            setStreamedText((prev) => prev + chunk)
          },
          onDone: (frame) => {
            setDoneFrame(frame)
            setHistory((prev) => {
              const next: RunHistoryEntry = {
                id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
                role: snapshotRole,
                sourceText: snapshotSourceText,
                contextEntries: snapshotEntries,
                output: accumulated,
                durationMs: frame.duration_ms,
                tokenCount: frame.token_count,
                timestamp: new Date().toISOString(),
              }
              return [next, ...prev].slice(0, MAX_HISTORY_ENTRIES)
            })
          },
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

  const restoreFromHistory = (entry: RunHistoryEntry) => {
    setRole(entry.role)
    setSourceText(entry.sourceText)
    setContextEntries(entry.contextEntries.map((e) => ({ ...e })))
  }

  const removeHistoryEntry = (id: string) =>
    setHistory((prev) => prev.filter((entry) => entry.id !== id))

  const clearHistory = () => setHistory([])

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
            {savedPresets.map((preset) => (
              <span
                key={preset.id}
                style={{ display: 'inline-flex', alignItems: 'center', gap: 2 }}
              >
                <button
                  type="button"
                  onClick={() => applyPreset(preset.entries)}
                  disabled={isStreaming}
                  className="btn-ghost"
                  style={{ fontSize: 11 }}
                  title={`${preset.entries.length} 项`}
                >
                  ★ {preset.label}
                </button>
                <button
                  type="button"
                  onClick={() => removeSavedPreset(preset.id)}
                  disabled={isStreaming}
                  className="btn-ghost"
                  style={{ fontSize: 11, padding: '2px 6px' }}
                  title="删除此预设"
                  aria-label={`删除预设 ${preset.label}`}
                >
                  <X size={11} />
                </button>
              </span>
            ))}
            {contextEntries.length > 0 && !isSavingPreset && (
              <button
                type="button"
                onClick={() => setIsSavingPreset(true)}
                disabled={isStreaming}
                className="btn-ghost"
                style={{ fontSize: 11, display: 'inline-flex', alignItems: 'center', gap: 4 }}
              >
                <Save size={11} /> 保存为预设
              </button>
            )}
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
          {isSavingPreset && (
            <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
              <input
                type="text"
                value={newPresetLabel}
                onChange={(e) => setNewPresetLabel(e.target.value)}
                placeholder="预设名称（如 director 依赖 screenwriter）"
                disabled={isStreaming}
                className="input"
                style={{ flex: '1 1 200px', fontSize: 12 }}
                autoFocus
              />
              <button
                type="button"
                onClick={saveCurrentAsPreset}
                disabled={isStreaming || !newPresetLabel.trim()}
                className="btn"
                style={{ fontSize: 11 }}
              >
                保存
              </button>
              <button
                type="button"
                onClick={() => {
                  setIsSavingPreset(false)
                  setNewPresetLabel('')
                }}
                disabled={isStreaming}
                className="btn-ghost"
                style={{ fontSize: 11 }}
              >
                取消
              </button>
            </div>
          )}
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
      {history.length > 0 && (
        <div className="sandbox-history">
          <button
            type="button"
            className="btn-ghost"
            onClick={() => setIsHistoryOpen((prev) => !prev)}
            aria-expanded={isHistoryOpen}
            style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12 }}
          >
            最近 {history.length} 条运行历史 {isHistoryOpen ? '▲' : '▼'}
          </button>
          {isHistoryOpen && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 8 }}>
              <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                <button type="button" className="btn-ghost" onClick={clearHistory} style={{ fontSize: 11 }}>
                  清空历史
                </button>
              </div>
              {history.map((entry) => {
                const roleLabel =
                  ROLE_OPTIONS.find((opt) => opt.value === entry.role)?.label ?? entry.role
                return (
                  <article
                    key={entry.id}
                    style={{
                      border: '1px solid rgba(255,255,255,0.08)',
                      borderRadius: 8,
                      padding: 10,
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 6,
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'baseline',
                        gap: 8,
                      }}
                    >
                      <strong style={{ fontSize: 13 }}>{roleLabel}</strong>
                      <small style={{ opacity: 0.6 }}>
                        {new Date(entry.timestamp).toLocaleTimeString()}
                        {typeof entry.durationMs === 'number' ? ` · ${entry.durationMs}ms` : ''}
                        {typeof entry.tokenCount === 'number' ? ` · ${entry.tokenCount} tok` : ''}
                      </small>
                    </div>
                    <div style={{ fontSize: 12, opacity: 0.75 }}>
                      源：{entry.sourceText.slice(0, 80)}
                      {entry.sourceText.length > 80 ? '…' : ''}
                    </div>
                    {entry.contextEntries.length > 0 && (
                      <div style={{ fontSize: 11, opacity: 0.55 }}>
                        context: {entry.contextEntries.map((e) => e.key).join(', ')}
                      </div>
                    )}
                    <pre
                      style={{
                        margin: 0,
                        padding: 6,
                        background: 'rgba(255,255,255,0.03)',
                        borderRadius: 4,
                        fontSize: 11,
                        maxHeight: 80,
                        overflow: 'auto',
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      {entry.output.slice(0, 240)}
                      {entry.output.length > 240 ? '…' : ''}
                    </pre>
                    <div style={{ display: 'flex', gap: 6 }}>
                      <label
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: 4,
                          fontSize: 11,
                          opacity: 0.8,
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={diffSelection.includes(entry.id)}
                          onChange={() => toggleDiffSelection(entry.id)}
                        />
                        对比
                      </label>
                      <button
                        type="button"
                        className="btn-ghost"
                        onClick={() => restoreFromHistory(entry)}
                        disabled={isStreaming}
                        style={{ fontSize: 11 }}
                      >
                        载入到表单
                      </button>
                      <button
                        type="button"
                        className="btn-ghost"
                        onClick={() => removeHistoryEntry(entry.id)}
                        style={{ fontSize: 11 }}
                      >
                        删除
                      </button>
                    </div>
                  </article>
                )
              })}
            </div>
          )}
        </div>
      )}
      {diffEntries.length > 0 && (
        <div className="sandbox-diff" style={{ marginTop: 12 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'baseline',
              justifyContent: 'space-between',
              gap: 8,
            }}
          >
            <strong style={{ fontSize: 13 }}>
              输出对比 ({diffEntries.length}/2)
            </strong>
            <button
              type="button"
              className="btn-ghost"
              onClick={() => setDiffSelection([])}
              style={{ fontSize: 11 }}
            >
              清除对比
            </button>
          </div>
          {diffEntries.length === 1 && (
            <div style={{ fontSize: 11, opacity: 0.6, marginTop: 6 }}>
              再勾选一条历史记录即可生成 diff。
            </div>
          )}
          {diffEntries.length === 2 && (
            <div style={{ marginTop: 8 }}>
              <div style={{ display: 'flex', gap: 8, fontSize: 11, opacity: 0.7, marginBottom: 4 }}>
                <span style={{ flex: 1 }}>
                  A · {new Date(diffEntries[0].timestamp).toLocaleTimeString()} ·{' '}
                  {ROLE_OPTIONS.find((o) => o.value === diffEntries[0].role)?.label ?? diffEntries[0].role}
                </span>
                <span style={{ flex: 1 }}>
                  B · {new Date(diffEntries[1].timestamp).toLocaleTimeString()} ·{' '}
                  {ROLE_OPTIONS.find((o) => o.value === diffEntries[1].role)?.label ?? diffEntries[1].role}
                </span>
              </div>
              <div
                style={{
                  border: '1px solid rgba(255,255,255,0.08)',
                  borderRadius: 6,
                  maxHeight: 320,
                  overflow: 'auto',
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                  fontSize: 11,
                }}
              >
                {computeLineDiff(diffEntries[0].output, diffEntries[1].output).map((line, idx) => {
                  const bg =
                    line.kind === 'add'
                      ? 'rgba(74, 222, 128, 0.08)'
                      : line.kind === 'del'
                      ? 'rgba(248, 113, 113, 0.08)'
                      : 'transparent'
                  return (
                    <div
                      key={idx}
                      style={{
                        display: 'grid',
                        gridTemplateColumns: '1fr 1fr',
                        background: bg,
                        borderBottom: '1px solid rgba(255,255,255,0.03)',
                      }}
                    >
                      <pre
                        style={{
                          margin: 0,
                          padding: '2px 6px',
                          whiteSpace: 'pre-wrap',
                          color: line.kind === 'del' ? '#fca5a5' : line.kind === 'eq' ? undefined : 'transparent',
                        }}
                      >
                        {line.left ?? ''}
                      </pre>
                      <pre
                        style={{
                          margin: 0,
                          padding: '2px 6px',
                          whiteSpace: 'pre-wrap',
                          borderLeft: '1px solid rgba(255,255,255,0.05)',
                          color: line.kind === 'add' ? '#86efac' : line.kind === 'eq' ? undefined : 'transparent',
                        }}
                      >
                        {line.right ?? ''}
                      </pre>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
