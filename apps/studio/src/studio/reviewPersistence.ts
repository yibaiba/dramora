import type { StoryAnalysis } from '../api/types'
import {
  buildAgentFeedbackSummary,
  buildAgentFeedbackSurfaceSummary,
  type AgentFollowUpFeedback,
  type ReturnedFollowUpHistoryEntry,
  type ReturnedFollowUpSummary,
} from './agentOutput'

const storyAnalysisFeedbackStorageKey = 'dramora.story-analysis.feedback-by-analysis'
const storyAnalysisReturnHistoryStorageKey = 'dramora.story-analysis.return-history-by-analysis'

export type PersistedStoryAnalysisFeedback = Record<
  string,
  Partial<Record<string, AgentFollowUpFeedback>>
>

export type PersistedStoryAnalysisReturnHistory = Record<string, ReturnedFollowUpHistoryEntry[]>

export type StoryAnalysisReviewSnapshot = {
  feedbackByRole: Partial<Record<string, AgentFollowUpFeedback>>
  feedbackSummary: ReturnType<typeof buildAgentFeedbackSummary>
  latestReturnedSummary: ReturnedFollowUpSummary | null
  returnHistory: ReturnedFollowUpHistoryEntry[]
  returnedHistorySummary: {
    assetsGraph: number
    storyboard: number
    total: number
  }
  surfaceSummary: ReturnType<typeof buildAgentFeedbackSurfaceSummary>
}

export function buildStoryAnalysisFeedbackStorageEntryKey(
  analysis: Pick<StoryAnalysis, 'episode_id' | 'id'>,
): string {
  return `${analysis.episode_id}:${analysis.id}`
}

export function readPersistedStoryAnalysisFeedback(): PersistedStoryAnalysisFeedback {
  if (typeof window === 'undefined') {
    return {}
  }

  const raw = window.localStorage.getItem(storyAnalysisFeedbackStorageKey)
  if (!raw) {
    return {}
  }

  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') {
      return {}
    }

    return Object.fromEntries(
      Object.entries(parsed).map(([analysisKey, value]) => [
        analysisKey,
        sanitizePersistedAgentFeedback(value),
      ]),
    )
  } catch {
    return {}
  }
}

export function persistStoryAnalysisFeedback(feedback: PersistedStoryAnalysisFeedback): void {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(storyAnalysisFeedbackStorageKey, JSON.stringify(feedback))
}

export function readPersistedStoryAnalysisReturnHistory(): PersistedStoryAnalysisReturnHistory {
  if (typeof window === 'undefined') {
    return {}
  }

  const raw = window.localStorage.getItem(storyAnalysisReturnHistoryStorageKey)
  if (!raw) {
    return {}
  }

  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') {
      return {}
    }

    return Object.fromEntries(
      Object.entries(parsed).map(([analysisKey, value]) => [
        analysisKey,
        sanitizeReturnedFollowUpHistory(value),
      ]),
    )
  } catch {
    return {}
  }
}

export function persistStoryAnalysisReturnHistory(history: PersistedStoryAnalysisReturnHistory): void {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(storyAnalysisReturnHistoryStorageKey, JSON.stringify(history))
}

export function appendReturnedFollowUpHistoryEntry(
  history: PersistedStoryAnalysisReturnHistory,
  key: string,
  entry: ReturnedFollowUpHistoryEntry,
): PersistedStoryAnalysisReturnHistory {
  return {
    ...history,
    [key]: [entry, ...(history[key] ?? [])].slice(0, 6),
  }
}

export function buildStoryAnalysisReviewSnapshot(
  analysis?: Pick<StoryAnalysis, 'agent_outputs' | 'episode_id' | 'id'>,
): StoryAnalysisReviewSnapshot | null {
  if (!analysis) {
    return null
  }

  const analysisKey = buildStoryAnalysisFeedbackStorageEntryKey(analysis)
  const feedbackByRole = readPersistedStoryAnalysisFeedback()[analysisKey] ?? {}
  const returnHistory = readPersistedStoryAnalysisReturnHistory()[analysisKey] ?? []
  const feedbackSummary = buildAgentFeedbackSummary(analysis.agent_outputs, feedbackByRole)
  const surfaceSummary = buildAgentFeedbackSurfaceSummary(analysis.agent_outputs, feedbackByRole)

  return {
    feedbackByRole,
    feedbackSummary,
    latestReturnedSummary: returnHistory[0]
      ? {
          agentLabel: returnHistory[0].agentLabel,
          agentRole: returnHistory[0].agentRole,
          feedback: returnHistory[0].feedback,
          resultNote: returnHistory[0].resultNote,
          sourcePage: returnHistory[0].sourcePage,
        }
      : null,
    returnHistory,
    returnedHistorySummary: {
      assetsGraph: returnHistory.filter((entry) => entry.sourcePage === 'Assets / Graph').length,
      storyboard: returnHistory.filter((entry) => entry.sourcePage === 'Storyboard').length,
      total: returnHistory.length,
    },
    surfaceSummary,
  }
}

function sanitizePersistedAgentFeedback(
  value: unknown,
): Partial<Record<string, AgentFollowUpFeedback>> {
  if (!value || typeof value !== 'object') {
    return {}
  }

  return Object.fromEntries(
    Object.entries(value).filter((entry): entry is [string, AgentFollowUpFeedback] =>
      entry[1] === 'adopted' || entry[1] === 'needs_follow_up',
    ),
  )
}

function sanitizeReturnedFollowUpHistory(value: unknown): ReturnedFollowUpHistoryEntry[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .filter((entry): entry is ReturnedFollowUpHistoryEntry => {
      if (!entry || typeof entry !== 'object') {
        return false
      }

      const candidate = entry as Partial<ReturnedFollowUpHistoryEntry>
      return (
        typeof candidate.agentLabel === 'string' &&
        typeof candidate.agentRole === 'string' &&
        typeof candidate.createdAt === 'string' &&
        (candidate.feedback === 'adopted' || candidate.feedback === 'needs_follow_up') &&
        typeof candidate.id === 'string' &&
        (candidate.sourcePage === 'Storyboard' || candidate.sourcePage === 'Assets / Graph')
      )
    })
    .slice(0, 6)
}
