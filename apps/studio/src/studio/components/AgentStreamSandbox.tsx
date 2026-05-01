import { useRef, useState } from 'react'
import { Sparkles } from 'lucide-react'
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

export function AgentStreamSandbox() {
  const [role, setRole] = useState('story_analyst')
  const [sourceText, setSourceText] = useState('小镇雨夜，少年送伞给陌生人。')
  const [streamedText, setStreamedText] = useState('')
  const [doneFrame, setDoneFrame] = useState<AgentStreamDoneFrame | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

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
        { role, source_text: sourceText },
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
