import type { GenerationJob, WorkflowRun } from '../../api/types'
import type { AgentFeedbackSummary } from '../agentOutput'
import {
  formatCheckpointSavedAt,
  jobStatusLabel,
  jobTaskLabel,
  productionSteps,
  workflowRunStatusLabel,
} from '../utils'

type ProductionFlowPanelProps = {
  analysesCount: number
  assetsCount: number
  feedbackSummary?: AgentFeedbackSummary
  gatesCount: number
  jobs: GenerationJob[]
  nextHint: string
  shotsCount: number
  storyMapReady: boolean
  workflowRun?: WorkflowRun
}

export function ProductionFlowPanel({
  analysesCount,
  assetsCount,
  feedbackSummary,
  gatesCount,
  jobs,
  nextHint,
  shotsCount,
  storyMapReady,
  workflowRun,
}: ProductionFlowPanelProps) {
  const steps = productionSteps({
    analysesCount,
    assetsCount,
    gatesCount,
    shotsCount,
    storyMapReady,
  })
  const checkpointSummary = workflowRun?.checkpoint_summary

  return (
    <section className="production-flow" aria-label="真实生产流程状态">
      <div className="flow-steps">
        {steps.map((step) => (
          <div className={`flow-step ${step.state}`} key={step.label}>
            <strong>{step.label}</strong>
            <span>{step.value}</span>
          </div>
        ))}
      </div>
      <div className="production-rails">
        {feedbackSummary && feedbackSummary.total > 0 && (
          <div className="job-rail feedback-rail" aria-label="Agent follow-up 状态">
            <div className="job-rail-header">
              <strong>反馈回路</strong>
              <span>
                {feedbackSummary.needs_follow_up > 0
                  ? '仍有待跟进项，建议继续回看 Agent 输出。'
                  : '当前 follow-up 已收口，可继续推进下游生产。'}
              </span>
            </div>
            <div className="job-pill-row">
              <span className="job-pill succeeded">已采纳 {feedbackSummary.adopted}</span>
              <span className={`job-pill ${feedbackSummary.needs_follow_up > 0 ? 'running' : 'succeeded'}`}>
                待跟进 {feedbackSummary.needs_follow_up}
              </span>
              <span className="job-pill queued">未标记 {feedbackSummary.unmarked}</span>
            </div>
          </div>
        )}
        {checkpointSummary && (
          <div className="job-rail checkpoint-rail" aria-label="Workflow checkpoint 摘要">
            <div className="job-rail-header">
              <strong>恢复快照</strong>
              <span>
                {workflowRunStatusLabel(workflowRun.status)} · {formatCheckpointSavedAt(checkpointSummary.saved_at)}
              </span>
            </div>
            <div className="job-pill-row">
              <span className="job-pill succeeded">已完成 {checkpointSummary.completed_nodes}</span>
              <span className={`job-pill ${checkpointSummary.waiting_nodes > 0 ? 'queued' : 'succeeded'}`}>
                待执行 {checkpointSummary.waiting_nodes}
              </span>
              <span className={`job-pill ${checkpointSummary.running_nodes > 0 ? 'running' : 'queued'}`}>
                运行中 {checkpointSummary.running_nodes}
              </span>
              <span className={`job-pill ${checkpointSummary.failed_nodes > 0 ? 'failed' : 'succeeded'}`}>
                失败 {checkpointSummary.failed_nodes}
              </span>
              <span className="job-pill">Blackboard {checkpointSummary.blackboard_roles.length}</span>
            </div>
          </div>
        )}
        <div className="job-rail" aria-label="当前剧集生成队列">
          <div className="job-rail-header">
            <strong>生成队列</strong>
            <span>{nextHint}</span>
          </div>
          <div className="job-pill-row">
            {jobs.length === 0 ? (
              <span className="job-empty">
                暂无任务，启动故事解析后 worker 会自动推进。
              </span>
            ) : (
              jobs.map((job) => (
                <span className={`job-pill ${job.status}`} key={job.id}>
                  {jobTaskLabel(job.task_type)} · {jobStatusLabel(job.status)}
                </span>
              ))
            )}
          </div>
        </div>
      </div>
    </section>
  )
}
