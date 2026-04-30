import { Check, Copy, FileText, Sparkles } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import type { StoryAgentOutput } from '../../api/types'
import type { AgentFollowUpFeedback, AgentOutputFollowUpTarget } from '../agentOutput'
import { buildStructuredAgentOutput } from '../agentOutput'
import { agentRoleLabel, agentStatusLabel } from '../utils'

type AgentOutputPanelProps = {
  agent: StoryAgentOutput
  feedbackState?: AgentFollowUpFeedback
  followUpTarget?: AgentOutputFollowUpTarget | null
  onClose: () => void
  onInsertIntoSource?: (agent: StoryAgentOutput) => void
  onSetFeedback?: (agent: StoryAgentOutput, feedback: AgentFollowUpFeedback) => void
}

type OutputTab = 'structured' | 'raw'

export function AgentOutputPanel({
  agent,
  feedbackState,
  followUpTarget,
  onClose,
  onInsertIntoSource,
  onSetFeedback,
}: AgentOutputPanelProps) {
  const [tab, setTab] = useState<OutputTab>('structured')
  const [panelFeedback, setPanelFeedback] = useState<string>('')
  const structuredOutput = useMemo(() => buildStructuredAgentOutput(agent), [agent])

  const handleCopy = async (mode: 'summary' | 'full') => {
    if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) {
      setPanelFeedback('当前环境不支持剪贴板复制，请手动复制。')
      return
    }

    try {
      await navigator.clipboard.writeText(
        mode === 'summary' ? structuredOutput.summaryClipboard : agent.output.trim(),
      )
      setPanelFeedback(mode === 'summary' ? '已复制重点摘要。' : '已复制完整输出。')
    } catch {
      setPanelFeedback('复制失败，请检查浏览器剪贴板权限。')
    }
  }

  const handleFeedback = (feedback: AgentFollowUpFeedback) => {
    onSetFeedback?.(agent, feedback)
    setPanelFeedback(feedback === 'adopted' ? '已标记为已采纳。' : '已标记为待跟进。')
  }

  const handleInsert = () => {
    onInsertIntoSource?.(agent)
    setPanelFeedback(`已把 ${agentRoleLabel(agent.role)} 的摘要回填到故事源补充区。`)
  }

  return (
    <aside className="agent-output-panel" aria-label={`${agentRoleLabel(agent.role)} 产出详情`}>
      <div className="agent-output-header">
        <div className="agent-output-title-block">
          <div>
            <span className="section-kicker">Agent output</span>
            <strong>{agentRoleLabel(agent.role)}</strong>
          </div>
          <span className={`agent-output-status-chip ${agent.status}`}>
            {agentStatusLabel(agent.status)}
          </span>
        </div>
        <div className="agent-output-toolbar">
          <div className="output-tabs" aria-label="输出视图切换">
            <button
              type="button"
              className={tab === 'structured' ? 'active' : ''}
              onClick={() => setTab('structured')}
            >
              结构化产出
            </button>
            <button
              type="button"
              className={tab === 'raw' ? 'active' : ''}
              onClick={() => setTab('raw')}
            >
              原始输出
            </button>
          </div>
          <div className="agent-output-actions">
            <button type="button" className="action-btn secondary" onClick={() => handleCopy('summary')}>
              <Sparkles aria-hidden="true" size={14} />
              复制摘要
            </button>
            <button type="button" className="action-btn secondary" onClick={() => handleCopy('full')}>
              <Copy aria-hidden="true" size={14} />
              复制全部
            </button>
            <button type="button" className="action-btn secondary" onClick={onClose}>
              关闭
            </button>
          </div>
        </div>
      </div>
      <div className="agent-output-body">
        <div className="agent-output-overview-grid">
          <article className="agent-output-overview-card">
            <span>状态</span>
            <strong>{agentStatusLabel(agent.status)}</strong>
            <small>当前节点执行结果</small>
          </article>
          <article className="agent-output-overview-card">
            <span>高亮</span>
            <strong>{agent.highlights.length}</strong>
            <small>已提炼重点条目</small>
          </article>
          <article className="agent-output-overview-card">
            <span>结构块</span>
            <strong>{structuredOutput.blocks.length}</strong>
            <small>当前可读分段</small>
          </article>
        </div>

        <div className="agent-output-feedback" aria-live="polite">
          {panelFeedback ? (
            <>
              <Check aria-hidden="true" size={14} />
              <span>{panelFeedback}</span>
            </>
          ) : (
            <>
              <FileText aria-hidden="true" size={14} />
              <span>支持一键复制重点摘要或完整输出，便于回填 Prompt、任务卡和评审记录。</span>
            </>
          )}
        </div>

        {tab === 'structured' ? (
          <div className="agent-output-structured">
            <section className="agent-output-section agent-output-summary-card">
              <div className="agent-output-section-head">
                <span className="section-kicker">Key summary</span>
                <strong>{structuredOutput.summaryHeadline}</strong>
              </div>
              <p className="agent-output-summary-text">{structuredOutput.summaryText}</p>
            </section>

            {agent.highlights.length > 0 && (
              <div className="agent-output-tags" aria-label="Agent 高亮标签">
                {agent.highlights.map((highlight) => (
                  <span key={highlight} className="agent-output-tag">
                    {highlight}
                  </span>
                ))}
              </div>
            )}

            {structuredOutput.keyPoints.length > 0 && (
              <section className="agent-output-section">
                <div className="agent-output-section-head">
                  <span className="section-kicker">Focus points</span>
                  <strong>重点摘要</strong>
                </div>
                <ul className="agent-output-point-list">
                  {structuredOutput.keyPoints.map((point) => (
                    <li key={point}>{point}</li>
                  ))}
                </ul>
              </section>
            )}

            <section className="agent-output-section">
              <div className="agent-output-section-head">
                <span className="section-kicker">Follow-up actions</span>
                <strong>后续动作</strong>
              </div>
              <p className="agent-output-follow-up-copy">
                {followUpTarget?.description ??
                  '当前输出可先回填到故事源补充区，再决定是否继续跳到下游工作台。'}
              </p>
              <div className="agent-output-follow-up-row">
                <button
                  type="button"
                  aria-pressed={feedbackState === 'adopted'}
                  className={feedbackState === 'adopted' ? 'asset-filter-chip active' : 'asset-filter-chip'}
                  onClick={() => handleFeedback('adopted')}
                >
                  标记已采纳
                </button>
                <button
                  type="button"
                  aria-pressed={feedbackState === 'needs_follow_up'}
                  className={
                    feedbackState === 'needs_follow_up'
                      ? 'asset-filter-chip active'
                      : 'asset-filter-chip'
                  }
                  onClick={() => handleFeedback('needs_follow_up')}
                >
                  标记待跟进
                </button>
                <button type="button" className="ghost-action" onClick={handleInsert}>
                  回填到故事源补充区
                </button>
                {followUpTarget ? (
                  <Link
                    className="asset-filter-chip node-preview-inline-action"
                    state={followUpTarget.state}
                    to={followUpTarget.path}
                  >
                    {followUpTarget.actionLabel}
                  </Link>
                ) : null}
              </div>
            </section>

            <div className="agent-output-structured-grid">
              {structuredOutput.blocks.map((block) => (
                <section className="agent-output-section" key={block.title}>
                  <div className="agent-output-section-head">
                    <span className="section-kicker">Structured block</span>
                    <strong>{block.title}</strong>
                  </div>
                  <ul className="agent-output-block-list">
                    {block.entries.map((entry) => (
                      <li key={entry}>{entry}</li>
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          </div>
        ) : (
          <pre className="agent-output-raw">{agent.output}</pre>
        )}
      </div>
    </aside>
  )
}
