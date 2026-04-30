import type { StoryAgentOutput } from '../api/types'
import type { InspectorTab } from './types'
import { studioRoutePaths } from './routes'
import { agentRoleLabel, agentStatusLabel } from './utils'

const MAX_KEY_POINTS = 4
const MAX_STRUCTURED_BLOCKS = 6
const MAX_BLOCK_ENTRIES = 5

export type StructuredOutputBlock = {
  title: string
  entries: string[]
}

export type StructuredAgentOutput = {
  blocks: StructuredOutputBlock[]
  keyPoints: string[]
  summaryHeadline: string
  summaryText: string
  summaryClipboard: string
}

export type AgentFollowUpFeedback = 'adopted' | 'needs_follow_up'
export type AgentFeedbackFilter = 'all' | 'adopted' | 'needs_follow_up' | 'unmarked'
export type AgentFeedbackSummary = {
  adopted: number
  needs_follow_up: number
  total: number
  unmarked: number
}
export type AgentFeedbackSurfaceSummary = {
  assetsGraph: AgentFeedbackSummary
  storyboard: AgentFeedbackSummary
}

export type StorySourceDraftInsertion = {
  id: string
  label: string
  text: string
}

export type AgentOutputReviewContext = {
  assetsGraphPendingCount: number
  assetsGraphReturnedCount: number
  storyboardPendingCount: number
  storyboardReturnedCount: number
  totalReturnedCount: number
  totalPendingCount: number
}

export type AgentOutputHandoffState = {
  agentLabel: string
  agentRole: string
  focusNodeKind?: 'character' | 'prop' | 'scene'
  followUpFeedback?: AgentFollowUpFeedback
  fromAgentOutput: true
  inspectorTab?: InspectorTab
  reviewContext?: AgentOutputReviewContext
}

export type AgentOutputFollowUpTarget = {
  actionLabel: string
  description: string
  path: string
  state: AgentOutputHandoffState
}

export type StoryAnalysisFollowUpHandoffState = {
  agentLabel: string
  agentRole: string
  followUpFeedback?: AgentFollowUpFeedback
  fromDownstreamFollowUp: true
  returnedFeedback?: AgentFollowUpFeedback
  resultNote?: string
  sourcePage: 'Storyboard' | 'Assets / Graph'
  suggestedFilter: AgentFeedbackFilter
}

export type ReturnedFollowUpSummary = {
  agentLabel: string
  agentRole: string
  feedback: AgentFollowUpFeedback
  resultNote?: string
  sourcePage: StoryAnalysisFollowUpHandoffState['sourcePage']
}

export type ReturnedFollowUpHistoryEntry = ReturnedFollowUpSummary & {
  createdAt: string
  id: string
}

export function buildStructuredAgentOutput(agent: StoryAgentOutput): StructuredAgentOutput {
  const normalizedOutput = normalizeAgentOutput(agent.output)
  const blocks = buildStructuredOutputBlocks(normalizedOutput)
  const keyPoints = uniqueStrings([
    ...agent.highlights,
    ...blocks.flatMap((block) => block.entries),
    extractLeadSentence(normalizedOutput),
  ]).slice(0, MAX_KEY_POINTS)
  const summaryHeadline =
    keyPoints[0] ?? `${agentRoleLabel(agent.role)} 已返回 ${agentStatusLabel(agent.status)} 输出`
  const summaryText =
    keyPoints.slice(1, 3).join('；') ||
    extractLeadSentence(normalizedOutput) ||
    '当前 Agent 已返回结果，可直接复制摘要或继续查看结构化明细。'
  const summaryClipboard = [
    `${agentRoleLabel(agent.role)} · ${agentStatusLabel(agent.status)}`,
    `重点：${summaryHeadline}`,
    summaryText ? `摘要：${summaryText}` : '',
    keyPoints.length > 1 ? `要点：${keyPoints.slice(1).join('；')}` : '',
  ]
    .filter(Boolean)
    .join('\n')

  return {
    blocks,
    keyPoints,
    summaryHeadline,
    summaryText,
    summaryClipboard,
  }
}

export function buildStorySourceDraftInsertion(agent: StoryAgentOutput): StorySourceDraftInsertion {
  const structured = buildStructuredAgentOutput(agent)
  return {
    id: `${agent.role}-${Date.now()}`,
    label: agentRoleLabel(agent.role),
    text: [
      `# Agent 回填 · ${agentRoleLabel(agent.role)}`,
      structured.summaryHeadline,
      structured.summaryText,
      structured.keyPoints.length > 1
        ? `补充要点：${structured.keyPoints.slice(1).join('；')}`
        : '',
    ]
      .filter(Boolean)
      .join('\n'),
  }
}

export function appendStorySourceDraftInsertion(
  current: string,
  draftInsertion: StorySourceDraftInsertion,
): string {
  if (current.includes(draftInsertion.text)) return current
  if (current.trim() === '') return draftInsertion.text
  return `${current.trim()}\n\n${draftInsertion.text}`
}

