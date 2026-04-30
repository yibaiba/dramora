import { Activity } from 'lucide-react'
import type { StoryAgentOutput, WorkflowRun } from '../../api/types'
import { agentRoleLabel, agentStatusLabel, formatCheckpointSavedAt, workflowRunStatusLabel } from '../utils'

const pipelineNodes = [
  { id: 'story_analyst', x: 20, y: 50 },
  { id: 'outline_planner', x: 140, y: 50 },
  { id: 'character_analyst', x: 260, y: 15 },
  { id: 'scene_analyst', x: 260, y: 50 },
  { id: 'prop_analyst', x: 260, y: 85 },
  { id: 'screenwriter', x: 380, y: 50 },
  { id: 'director', x: 500, y: 15 },
  { id: 'cinematographer', x: 500, y: 50 },
  { id: 'voice_subtitle', x: 500, y: 85 },
] as const

const pipelineEdges = [
  { from: 'story_analyst', to: 'outline_planner' },
  { from: 'outline_planner', to: 'character_analyst' },
  { from: 'outline_planner', to: 'scene_analyst' },
  { from: 'outline_planner', to: 'prop_analyst' },
  { from: 'character_analyst', to: 'screenwriter' },
  { from: 'scene_analyst', to: 'screenwriter' },
  { from: 'prop_analyst', to: 'screenwriter' },
  { from: 'screenwriter', to: 'director' },
  { from: 'screenwriter', to: 'cinematographer' },
  { from: 'screenwriter', to: 'voice_subtitle' },
] as const

type AgentBoardProps = {
  agents: StoryAgentOutput[]
  onSelectAgent: (agent: StoryAgentOutput) => void
  expandedRole?: string
  workflowRun?: WorkflowRun
}

export function AgentBoard({ agents, onSelectAgent, expandedRole, workflowRun }: AgentBoardProps) {
  const succeeded = agents.filter((a) => a.status === 'succeeded').length
  const checkpointSummary = workflowRun?.checkpoint_summary
  return (
    <section className="agent-board" aria-label="Agent 执行看板">
      <div className="panel-title-row">
        <div>
          <span className="section-kicker">Agent 执行看板</span>
          <strong>
            {succeeded}/{agents.length} 完成
          </strong>
        </div>
      </div>
      {checkpointSummary ? (
        <div className="agent-board-meta" aria-label="Checkpoint 恢复摘要">
          <span className="agent-board-meta-pill">
            {workflowRunStatusLabel(workflowRun.status)} · 快照 #{checkpointSummary.sequence}
          </span>
          <span className="agent-board-meta-pill">{formatCheckpointSavedAt(checkpointSummary.saved_at)}</span>
          <span className="agent-board-meta-pill">待执行 {checkpointSummary.waiting_nodes}</span>
          <span className="agent-board-meta-pill">失败 {checkpointSummary.failed_nodes}</span>
        </div>
      ) : null}
      <div className="agent-card-grid">
        {agents.map((agent) => (
          <AgentCard
            key={agent.role}
            agent={agent}
            expanded={expandedRole === agent.role}
            onSelect={() => onSelectAgent(agent)}
          />
        ))}
      </div>
    </section>
  )
}

function AgentCard({
  agent,
  expanded,
  onSelect,
}: {
  agent: StoryAgentOutput
  expanded: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      className={`agent-card ${agent.status} ${expanded ? 'expanded' : ''}`}
      onClick={onSelect}
      aria-expanded={expanded}
    >
      <div className="agent-card-header">
        <span className={`status-dot agent-${agent.status}`} />
        <strong>{agentRoleLabel(agent.role)}</strong>
        <span className="agent-status-label">{agentStatusLabel(agent.status)}</span>
      </div>
      {agent.highlights.length > 0 && (
        <div className="agent-highlights">
          {agent.highlights.slice(0, 3).map((h, i) => (
            <span key={i} className="agent-highlight-tag">
              {h}
            </span>
          ))}
        </div>
      )}
    </button>
  )
}

