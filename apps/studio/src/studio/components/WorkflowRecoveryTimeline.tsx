import { useMemo } from 'react'

import type { WorkflowRun, WorkflowNodeRun } from '../../api/types'
import {
  formatCheckpointSavedAt,
  workflowNodeLabel,
  workflowNodeStatusLabel,
  workflowRunStatusLabel,
} from '../utils'

type WorkflowRecoveryTimelineProps = {
  workflowRun?: WorkflowRun
}

function computeNodeTiers(nodes: WorkflowNodeRun[]): Map<string, number> {
  const byID = new Map(nodes.map((node) => [node.node_id, node]))
  const tiers = new Map<string, number>()
  const visiting = new Set<string>()

  const resolve = (nodeID: string): number => {
    if (tiers.has(nodeID)) return tiers.get(nodeID)!
    if (visiting.has(nodeID)) return 0
    visiting.add(nodeID)
    const node = byID.get(nodeID)
    if (!node || node.upstream_node_ids.length === 0) {
      tiers.set(nodeID, 0)
      visiting.delete(nodeID)
      return 0
    }
    let max = 0
    for (const upstream of node.upstream_node_ids) {
      if (!byID.has(upstream)) continue
      const tier = resolve(upstream) + 1
      if (tier > max) max = tier
    }
    tiers.set(nodeID, max)
    visiting.delete(nodeID)
    return max
  }

  for (const node of nodes) {
    resolve(node.node_id)
  }
  return tiers
}

export function WorkflowRecoveryTimeline({ workflowRun }: WorkflowRecoveryTimelineProps) {
  const checkpointSummary = workflowRun?.checkpoint_summary
  const nodeRuns = useMemo(() => workflowRun?.node_runs ?? [], [workflowRun?.node_runs])

  const tierGroups = useMemo(() => {
    if (nodeRuns.length === 0) return [] as { tier: number; nodes: WorkflowNodeRun[] }[]
    const tiers = computeNodeTiers(nodeRuns)
    const grouped = new Map<number, WorkflowNodeRun[]>()
    for (const node of nodeRuns) {
      const tier = tiers.get(node.node_id) ?? 0
      if (!grouped.has(tier)) grouped.set(tier, [])
      grouped.get(tier)!.push(node)
    }
    return Array.from(grouped.entries())
      .sort(([a], [b]) => a - b)
      .map(([tier, nodes]) => ({ tier, nodes }))
  }, [nodeRuns])

  if (!workflowRun || !checkpointSummary || nodeRuns.length === 0) {
    return null
  }

  return (
    <article className="surface-card workflow-recovery-timeline">
      <div className="panel-title-row">
        <div>
          <span>Recovery detail</span>
          <strong>节点级恢复时间线（按 DAG 层级）</strong>
        </div>
        <small>
          {workflowRunStatusLabel(workflowRun.status)} · {formatCheckpointSavedAt(checkpointSummary.saved_at)}
        </small>
      </div>
      <div className="workflow-recovery-dag" aria-label="Workflow DAG 节点恢复详情">
        {tierGroups.map(({ tier, nodes }) => (
          <div className="workflow-recovery-tier" key={`tier-${tier}`}>
            <div className="workflow-recovery-tier-heading">
              <span>Tier {tier + 1}</span>
              <small>{nodes.length} 节点</small>
            </div>
            <div className="workflow-recovery-tier-nodes">
              {nodes.map((node) => {
                const tone =
                  node.status === 'pending'
                    ? 'queued'
                    : node.status === 'waiting_approval'
                      ? 'queued'
                      : node.status
                return (
                  <article className="workflow-recovery-card" key={node.node_id}>
                    <div className="workflow-recovery-card-header">
                      <div>
                        <strong>{workflowNodeLabel(node.node_id, node.kind)}</strong>
                        <span>
                          {node.upstream_node_ids.length > 0
                            ? `依赖 ${node.upstream_node_ids.map((item) => workflowNodeLabel(item)).join(' · ')}`
                            : '起始节点'}
                        </span>
                      </div>
                      <span className={`job-pill ${tone}`}>{workflowNodeStatusLabel(node.status)}</span>
                    </div>
                    {node.upstream_node_ids.length > 0 ? (
                      <div className="workflow-recovery-upstream">
                        {node.upstream_node_ids.map((upstream) => (
                          <span className="workflow-recovery-upstream-tag" key={`${node.node_id}-from-${upstream}`}>
                            ← {workflowNodeLabel(upstream)}
                          </span>
                        ))}
                      </div>
                    ) : null}
                    {node.summary ? <p className="workflow-recovery-copy">{node.summary}</p> : null}
                    {node.highlights.length > 0 ? (
                      <div className="agent-highlights">
                        {node.highlights.slice(0, 3).map((highlight, index) => (
                          <span className="agent-highlight-tag" key={`${node.node_id}-${index}`}>
                            {highlight}
                          </span>
                        ))}
                      </div>
                    ) : null}
                    {node.error_message ? (
                      <p className="workflow-recovery-error">恢复错误：{node.error_message}</p>
                    ) : null}
                  </article>
                )
              })}
            </div>
          </div>
        ))}
      </div>
    </article>
  )
}