export function buildAgentFollowUpTarget(
  agent: StoryAgentOutput,
  followUpFeedback?: AgentFollowUpFeedback,
  reviewContext?: AgentOutputReviewContext,
): AgentOutputFollowUpTarget | null {
  const stateBase = {
    agentLabel: agentRoleLabel(agent.role),
    agentRole: agent.role,
    followUpFeedback,
    fromAgentOutput: true as const,
    reviewContext,
  }

  switch (agent.role) {
    case 'character_analyst':
      return {
        actionLabel: '前往 Assets / Graph',
        description: '聚焦角色图谱节点，继续补 Character Bible 与引用图。',
        path: studioRoutePaths.assetsGraph,
        state: {
          ...stateBase,
          focusNodeKind: 'character',
        },
      }
    case 'scene_analyst':
      return {
        actionLabel: '前往 Assets / Graph',
        description: '聚焦场景图谱节点，继续检查场景候选与引用资产。',
        path: studioRoutePaths.assetsGraph,
        state: {
          ...stateBase,
          focusNodeKind: 'scene',
        },
      }
    case 'prop_analyst':
      return {
        actionLabel: '前往 Assets / Graph',
        description: '聚焦道具图谱节点，继续锁定道具与补齐参考资产。',
        path: studioRoutePaths.assetsGraph,
        state: {
          ...stateBase,
          focusNodeKind: 'prop',
        },
      }
    case 'story_analyst':
    case 'outline_planner':
      return {
        actionLabel: '前往 Storyboard',
        description: '带着当前节拍分析跳到 Storyboard，检查剧情落点和镜头拆分。',
        path: studioRoutePaths.storyboard,
        state: {
          ...stateBase,
          inspectorTab: 'details',
        },
      }
    case 'screenwriter':
    case 'director':
    case 'cinematographer':
    case 'voice_subtitle':
      return {
        actionLabel: '前往 Storyboard',
        description: '直接进入 Prompt / 检查器工作流，继续做镜头和提示词微调。',
        path: studioRoutePaths.storyboard,
        state: {
          ...stateBase,
          inspectorTab: 'prompt',
        },
      }
    default:
      return null
  }
}

export function agentFollowUpFeedbackLabel(
  feedback: AgentFollowUpFeedback | undefined,
): string {
  if (feedback === 'adopted') return '已采纳'
  if (feedback === 'needs_follow_up') return '待跟进'
  return '未标记'
}

export function agentFeedbackFilterLabel(filter: AgentFeedbackFilter): string {
  const labels: Record<AgentFeedbackFilter, string> = {
    adopted: '已采纳',
    all: '全部',
    needs_follow_up: '待跟进',
    unmarked: '未标记',
  }
  return labels[filter]
}

export function buildAgentFeedbackSummary(
  agents: StoryAgentOutput[],
  feedbackByRole: Partial<Record<string, AgentFollowUpFeedback>>,
): AgentFeedbackSummary {
  return {
    adopted: agents.filter((agent) => feedbackByRole[agent.role] === 'adopted').length,
    needs_follow_up: agents.filter((agent) => feedbackByRole[agent.role] === 'needs_follow_up').length,
    total: agents.length,
    unmarked: agents.filter((agent) => feedbackByRole[agent.role] === undefined).length,
  }
}

export function buildAgentFeedbackSurfaceSummary(
  agents: StoryAgentOutput[],
  feedbackByRole: Partial<Record<string, AgentFollowUpFeedback>>,
): AgentFeedbackSurfaceSummary {
  return {
    assetsGraph: buildAgentFeedbackSummary(
      agents.filter((agent) => resolveAgentFeedbackSurface(agent.role) === 'assetsGraph'),
      feedbackByRole,
    ),
    storyboard: buildAgentFeedbackSummary(
      agents.filter((agent) => resolveAgentFeedbackSurface(agent.role) === 'storyboard'),
      feedbackByRole,
    ),
  }
}

export function buildStoryAnalysisFollowUpReturnState(
  handoff: Pick<AgentOutputHandoffState, 'agentLabel' | 'agentRole' | 'followUpFeedback'>,
  sourcePage: StoryAnalysisFollowUpHandoffState['sourcePage'],
  returnedFeedback?: AgentFollowUpFeedback,
  resultNote?: string,
): StoryAnalysisFollowUpHandoffState {
  return {
    agentLabel: handoff.agentLabel,
    agentRole: handoff.agentRole,
    followUpFeedback: handoff.followUpFeedback,
    fromDownstreamFollowUp: true,
    returnedFeedback,
    resultNote,
    sourcePage,
    suggestedFilter:
      returnedFeedback === 'adopted'
        ? 'all'
        : handoff.followUpFeedback === 'needs_follow_up'
          ? 'needs_follow_up'
          : 'all',
  }
}

export function matchesAgentFeedbackFilter(
  role: string,
  feedbackByRole: Partial<Record<string, AgentFollowUpFeedback>>,
  filter: AgentFeedbackFilter,
): boolean {
  const feedback = feedbackByRole[role]
  if (filter === 'all') return true
  if (filter === 'unmarked') return feedback === undefined
  return feedback === filter
}

