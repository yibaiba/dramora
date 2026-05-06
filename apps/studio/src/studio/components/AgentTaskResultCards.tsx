import { ArrowRight, Sparkles } from 'lucide-react'
import { Link } from 'react-router-dom'

import type { StoryAgentOutput } from '../../api/types'
import type { AgentTask } from '../../state/agentTaskCenterStore'
import { buildAgentFollowUpTarget, buildStructuredAgentOutput } from '../agentOutput'
import { agentRoleLabel } from '../utils'

type AgentTaskResultCardsProps = {
  task: AgentTask
}

function toStoryAgentStatus(status: AgentTask['status']): StoryAgentOutput['status'] {
  const mapping: Record<AgentTask['status'], StoryAgentOutput['status']> = {
    cancelled: 'skipped',
    done: 'succeeded',
    error: 'failed',
    running: 'running',
  }
  return mapping[status]
}

function toStoryAgentOutput(task: AgentTask): StoryAgentOutput {
  return {
    role: task.role,
    status: toStoryAgentStatus(task.status),
    output: task.doneFrame?.output || task.streamedText,
    highlights: task.doneFrame?.highlights ?? [],
  }
}

export function AgentTaskResultCards({ task }: AgentTaskResultCardsProps) {
  const output = task.doneFrame?.output || task.streamedText
  if (!output.trim()) {
    return (
      <div className="agent-task-result-empty">
        <Sparkles size={18} aria-hidden="true" />
        <div>
          <strong>等待结构化结果</strong>
          <p>收到足够的输出后，这里会自动提炼摘要、关键要点和结构化分块。</p>
        </div>
      </div>
    )
  }

  const agent = toStoryAgentOutput(task)
  const structured = buildStructuredAgentOutput(agent)
  const followUp = buildAgentFollowUpTarget(agent)

  return (
    <div className="agent-task-result-stack">
      <section className="agent-task-result-hero">
        <div className="agent-task-result-hero-head">
          <div>
            <span className="section-kicker">Structured result</span>
            <strong>{structured.summaryHeadline}</strong>
          </div>
          <span className="agent-task-result-role">{agentRoleLabel(task.role)}</span>
        </div>
        <p>{structured.summaryText}</p>
      </section>

      <section className="agent-task-result-section" aria-label="关键要点">
        <div className="agent-task-section-heading">
          <h3>关键要点</h3>
          <span>{structured.keyPoints.length} 条</span>
        </div>
        <div className="agent-task-keypoints">
          {structured.keyPoints.map((point) => (
            <span key={point} className="agent-task-keypoint">
              {point}
            </span>
          ))}
        </div>
      </section>

      <section className="agent-task-result-section" aria-label="结构化分块">
        <div className="agent-task-section-heading">
          <h3>结构化分块</h3>
          <span>{structured.blocks.length} 个卡片</span>
        </div>
        <div className="agent-task-result-grid">
          {structured.blocks.map((block) => (
            <article key={block.title} className="agent-task-result-card">
              <strong>{block.title}</strong>
              <ul>
                {block.entries.map((entry) => (
                  <li key={entry}>{entry}</li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </section>

      {followUp ? (
        <section className="agent-task-followup-card" aria-label="建议下一步">
          <div>
            <span className="section-kicker">Recommended next step</span>
            <strong>{followUp.actionLabel}</strong>
            <p>{followUp.description}</p>
          </div>
          <Link className="btn-ghost" to={followUp.path} state={followUp.state}>
            前往继续处理
            <ArrowRight size={16} aria-hidden="true" />
          </Link>
        </section>
      ) : null}
    </div>
  )
}
