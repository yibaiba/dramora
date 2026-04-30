import { Layers3, LibraryBig, Sparkles, Workflow } from 'lucide-react'
import type { StoryAnalysis, StoryboardWorkspace } from '../../api/types'
import type {
  AgentFeedbackSummary,
  AgentFollowUpFeedback,
  ReturnedFollowUpSummary,
} from '../agentOutput'
import { agentFollowUpFeedbackLabel } from '../agentOutput'
import { storyAnalysisTemplates } from './analysisTemplates'
import { agentRoleLabel, agentStatusLabel } from '../utils'

type TemplateSelectorProps = {
  selectedTemplateId: string
  onSelect: (templateId: string) => void
}

export function TemplateSelector({ onSelect, selectedTemplateId }: TemplateSelectorProps) {
  const activeTemplate =
    storyAnalysisTemplates.find((template) => template.id === selectedTemplateId) ??
    storyAnalysisTemplates[0]

  return (
    <section className="template-selector" aria-labelledby="template-selector-title">
      <div className="panel-title-row">
        <div>
          <span className="section-kicker">Template presets</span>
          <strong id="template-selector-title">故事解析模板选择栏</strong>
        </div>
        <small>{activeTemplate.tone}</small>
      </div>
      <div className="template-chip-row" role="list" aria-label="故事解析模板">
        {storyAnalysisTemplates.map((template) => (
          <button
            aria-pressed={selectedTemplateId === template.id}
            className={selectedTemplateId === template.id ? 'template-chip active' : 'template-chip'}
            key={template.id}
            onClick={() => onSelect(template.id)}
            type="button"
          >
            <strong>{template.name}</strong>
            <span>{template.tone}</span>
          </button>
        ))}
      </div>
      <p className="template-description">{activeTemplate.description}</p>
      <small className="template-persist-note">当前模板会自动记住，下次打开 Story Analysis 继续沿用。</small>
    </section>
  )
}

type BlackboardViewProps = {
  analysis?: StoryAnalysis
  feedbackByRole?: Partial<Record<string, AgentFollowUpFeedback>>
  feedbackSummary?: AgentFeedbackSummary
  returnedFollowUpSummary?: ReturnedFollowUpSummary | null
  sourcesCount: number
  workspace?: StoryboardWorkspace
}

export function BlackboardView({
  analysis,
  feedbackByRole,
  feedbackSummary,
  returnedFollowUpSummary,
  sourcesCount,
  workspace,
}: BlackboardViewProps) {
  const agentOutputs = analysis?.agent_outputs ?? []
  const completedAgents = agentOutputs.filter((agent) => agent.status === 'succeeded').length
  const beats = analysis?.outline.slice(0, 3) ?? []
  const storyMap = workspace?.story_map
  const storyMapSummary = [
    `角色 ${storyMap?.characters.length ?? 0}`,
    `场景 ${storyMap?.scenes.length ?? 0}`,
    `道具 ${storyMap?.props.length ?? 0}`,
  ]

  return (
    <section className="analysis-blackboard" aria-labelledby="analysis-blackboard-title">
      <div className="panel-title-row">
        <div>
          <span className="section-kicker">Blackboard snapshot</span>
          <strong id="analysis-blackboard-title">项目状态快照</strong>
        </div>
        <small>{analysis ? `Analysis v${analysis.version}` : '等待解析结果'}</small>
      </div>
      <div className="analysis-blackboard-grid">
        <article className="blackboard-card">
          <div className="blackboard-card-header">
            <LibraryBig aria-hidden="true" />
            <strong>故事源上下文</strong>
          </div>
          <div className="blackboard-stat-row">
            <span>{sourcesCount} 份故事源</span>
            <span>{analysis ? analysis.summary : '等待启动解析'}</span>
          </div>
          <p>{analysis ? '最新摘要已进入多 Agent 流程，可继续观察大纲与下游生产 readiness。' : '先保存故事源，再启动解析以生成结构化黑板数据。'}</p>
        </article>

        <article className="blackboard-card">
          <div className="blackboard-card-header">
            <Workflow aria-hidden="true" />
            <strong>Agent 流水线</strong>
          </div>
          <div className="blackboard-stat-row">
            <span>
              {completedAgents}/{agentOutputs.length} 完成
            </span>
            <span>
              {agentOutputs.length > 0
                ? feedbackSummary
                  ? `已采纳 ${feedbackSummary.adopted} · 待跟进 ${feedbackSummary.needs_follow_up}`
                  : '状态由执行流实时驱动'
                : '等待 Agent 输出'}
            </span>
          </div>
          <div className="blackboard-chip-row">
            {agentOutputs.length === 0 ? (
              <span className="blackboard-chip muted">等待 Agent 输出</span>
            ) : (
              agentOutputs.map((agent) => (
                <span className="blackboard-chip" key={agent.role}>
                  {agentRoleLabel(agent.role)} · {agentStatusLabel(agent.status)}
                  {feedbackByRole?.[agent.role]
                    ? ` · ${agentFollowUpFeedbackLabel(feedbackByRole[agent.role])}`
                    : ''}
                </span>
              ))
            )}
            {returnedFollowUpSummary ? (
              <span className="blackboard-chip">
                最近回传 · {returnedFollowUpSummary.sourcePage} · {returnedFollowUpSummary.agentLabel} ·{' '}
                {agentFollowUpFeedbackLabel(returnedFollowUpSummary.feedback)}
              </span>
            ) : null}
          </div>
        </article>

        <article className="blackboard-card">
          <div className="blackboard-card-header">
            <Sparkles aria-hidden="true" />
            <strong>大纲与节拍</strong>
          </div>
          {beats.length === 0 ? (
            <p>暂无大纲节拍。启动解析后，这里会展示前三个关键 beat。</p>
          ) : (
            <div className="blackboard-list">
              {beats.map((beat) => (
                <div className="blackboard-list-item" key={beat.code}>
                  <strong>{beat.code}</strong>
                  <span>{beat.title}</span>
                  <small>{beat.summary}</small>
                </div>
              ))}
            </div>
          )}
        </article>

        <article className="blackboard-card">
          <div className="blackboard-card-header">
            <Layers3 aria-hidden="true" />
            <strong>下游生产状态</strong>
          </div>
          <div className="blackboard-stat-row">
            <span>{workspace?.summary.story_map_ready ? '图谱已就绪' : '图谱未就绪'}</span>
            <span>待审批 {workspace?.summary.pending_approval_gates_count ?? 0}</span>
          </div>
          <div className="blackboard-chip-row">
            {storyMapSummary.map((entry) => (
              <span className="blackboard-chip" key={entry}>
                {entry}
              </span>
            ))}
            <span className="blackboard-chip">镜头 {workspace?.storyboard_shots.length ?? 0}</span>
            <span className="blackboard-chip">候选资产 {workspace?.assets.length ?? 0}</span>
            <span className="blackboard-chip">已锁定 {workspace?.summary.ready_assets_count ?? 0}</span>
            <span className="blackboard-chip">生成任务 {workspace?.generation_jobs.length ?? 0}</span>
          </div>
        </article>
      </div>
    </section>
  )
}