function buildStructuredOutputBlocks(output: string): StructuredOutputBlock[] {
  const parsed = tryParseJson(output)
  if (parsed) {
    if (Array.isArray(parsed)) {
      return [{ title: '输出列表', entries: valueToEntries(parsed) }]
    }
    if (isRecord(parsed)) {
      const blocks = Object.entries(parsed)
        .map(([key, value]) => ({
          title: formatStructuredKey(key),
          entries: valueToEntries(value),
        }))
        .filter((block) => block.entries.length > 0)
        .slice(0, MAX_STRUCTURED_BLOCKS)
      if (blocks.length > 0) return blocks
    }
  }

  const paragraphs = output
    .split(/\n\s*\n/)
    .map((paragraph) => paragraph.trim())
    .filter(Boolean)
    .slice(0, MAX_STRUCTURED_BLOCKS)

  if (paragraphs.length > 1) {
    return paragraphs.map((paragraph, index) => ({
      title: index === 0 ? '核心描述' : `补充说明 ${index}`,
      entries: paragraphToEntries(paragraph),
    }))
  }

  return [
    {
      title: '输出内容',
      entries: paragraphToEntries(output),
    },
  ]
}

function resolveAgentFeedbackSurface(role: string): keyof AgentFeedbackSurfaceSummary | null {
  if (role === 'character_analyst' || role === 'scene_analyst' || role === 'prop_analyst') {
    return 'assetsGraph'
  }
  if (
    role === 'story_analyst' ||
    role === 'outline_planner' ||
    role === 'screenwriter' ||
    role === 'director' ||
    role === 'cinematographer' ||
    role === 'voice_subtitle'
  ) {
    return 'storyboard'
  }
  return null
}

function valueToEntries(value: unknown): string[] {
  if (Array.isArray(value)) {
    return uniqueStrings(value.map((entry) => summarizeStructuredValue(entry))).slice(
      0,
      MAX_BLOCK_ENTRIES,
    )
  }

  const summary = summarizeStructuredValue(value)
  return summary ? [summary] : []
}

function summarizeStructuredValue(value: unknown): string {
  if (typeof value === 'string') return collapseWhitespace(value)
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)

  if (Array.isArray(value)) {
    return uniqueStrings(value.map((entry) => summarizeStructuredValue(entry)))
      .slice(0, 3)
      .join('；')
  }

  if (!isRecord(value)) return ''

  const primaryParts = [
    value.code,
    value.title,
    value.name,
    value.character,
    value.scene,
    value.shot_size,
    value.style,
  ]
    .filter((part): part is string => typeof part === 'string' && part.trim().length > 0)
    .map((part) => collapseWhitespace(part))

  const secondaryParts = [
    value.summary,
    value.description,
    value.notes,
    value.visual_goal,
    value.reason,
  ]
    .filter((part): part is string => typeof part === 'string' && part.trim().length > 0)
    .map((part) => collapseWhitespace(part))

  const preferred = uniqueStrings([...primaryParts, ...secondaryParts]).slice(0, 3)
  if (preferred.length > 0) return preferred.join(' · ')

  return Object.entries(value)
    .map(([key, entryValue]) => {
      const content = summarizeLeaf(entryValue)
      return content ? `${formatStructuredKey(key)}：${content}` : ''
    })
    .filter(Boolean)
    .slice(0, 4)
    .join(' · ')
}

function summarizeLeaf(value: unknown): string {
  if (typeof value === 'string') return collapseWhitespace(value)
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (Array.isArray(value)) {
    return value
      .map((entry) => summarizeLeaf(entry))
      .filter(Boolean)
      .slice(0, 3)
      .join('；')
  }
  return ''
}

function paragraphToEntries(paragraph: string): string[] {
  const lines = paragraph
    .split('\n')
    .map((line) => line.replace(/^[-*•\d.)\s]+/, '').trim())
    .filter(Boolean)
  if (lines.length > 1) return uniqueStrings(lines).slice(0, MAX_BLOCK_ENTRIES)

  const sentences = paragraph
    .split(/(?<=[。！？])/)
    .map((sentence) => sentence.trim())
    .filter(Boolean)
  return uniqueStrings(sentences.length > 0 ? sentences : [paragraph.trim()]).slice(0, MAX_BLOCK_ENTRIES)
}

function extractLeadSentence(output: string): string {
  return (
    output
      .split(/(?<=[。！？])/)
      .map((sentence) => sentence.trim())
      .find(Boolean) ?? ''
  )
}

function normalizeAgentOutput(output: string): string {
  return output
    .trim()
    .replace(/^```[a-zA-Z0-9_-]*\n?/, '')
    .replace(/```$/, '')
    .trim()
}

function tryParseJson(output: string): unknown {
  try {
    return JSON.parse(output)
  } catch {
    return null
  }
}

function formatStructuredKey(key: string): string {
  return key.replaceAll('_', ' ')
}

function collapseWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

function uniqueStrings(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