export function AgentPipeline({
  agents,
  onSelectAgent,
  selectedRole,
}: {
  agents: StoryAgentOutput[]
  onSelectAgent?: (agent: StoryAgentOutput) => void
  selectedRole?: string
}) {
  const agentMap = new Map(agents.map((a) => [a.role, a]))
  const visibleNodes = pipelineNodes.filter((node) => agentMap.has(node.id))
  const visibleEdges = pipelineEdges.filter(
    (e) => agentMap.has(e.from) && agentMap.has(e.to),
  )
  const relatedRoles = new Set(
    selectedRole
      ? visibleEdges.flatMap((edge) =>
          edge.from === selectedRole || edge.to === selectedRole
            ? [edge.from, edge.to]
            : [],
        )
      : [],
  )
  const statusCounts = agents.reduce<Record<string, number>>((counts, agent) => {
    counts[agent.status] = (counts[agent.status] ?? 0) + 1
    return counts
  }, {})

  return (
    <div className="agent-pipeline" aria-label="Agent DAG 流水线">
      <div className="pipeline-legend" aria-label="DAG 状态图例">
        <span className="pipeline-legend-chip">总节点 {visibleNodes.length}</span>
        {['running', 'waiting', 'succeeded', 'failed'].map((status) =>
          statusCounts[status] ? (
            <span className={`pipeline-legend-chip ${status}`} key={status}>
              {agentStatusLabel(status)} {statusCounts[status]}
            </span>
          ) : null,
        )}
        <span className="pipeline-legend-copy">
          {selectedRole
            ? `当前聚焦 ${agentRoleLabel(selectedRole)}，已高亮相邻依赖。`
            : '点击任一 Agent 节点即可查看输出并追踪上下游依赖。'}
        </span>
      </div>
      <svg viewBox="0 0 620 110" className="pipeline-svg">
        {visibleEdges.map((edge) => {
          const from = pipelineNodes.find((node) => node.id === edge.from)!
          const to = pipelineNodes.find((node) => node.id === edge.to)!
          const highlighted =
            !selectedRole || edge.from === selectedRole || edge.to === selectedRole
          return (
            <line
              key={`${edge.from}-${edge.to}`}
              x1={from.x + 50}
              y1={from.y + 10}
              x2={to.x}
              y2={to.y + 10}
              className={highlighted ? 'pipeline-edge active' : 'pipeline-edge inactive'}
            />
          )
        })}
        {visibleNodes.map((node) => {
          const agent = agentMap.get(node.id)
          const isSelected = selectedRole === node.id
          const isRelated = !isSelected && relatedRoles.has(node.id)
          return (
            <g
              aria-label={`${agentRoleLabel(node.id)} · ${agentStatusLabel(agent?.status ?? 'waiting')}`}
              aria-pressed={isSelected}
              className={
                isSelected
                  ? 'pipeline-node-button selected'
                  : isRelated
                    ? 'pipeline-node-button related'
                    : 'pipeline-node-button'
              }
              key={node.id}
              onClick={() => agent && onSelectAgent?.(agent)}
              onKeyDown={(event) => {
                if (!agent) return
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  onSelectAgent?.(agent)
                }
              }}
              role="button"
              tabIndex={0}
              transform={`translate(${node.x}, ${node.y})`}
            >
              <rect
                width="100"
                height="20"
                rx="4"
                className={`pipeline-node ${agent?.status ?? 'waiting'}`}
              />
              <text x="50" y="14" textAnchor="middle" className="pipeline-label">
                {agentRoleLabel(node.id)}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

export function GlobalAgentIndicator({ agents }: { agents: StoryAgentOutput[] }) {
  const running = agents.filter((a) => a.status === 'running' || a.status === 'waiting').length
  if (running === 0) return null
  return (
    <span className="global-agent-indicator" aria-label={`${running} Agent 执行中`}>
      <Activity size={14} aria-hidden="true" className="agent-pulse-icon" />
      {running} Agent 执行中
    </span>
  )
}
