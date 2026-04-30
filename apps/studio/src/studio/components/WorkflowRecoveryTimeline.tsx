import type { WorkflowRun } from '../../api/types'
import {
  formatCheckpointSavedAt,
  workflowNodeLabel,
  workflowNodeStatusLabel,
  workflowRunStatusLabel,
} from '../utils'

type WorkflowRecoveryTimelineProps = {
  workflowRun?: WorkflowRun
}

export function WorkflowRecoveryTimeline({ workflowRun }: WorkflowRecoveryTimelineProps) {
  const checkpointSummary = workflowRun?.checkpoint_summary
  const nodeRuns = workflowRun?.node_runs ?? []

  if (!workflowRun || !checkpointSummary || nodeRuns.length === 0) {
    return null
  }

  return (
    <article className="surface-card workflow-recovery-timeline">
      <div className="panel-title-row">
        <div>
          <span>Recovery detail</span>
          <strong>节点级恢复时间线</strong>
        </div>
        <small>
          {workflowRunStatusLabel(workflowRun.status)} · {formatCheckpointSavedAt(checkpointSummary.saved_at)}
        </small>
      </div>
      <div className="workflow-recovery-list" aria-label="Workflow 节点恢复详情">
        {nodeRuns.map((node) => {
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
    </article>
  )
}
