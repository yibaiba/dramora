import { useEffect, useMemo, useState } from 'react'
import { BookOpenText, Boxes, Layers3, Sparkles } from 'lucide-react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  useGenerationJobs,
  useSeedEpisodeProduction,
  useStartStoryAnalysis,
  useStoryAnalyses,
  useStorySources,
  useStoryboardWorkspace,
  useWorkflowRun,
} from '../../api/hooks'
import type { GenerationJob, StoryAgentOutput, WorkflowRun } from '../../api/types'
import {
  buildAgentFeedbackSurfaceSummary,
  appendStorySourceDraftInsertion,
  agentFollowUpFeedbackLabel,
  buildAgentFeedbackSummary,
  buildAgentFollowUpTarget,
  buildStorySourceDraftInsertion,
  matchesAgentFeedbackFilter,
  type ReturnedFollowUpHistoryEntry,
  type ReturnedFollowUpSummary,
  type StoryAnalysisFollowUpHandoffState,
  type AgentFeedbackFilter,
  type AgentFollowUpFeedback,
} from '../agentOutput'
import { BlackboardView, TemplateSelector } from '../components/AnalysisExtensions'
import type { AnalysisTemplate } from '../components/analysisTemplates'
import { storyAnalysisTemplates } from '../components/analysisTemplates'
import { AgentBoard, AgentPipeline, GlobalAgentIndicator } from '../components/AgentBoard'
import { AgentFeedbackWorkspace } from '../components/AgentFeedbackWorkspace'
import { AgentOutputPanel } from '../components/AgentOutputPanel'
import { ActionButton } from '../components/ActionButton'
import { ProductionFlowPanel } from '../components/ProductionFlowPanel'
import { StoryAnalysisPanel } from '../components/StoryAnalysisPanel'
import { WorkflowRecoveryTimeline } from '../components/WorkflowRecoveryTimeline'
import { useStudioSelection } from '../hooks/useStudioSelection'
import {
  appendReturnedFollowUpHistoryEntry,
  buildStoryAnalysisFeedbackStorageEntryKey,
  mergeReturnedFollowUpHistoryEntries,
  persistStoryAnalysisFeedback,
  persistStoryAnalysisReturnHistory,
  readPersistedStoryAnalysisFeedback,
  readPersistedStoryAnalysisReturnHistory,
} from '../reviewPersistence'
import { studioRoutePaths } from '../routes'
import { agentRoleLabel, productionHint, resolveEpisodeWorkflowRunId } from '../utils'

const storyAnalysisTemplateStorageKey = 'dramora.story-analysis.template-id'

export function StoryAnalysisPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const storyAnalysisHandoff = location.state as StoryAnalysisFollowUpHandoffState | null
  const { activeEpisode } = useStudioSelection()
  const startStoryAnalysis = useStartStoryAnalysis()
  const seedProduction = useSeedEpisodeProduction()
  const { data: generationJobs = [] } = useGenerationJobs()
  const { data: analyses = [] } = useStoryAnalyses(activeEpisode?.id)
  const { data: sources = [] } = useStorySources(activeEpisode?.id)
  const { data: storyboardWorkspace } = useStoryboardWorkspace(activeEpisode?.id)
  const [reviewClosureNotice, setReviewClosureNotice] = useState<string | null>(null)
  const [persistedReturnHistory, setPersistedReturnHistory] = useState(() =>
    readPersistedStoryAnalysisReturnHistory(),
  )
  const [sourceComposerFocusToken, setSourceComposerFocusToken] = useState(0)
  const [sourceDraftNotice, setSourceDraftNotice] = useState<string | null>(null)
  const [sourceDraftText, setSourceDraftText] = useState('')
  const [feedbackFilter, setFeedbackFilter] = useState<AgentFeedbackFilter>(() =>
    initialFeedbackFilterForHandoff(storyAnalysisHandoff),
  )
  const [persistedAgentFeedback, setPersistedAgentFeedback] = useState(() =>
    readPersistedStoryAnalysisFeedback(),
  )
  const [selectedAgent, setSelectedAgent] = useState<StoryAgentOutput | null>(null)
  const [viewMode, setViewMode] = useState<'board' | 'dag'>('board')
  const [selectedTemplateId, setSelectedTemplateId] = useState(() => readPersistedStoryAnalysisTemplateId())

  const latestAnalysis = analyses[0]
  const currentWorkflowRunId = useMemo(
    () => resolveEpisodeWorkflowRunId(activeEpisode?.id, analyses, generationJobs),
    [activeEpisode?.id, analyses, generationJobs],
  )
  const { data: workflowRun } = useWorkflowRun(currentWorkflowRunId)
  const feedbackStorageEntryKey = latestAnalysis
    ? buildStoryAnalysisFeedbackStorageEntryKey(latestAnalysis)
    : null
  const agentOutputs = latestAnalysis?.agent_outputs ?? []
  const agentFollowUpFeedback = feedbackStorageEntryKey
    ? persistedAgentFeedback[feedbackStorageEntryKey] ?? {}
    : {}
  const returnedFollowUpHistory = feedbackStorageEntryKey
    ? persistedReturnHistory[feedbackStorageEntryKey] ?? []
    : []
  const returnedHistorySummary = useMemo(
    () => ({
      assetsGraph: returnedFollowUpHistory.filter((entry) => entry.sourcePage === 'Assets / Graph').length,
      storyboard: returnedFollowUpHistory.filter((entry) => entry.sourcePage === 'Storyboard').length,
      total: returnedFollowUpHistory.length,
    }),
    [returnedFollowUpHistory],
  )
  const visibleAgentOutputs = useMemo(
    () =>
      agentOutputs.filter((agent) =>
        matchesAgentFeedbackFilter(agent.role, agentFollowUpFeedback, feedbackFilter),
      ),
    [agentFollowUpFeedback, agentOutputs, feedbackFilter],
  )
  const filteredAgentRoles = visibleAgentOutputs.map((agent) => agent.role)
  const agentFeedbackSummary = useMemo(
    () => buildAgentFeedbackSummary(agentOutputs, agentFollowUpFeedback),
    [agentFollowUpFeedback, agentOutputs],
  )
  const agentFeedbackSurfaceSummary = useMemo(
    () => buildAgentFeedbackSurfaceSummary(agentOutputs, agentFollowUpFeedback),
    [agentFollowUpFeedback, agentOutputs],
  )
  const handoffReviewContext = useMemo(
    () => ({
      assetsGraphPendingCount: agentFeedbackSurfaceSummary.assetsGraph.needs_follow_up,
      assetsGraphReturnedCount: returnedHistorySummary.assetsGraph,
      storyboardPendingCount: agentFeedbackSurfaceSummary.storyboard.needs_follow_up,
      storyboardReturnedCount: returnedHistorySummary.storyboard,
      totalReturnedCount: returnedHistorySummary.total,
      totalPendingCount: agentFeedbackSummary.needs_follow_up,
    }),
    [agentFeedbackSummary.needs_follow_up, agentFeedbackSurfaceSummary, returnedHistorySummary],
  )
  const handoffAgent = storyAnalysisHandoff?.agentRole
    ? agentOutputs.find((agent) => agent.role === storyAnalysisHandoff.agentRole)
    : null
  const returnedFollowUpSummary: ReturnedFollowUpSummary | null =
    storyAnalysisHandoff?.fromDownstreamFollowUp && storyAnalysisHandoff.returnedFeedback
      ? {
          agentLabel: storyAnalysisHandoff.agentLabel,
          agentRole: storyAnalysisHandoff.agentRole,
          feedback: storyAnalysisHandoff.returnedFeedback,
          resultNote: storyAnalysisHandoff.resultNote,
          sourcePage: storyAnalysisHandoff.sourcePage,
        }
      : null
  const storyboardFollowUpAgent =
    agentOutputs.find(
      (agent) =>
        agentFollowUpFeedback[agent.role] === 'needs_follow_up' &&
        buildAgentFollowUpTarget(agent, agentFollowUpFeedback[agent.role])?.path === studioRoutePaths.storyboard,
    ) ?? null
  const assetsGraphFollowUpAgent =
    agentOutputs.find(
      (agent) =>
        agentFollowUpFeedback[agent.role] === 'needs_follow_up' &&
        buildAgentFollowUpTarget(agent, agentFollowUpFeedback[agent.role])?.path === studioRoutePaths.assetsGraph,
    ) ?? null
  const canAutoCloseReturnedFeedback =
    Boolean(handoffAgent && storyAnalysisHandoff?.returnedFeedback) &&
    isReviewCycleComplete(
      buildAgentFeedbackSummary(
        agentOutputs,
        applyAgentFeedbackBatch(
          agentFollowUpFeedback,
          [storyAnalysisHandoff!.agentRole],
          storyAnalysisHandoff!.returnedFeedback,
        ),
      ),
    )
  const selectedTemplate =
    storyAnalysisTemplates.find((template) => template.id === selectedTemplateId) ??
    storyAnalysisTemplates[0]
  const storyMapReady = storyboardWorkspace?.summary.story_map_ready ?? false
  const nextHint = productionHint({
    activeEpisode,
    hasAnalysis: analyses.length > 0,
    storyMapReady,
  })
  const dominantFollowUpSurface =
    agentFeedbackSurfaceSummary.storyboard.needs_follow_up >=
      agentFeedbackSurfaceSummary.assetsGraph.needs_follow_up
      ? 'Storyboard'
      : 'Assets / Graph'
  const nextHandoffHeadline =
    agentFeedbackSummary.needs_follow_up > 0
      ? `优先收口 ${agentFeedbackSummary.needs_follow_up} 个待跟进`
      : agentFeedbackSummary.unmarked > 0
        ? `还有 ${agentFeedbackSummary.unmarked} 个未标记 Agent`
        : returnedHistorySummary.total > 0
          ? `本轮已回传 ${returnedHistorySummary.total} 条`
          : analyses.length > 0
            ? '可继续生成分镜'
            : '先完成解析'
  const nextHandoffDescription =
    agentFeedbackSummary.needs_follow_up > 0
      ? `${dominantFollowUpSurface} 当前是主要收口面，建议优先处理后再推进下一轮生产。`
      : agentFeedbackSummary.unmarked > 0
        ? '先完成本轮 review 标记，再决定是否收口或继续下游生产。'
        : returnedHistorySummary.total > 0
          ? '最近下游回传已持续收口，可结合导演台建议直接结束本轮 review。'
          : '分析结果会直接驱动故事图谱、候选资产与镜头卡生成。'
  const handleTemplateSelect = (templateId: string) => {
    const nextTemplateId = resolveStoryAnalysisTemplateId(templateId)
    setSelectedTemplateId(nextTemplateId)
    persistStoryAnalysisTemplateId(nextTemplateId)
  }
  useEffect(() => {
    persistStoryAnalysisFeedback(persistedAgentFeedback)
  }, [persistedAgentFeedback])
  useEffect(() => {
    persistStoryAnalysisReturnHistory(persistedReturnHistory)
  }, [persistedReturnHistory])
  const selectedAgentFollowUpTarget = selectedAgent
    ? buildAgentFollowUpTarget(
        selectedAgent,
        agentFollowUpFeedback[selectedAgent.role],
        handoffReviewContext,
      )
    : null
  const handleInsertAgentOutput = (agent: StoryAgentOutput) => {
    setReviewClosureNotice(null)
    const nextInsertion = buildStorySourceDraftInsertion(agent)
    setSourceDraftText((current) => appendStorySourceDraftInsertion(current, nextInsertion))
    setSourceDraftNotice(`已回填 ${nextInsertion.label} 的摘要，可继续补充后重新保存故事源。`)
    setSourceComposerFocusToken(Date.now())
  }
  const handleSetAgentFeedback = (agent: StoryAgentOutput, feedback: AgentFollowUpFeedback) => {
    if (!feedbackStorageEntryKey) return

    setReviewClosureNotice(null)
    setPersistedAgentFeedback((current) => {
      const currentFeedback = current[feedbackStorageEntryKey] ?? {}
      if (currentFeedback[agent.role] === feedback) {
        const nextFeedback = { ...currentFeedback }
        delete nextFeedback[agent.role]
        if (Object.keys(nextFeedback).length === 0) {
          const next = { ...current }
          delete next[feedbackStorageEntryKey]
          return next
        }

        return {
          ...current,
          [feedbackStorageEntryKey]: nextFeedback,
        }
      }

      return {
        ...current,
        [feedbackStorageEntryKey]: {
          ...currentFeedback,
          [agent.role]: feedback,
        },
      }
    })
  }

  const handleClearFeedback = () => {
    if (!feedbackStorageEntryKey) return

    setReviewClosureNotice(null)
    setPersistedAgentFeedback((current) => {
      if (!current[feedbackStorageEntryKey]) {
        return current
      }

      const next = { ...current }
      delete next[feedbackStorageEntryKey]
      return next
    })
  }
  const handleApplyBatchFeedback = (feedback: AgentFollowUpFeedback) => {
    if (!feedbackStorageEntryKey || filteredAgentRoles.length === 0) return

    setReviewClosureNotice(null)
    setPersistedAgentFeedback((current) => {
      const currentFeedback = current[feedbackStorageEntryKey] ?? {}
      return {
        ...current,
        [feedbackStorageEntryKey]: applyAgentFeedbackBatch(
          currentFeedback,
          filteredAgentRoles,
          feedback,
        ),
      }
    })
  }
  const handleClearFilteredFeedback = () => {
    if (!feedbackStorageEntryKey || filteredAgentRoles.length === 0) return

    setReviewClosureNotice(null)
    setPersistedAgentFeedback((current) => {
      const currentFeedback = current[feedbackStorageEntryKey]
      if (!currentFeedback) {
        return current
      }

      const nextFeedback = applyAgentFeedbackBatch(currentFeedback, filteredAgentRoles)
      if (Object.keys(nextFeedback).length === 0) {
        const next = { ...current }
        delete next[feedbackStorageEntryKey]
        return next
      }

      return {
        ...current,
        [feedbackStorageEntryKey]: nextFeedback,
      }
    })
  }
  const handleOpenNextFollowUp = () => {
    setReviewClosureNotice(null)
    const followUpAgents = visibleAgentOutputs.filter(
      (agent) => agentFollowUpFeedback[agent.role] === 'needs_follow_up',
    )
    if (followUpAgents.length === 0) {
      return
    }

    const currentIndex = selectedAgent
      ? followUpAgents.findIndex((agent) => agent.role === selectedAgent.role)
      : -1
    const nextAgent = followUpAgents[(currentIndex + 1) % followUpAgents.length]
    setSelectedAgent(nextAgent)
  }
  const handleOpenFollowUpTarget = (agent: StoryAgentOutput) => {
    setReviewClosureNotice(null)
    const target = buildAgentFollowUpTarget(
      agent,
      agentFollowUpFeedback[agent.role],
      handoffReviewContext,
    )
    if (!target) {
      return
    }

    navigate(target.path, { state: target.state })
  }
  const handleApplyReturnedFeedback = (
    agent: StoryAgentOutput,
    feedback: AgentFollowUpFeedback,
    sourcePage: ReturnedFollowUpSummary['sourcePage'],
    resultNote?: string,
    closeReviewCycle = false,
  ) => {
    if (!feedbackStorageEntryKey) return

    setReviewClosureNotice(null)
    setPersistedAgentFeedback((current) => {
      const currentFeedback = current[feedbackStorageEntryKey] ?? {}
      const nextFeedback = applyAgentFeedbackBatch(currentFeedback, [agent.role], feedback)
      if (closeReviewCycle && isReviewCycleComplete(buildAgentFeedbackSummary(agentOutputs, nextFeedback))) {
        const next = { ...current }
        delete next[feedbackStorageEntryKey]
        return next
      }

      return {
        ...current,
        [feedbackStorageEntryKey]: nextFeedback,
      }
    })
    setFeedbackFilter('all')
    setSelectedAgent(closeReviewCycle ? null : agent)
    setPersistedReturnHistory((current) =>
      appendReturnedFollowUpHistoryEntry(current, feedbackStorageEntryKey, {
        agentLabel: agentRoleLabel(agent.role),
        agentRole: agent.role,
        createdAt: new Date().toISOString(),
        feedback,
        id: `${agent.role}-${Date.now()}`,
        resultNote,
        sourcePage,
      }),
    )
    setReviewClosureNotice(
      closeReviewCycle
        ? '已应用下游回传结果，并自动收口本轮反馈。'
        : resultNote ?? `已将 ${agentRoleLabel(agent.role)} 标记为${agentFollowUpFeedbackLabel(feedback)}。`,
    )
  }
  const handleSelectHistoryEntry = (entry: ReturnedFollowUpHistoryEntry) => {
    setReviewClosureNotice(null)
    setFeedbackFilter('all')
    const agent = agentOutputs.find((candidate) => candidate.role === entry.agentRole)
    if (agent) {
      setSelectedAgent(agent)
    }
  }
  const handleOpenHistorySource = (entry: ReturnedFollowUpHistoryEntry) => {
    const agent = agentOutputs.find((candidate) => candidate.role === entry.agentRole)
    if (agent) {
      handleOpenFollowUpTarget(agent)
    }
  }
  const handleImportHistoryEntries = (entries: ReturnedFollowUpHistoryEntry[]): number => {
    if (!feedbackStorageEntryKey || entries.length === 0) return 0

    let importedCount = 0
    setPersistedReturnHistory((current) => {
      const existing = current[feedbackStorageEntryKey] ?? []
      const merged = mergeReturnedFollowUpHistoryEntries(existing, entries)
      importedCount = merged.length - existing.length
      if (merged.length === existing.length) {
        return current
      }
      return { ...current, [feedbackStorageEntryKey]: merged }
    })
    if (importedCount > 0) {
      setReviewClosureNotice(`已导入 ${importedCount} 条回传记录到当前分析。`)
    }
    return Math.max(importedCount, 0)
  }
  const handleRemoveHistoryEntry = (entry: ReturnedFollowUpHistoryEntry) => {
    if (!feedbackStorageEntryKey) return

    setPersistedReturnHistory((current) => {
      const existing = current[feedbackStorageEntryKey]
      if (!existing) {
        return current
      }

      const next = existing.filter((candidate) => candidate.id !== entry.id)
      if (next.length === existing.length) {
        return current
      }
      if (next.length === 0) {
        const cleared = { ...current }
        delete cleared[feedbackStorageEntryKey]
        return cleared
      }
      return { ...current, [feedbackStorageEntryKey]: next }
    })
    setReviewClosureNotice(`已移除来自 ${entry.sourcePage} · ${entry.agentLabel} 的回传记录。`)
  }
  const handleRemoveHistoryEntries = (entries: ReturnedFollowUpHistoryEntry[]) => {
    if (!feedbackStorageEntryKey || entries.length === 0) return

    const removeIds = new Set(entries.map((entry) => entry.id))
    setPersistedReturnHistory((current) => {
      const existing = current[feedbackStorageEntryKey]
      if (!existing) {
        return current
      }

      const next = existing.filter((candidate) => !removeIds.has(candidate.id))
      if (next.length === existing.length) {
        return current
      }
      if (next.length === 0) {
        const cleared = { ...current }
        delete cleared[feedbackStorageEntryKey]
        return cleared
      }
      return { ...current, [feedbackStorageEntryKey]: next }
    })
    setReviewClosureNotice(`已批量移除 ${entries.length} 条回传记录。`)
  }
  const handleCloseReviewCycle = () => {
    if (!feedbackStorageEntryKey) return

    setPersistedAgentFeedback((current) => {
      if (!current[feedbackStorageEntryKey]) {
        return current
      }

      const next = { ...current }
      delete next[feedbackStorageEntryKey]
      return next
    })
    setSelectedAgent(null)
    setFeedbackFilter('all')
    setReviewClosureNotice('已收口本轮反馈，可继续开启下一轮 review。')
  }

  return (
    <section className="studio-page" aria-labelledby="story-analysis-title">
      <div className="board-header">
        <div>
          <h1 id="story-analysis-title">Story Analysis</h1>
          <span>保存故事源、触发多 Agent 解析，并把结果送往分镜与资产生产。</span>
        </div>
        <div className="board-actions">
          <GlobalAgentIndicator agents={agentOutputs} />
          {agentOutputs.length > 0 && (
            <div className="view-toggle">
              <button
                type="button"
                className={viewMode === 'board' ? 'active' : ''}
                onClick={() => setViewMode('board')}
              >
                看板
              </button>
              <button
                type="button"
                className={viewMode === 'dag' ? 'active' : ''}
                onClick={() => setViewMode('dag')}
              >
                DAG
              </button>
            </div>
          )}
          <button
            className="hero-primary-action"
            disabled={!activeEpisode || startStoryAnalysis.isPending}
            onClick={() => activeEpisode && startStoryAnalysis.mutate(activeEpisode.id)}
            type="button"
          >
            <Sparkles aria-hidden="true" />
            {startStoryAnalysis.isPending ? '解析中...' : '启动故事解析'}
          </button>
          <Link className="hero-secondary-action" to={studioRoutePaths.storyboard}>
            <Boxes aria-hidden="true" />
            前往分镜台
          </Link>
        </div>
      </div>

      <div className="dashboard-grid">
        <article className="surface-card">
          <span className="section-kicker">Source readiness</span>
          <strong>{sources.length} 份故事源</strong>
          <p>{sources[0]?.title ?? '当前剧集还没有保存任何故事源。'}</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Analysis runs</span>
          <strong>{analyses.length} 份解析结果</strong>
          <p>{analyses[0]?.summary ?? '启动解析后，这里会展示最新摘要。'}</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Next handoff</span>
          <strong>{nextHandoffHeadline}</strong>
          <p>{nextHandoffDescription}</p>
        </article>
      </div>

      <TemplateSelector selectedTemplateId={selectedTemplateId} onSelect={handleTemplateSelect} />

      <AutomationEntryPanel
        activeEpisodeId={activeEpisode?.id}
        analysesCount={analyses.length}
        assetsGraphFollowUpCount={agentFeedbackSurfaceSummary.assetsGraph.needs_follow_up}
        feedbackSurfaceSummary={agentFeedbackSurfaceSummary}
        feedbackSummary={agentFeedbackSummary}
        jobs={storyboardWorkspace?.generation_jobs ?? []}
        nextHint={nextHint}
        onContinueAssetsGraphFollowUp={() =>
          assetsGraphFollowUpAgent && handleOpenFollowUpTarget(assetsGraphFollowUpAgent)
        }
        onCloseReviewCycle={handleCloseReviewCycle}
        onFocusUnmarkedQueue={() => {
          setReviewClosureNotice(null)
          setFeedbackFilter('unmarked')
        }}
        onOpenNextFollowUp={handleOpenNextFollowUp}
        onContinueStoryboardFollowUp={() =>
          storyboardFollowUpAgent && handleOpenFollowUpTarget(storyboardFollowUpAgent)
        }
        onSeedProduction={() => activeEpisode && seedProduction.mutate(activeEpisode.id)}
        onStartStoryAnalysis={() => activeEpisode && startStoryAnalysis.mutate(activeEpisode.id)}
        readyAssetsCount={storyboardWorkspace?.summary.ready_assets_count ?? 0}
        returnedFollowUpSummary={returnedFollowUpSummary}
        returnedHistorySummary={returnedHistorySummary}
        seedProductionPending={seedProduction.isPending}
        selectedTemplate={selectedTemplate}
        sourcesCount={sources.length}
        startStoryAnalysisPending={startStoryAnalysis.isPending}
        storyMapReady={storyMapReady}
        storyboardFollowUpCount={agentFeedbackSurfaceSummary.storyboard.needs_follow_up}
        storyboardShotsCount={storyboardWorkspace?.storyboard_shots.length ?? 0}
        workflowRun={workflowRun}
        workspaceAssetsCount={storyboardWorkspace?.assets.length ?? 0}
        workspaceGatesCount={storyboardWorkspace?.approval_gates.length ?? 0}
      />
      <WorkflowRecoveryTimeline workflowRun={workflowRun} />
      {storyAnalysisHandoff?.fromDownstreamFollowUp ? (
        <div className="board-notice timeline-handoff-notice">
          已从 {storyAnalysisHandoff.sourcePage} 回到 Story Analysis ·
          {` ${storyAnalysisHandoff.agentLabel} 当前标记 ${agentFollowUpFeedbackLabel(
            storyAnalysisHandoff.followUpFeedback,
          )}`}
          {storyAnalysisHandoff.suggestedFilter === 'needs_follow_up'
            ? ' · 已默认切到待跟进筛选'
            : ''}
          {storyAnalysisHandoff.resultNote ? ` · ${storyAnalysisHandoff.resultNote}` : ''}
          {handoffAgent ? (
            <button
              type="button"
              className="ghost-action"
              onClick={() => {
                setReviewClosureNotice(null)
                setSelectedAgent(handoffAgent)
              }}
            >
              打开对应 Agent
            </button>
          ) : null}
          {handoffAgent && storyAnalysisHandoff.returnedFeedback ? (
            <button
              type="button"
              className="ghost-action"
              onClick={() =>
                handleApplyReturnedFeedback(
                  handoffAgent,
                  storyAnalysisHandoff.returnedFeedback!,
                  storyAnalysisHandoff.sourcePage,
                  storyAnalysisHandoff.resultNote,
                )
              }
            >
              应用回传结果
            </button>
          ) : null}
          {handoffAgent && storyAnalysisHandoff.returnedFeedback && canAutoCloseReturnedFeedback ? (
            <button
              type="button"
              className="ghost-action"
              onClick={() =>
                handleApplyReturnedFeedback(
                  handoffAgent,
                  storyAnalysisHandoff.returnedFeedback!,
                  storyAnalysisHandoff.sourcePage,
                  storyAnalysisHandoff.resultNote,
                  true,
                )
              }
            >
              应用并收口本轮反馈
            </button>
          ) : null}
        </div>
      ) : null}

      {agentOutputs.length > 0 && (
        <>
          <AgentFeedbackWorkspace
            agents={agentOutputs}
            closureNotice={reviewClosureNotice}
            feedbackByRole={agentFollowUpFeedback}
            filter={feedbackFilter}
            onApplyBatchFeedback={handleApplyBatchFeedback}
            onCloseReviewCycle={handleCloseReviewCycle}
            onClearFeedback={handleClearFeedback}
            onClearFilteredFeedback={handleClearFilteredFeedback}
            onFilterChange={setFeedbackFilter}
            onInsertIntoSource={handleInsertAgentOutput}
            onOpenNextFollowUp={handleOpenNextFollowUp}
            onOpenFollowUpTarget={handleOpenFollowUpTarget}
            onOpenHistorySource={handleOpenHistorySource}
            onImportHistoryEntries={handleImportHistoryEntries}
            onRemoveHistoryEntry={handleRemoveHistoryEntry}
            onRemoveHistoryEntries={handleRemoveHistoryEntries}
            onSelectAgent={setSelectedAgent}
            onSelectHistoryEntry={handleSelectHistoryEntry}
            returnedFollowUpHistory={returnedFollowUpHistory}
            returnedFollowUpSummary={returnedFollowUpSummary}
            selectedRole={selectedAgent?.role}
          />
          {visibleAgentOutputs.length > 0
            ? viewMode === 'board'
              ? (
                <AgentBoard
                  agents={visibleAgentOutputs}
                  expandedRole={selectedAgent?.role}
                  onSelectAgent={setSelectedAgent}
                  workflowRun={workflowRun}
                />
              )
              : (
                <AgentPipeline
                  agents={visibleAgentOutputs}
                  onSelectAgent={setSelectedAgent}
                  selectedRole={selectedAgent?.role}
                />
              )
            : null}

          {selectedAgent && (
            <AgentOutputPanel
              agent={selectedAgent}
              feedbackState={agentFollowUpFeedback[selectedAgent.role]}
              followUpTarget={selectedAgentFollowUpTarget}
              onClose={() => setSelectedAgent(null)}
              onInsertIntoSource={handleInsertAgentOutput}
              onSetFeedback={handleSetAgentFeedback}
            />
          )}
        </>
      )}

      <BlackboardView
        analysis={latestAnalysis}
        feedbackByRole={agentFollowUpFeedback}
        feedbackSummary={agentFeedbackSummary}
        returnedFollowUpSummary={returnedFollowUpSummary}
        sourcesCount={sources.length}
        workspace={storyboardWorkspace}
      />

      <StoryAnalysisPanel
        activeEpisode={activeEpisode}
        analyses={analyses}
        draftNotice={sourceDraftNotice}
        onDraftNoticeChange={setSourceDraftNotice}
        onSourceTextChange={setSourceDraftText}
        sourceComposerFocusToken={sourceComposerFocusToken}
        sourceText={sourceDraftText}
        selectedTemplate={selectedTemplate}
      />

      <article className="surface-card">
        <div className="panel-title-row">
          <div>
            <span>多页面工作流</span>
            <strong>故事解析完成后的下一步</strong>
          </div>
        </div>
        <div className="page-link-grid">
          <Link className="page-link-card" to={studioRoutePaths.home}>
            <BookOpenText aria-hidden="true" />
            <strong>返回 Home</strong>
            <small>查看全局生产状态与路由入口。</small>
          </Link>
          <Link className="page-link-card" to={studioRoutePaths.storyboard}>
            <Boxes aria-hidden="true" />
            <strong>进入 Storyboard</strong>
            <small>
              {agentFeedbackSurfaceSummary.storyboard.needs_follow_up > 0
                ? `当前有 ${agentFeedbackSurfaceSummary.storyboard.needs_follow_up} 个待跟进 Agent 指向 Storyboard。`
                : '继续生成镜头卡、提示词包和审批动作。'}
            </small>
          </Link>
          <Link className="page-link-card" to={studioRoutePaths.assetsGraph}>
            <Sparkles aria-hidden="true" />
            <strong>前往 Assets / Graph</strong>
            <small>
              {agentFeedbackSurfaceSummary.assetsGraph.needs_follow_up > 0
                ? `当前有 ${agentFeedbackSurfaceSummary.assetsGraph.needs_follow_up} 个待跟进 Agent 指向 Assets / Graph。`
                : '查看图谱节点、候选资产池和已锁定参考资产。'}
            </small>
          </Link>
        </div>
      </article>
    </section>
  )
}

function AutomationEntryPanel({
  activeEpisodeId,
  analysesCount,
  assetsGraphFollowUpCount,
  feedbackSurfaceSummary,
  feedbackSummary,
  jobs,
  nextHint,
  onContinueAssetsGraphFollowUp,
  onCloseReviewCycle,
  onFocusUnmarkedQueue,
  onOpenNextFollowUp,
  onContinueStoryboardFollowUp,
  onSeedProduction,
  onStartStoryAnalysis,
  readyAssetsCount,
  returnedFollowUpSummary,
  returnedHistorySummary,
  seedProductionPending,
  selectedTemplate,
  sourcesCount,
  startStoryAnalysisPending,
  storyMapReady,
  storyboardFollowUpCount,
  storyboardShotsCount,
  workflowRun,
  workspaceAssetsCount,
  workspaceGatesCount,
}: {
  activeEpisodeId?: string
  analysesCount: number
  assetsGraphFollowUpCount: number
  feedbackSurfaceSummary: ReturnType<typeof buildAgentFeedbackSurfaceSummary>
  feedbackSummary: ReturnType<typeof buildAgentFeedbackSummary>
  jobs: GenerationJob[]
  nextHint: string
  onContinueAssetsGraphFollowUp: () => void
  onCloseReviewCycle: () => void
  onFocusUnmarkedQueue: () => void
  onOpenNextFollowUp: () => void
  onContinueStoryboardFollowUp: () => void
  onSeedProduction: () => void
  onStartStoryAnalysis: () => void
  readyAssetsCount: number
  returnedFollowUpSummary: ReturnedFollowUpSummary | null
  returnedHistorySummary: {
    assetsGraph: number
    storyboard: number
    total: number
  }
  seedProductionPending: boolean
  selectedTemplate: AnalysisTemplate
  sourcesCount: number
  startStoryAnalysisPending: boolean
  storyMapReady: boolean
  storyboardFollowUpCount: number
  storyboardShotsCount: number
  workflowRun?: WorkflowRun
  workspaceAssetsCount: number
  workspaceGatesCount: number
}) {
  const automationState =
    analysesCount === 0
      ? '先保存故事源并启动故事解析，再把 Agent 输出继续送往分镜生产。'
      : storyboardShotsCount > 0
        ? '解析结果已经落到下游分镜，可继续做角色一致性和提示词微调。'
        : '解析结果已就绪，现在可以直接一键自动生产分镜包。'

  return (
    <article className="surface-card analysis-automation-card">
      <div className="panel-title-row">
        <div>
          <span>Agent orchestration</span>
          <strong>自动化生产入口</strong>
        </div>
        <small>{selectedTemplate.name} · {selectedTemplate.tone}</small>
      </div>
      <p className="analysis-automation-description">{automationState}</p>
      <div className="blackboard-chip-row">
        <span className="blackboard-chip">故事源 {sourcesCount}</span>
        <span className="blackboard-chip">解析结果 {analysesCount}</span>
        <span className="blackboard-chip">{storyMapReady ? '图谱已就绪' : '图谱待生成'}</span>
        <span className="blackboard-chip">镜头 {storyboardShotsCount}</span>
        <span className="blackboard-chip">候选资产 {workspaceAssetsCount}</span>
        <span className="blackboard-chip">已锁定 {readyAssetsCount}</span>
        <span className="blackboard-chip">审批点 {workspaceGatesCount}</span>
      </div>
      <div className="blackboard-chip-row">
        <span className="blackboard-chip">
          Storyboard 待跟进 {feedbackSurfaceSummary.storyboard.needs_follow_up}
        </span>
        <span className="blackboard-chip">
          Assets / Graph 待跟进 {feedbackSurfaceSummary.assetsGraph.needs_follow_up}
        </span>
        <span className="blackboard-chip">Storyboard 已回传 {returnedHistorySummary.storyboard}</span>
        <span className="blackboard-chip">Assets / Graph 已回传 {returnedHistorySummary.assetsGraph}</span>
        <span className="blackboard-chip">
          {feedbackSummary.needs_follow_up > 0 ? '优先处理待跟进项，再继续一键收口。' : '当前下游协同提示已收口。'}
        </span>
      </div>
      {feedbackSummary.total > 0 ? (
        <div className="board-notice analysis-automation-follow-up-notice">
          {feedbackSummary.needs_follow_up > 0
            ? `导演台建议：本轮已回传 ${returnedHistorySummary.total} 条，仍有 ${feedbackSummary.needs_follow_up} 条待跟进；先打开下一条待跟进 Agent。`
            : feedbackSummary.unmarked > 0
              ? `导演台建议：本轮已回传 ${returnedHistorySummary.total} 条，但还有 ${feedbackSummary.unmarked} 个未标记 Agent；先完成标记再继续推进。`
              : returnedHistorySummary.total > 0
                ? `导演台建议：本轮已累计回传 ${returnedHistorySummary.total} 条，可直接收口反馈。`
                : '导演台建议：本轮 Agent 已全部处理完成，可以直接收口反馈。'}
          {feedbackSummary.needs_follow_up > 0 ? (
            <button type="button" className="ghost-action" onClick={onOpenNextFollowUp}>
              打开下一条待跟进
            </button>
          ) : feedbackSummary.unmarked > 0 ? (
            <button type="button" className="ghost-action" onClick={onFocusUnmarkedQueue}>
              查看未标记队列
            </button>
          ) : (
            <button type="button" className="ghost-action" onClick={onCloseReviewCycle}>
              收口本轮反馈
            </button>
          )}
        </div>
      ) : null}
      {returnedFollowUpSummary ? (
        <div className="board-notice analysis-automation-follow-up-notice">
          最近回传：{returnedFollowUpSummary.sourcePage} · {returnedFollowUpSummary.agentLabel} ·{' '}
          {agentFollowUpFeedbackLabel(returnedFollowUpSummary.feedback)}
          {returnedFollowUpSummary.resultNote ? ` · ${returnedFollowUpSummary.resultNote}` : ''}
        </div>
      ) : null}
      <div className="template-guidance-row">
        {selectedTemplate.hints.map((hint) => (
          <span className="template-hint-chip" key={hint}>
            {hint}
          </span>
        ))}
      </div>
      <ProductionFlowPanel
        analysesCount={analysesCount}
        assetsCount={workspaceAssetsCount}
        feedbackSummary={feedbackSummary}
        gatesCount={workspaceGatesCount}
        jobs={jobs}
        nextHint={nextHint}
        shotsCount={storyboardShotsCount}
        storyMapReady={storyMapReady}
        workflowRun={workflowRun}
      />
      {storyboardFollowUpCount > 0 || assetsGraphFollowUpCount > 0 ? (
        <div className="board-notice analysis-automation-follow-up-notice">
          当前仍有下游待跟进项，建议优先继续收口后再推进下一轮生产。
          {storyboardFollowUpCount > 0 ? (
            <button type="button" className="ghost-action" onClick={onContinueStoryboardFollowUp}>
              继续处理 Storyboard 待跟进
            </button>
          ) : null}
          {assetsGraphFollowUpCount > 0 ? (
            <button type="button" className="ghost-action" onClick={onContinueAssetsGraphFollowUp}>
              继续处理 Assets / Graph 待跟进
            </button>
          ) : null}
        </div>
      ) : null}
      <div className="analysis-automation-actions" aria-label="自动化生产动作">
        <ActionButton
          disabled={!activeEpisodeId || startStoryAnalysisPending}
          icon={Sparkles}
          label={startStoryAnalysisPending ? '解析中...' : '启动故事解析'}
          onClick={onStartStoryAnalysis}
        />
        <ActionButton
          disabled={!activeEpisodeId || analysesCount === 0 || seedProductionPending}
          disabledReason={
            analysesCount === 0 ? '先保存故事源并完成一次故事解析，才能把 Agent 输出继续送往自动生产。' : undefined
          }
          icon={BookOpenText}
          label={seedProductionPending ? '生产中...' : '一键自动生产'}
          onClick={onSeedProduction}
        />
        <Link className="hero-secondary-action" to={studioRoutePaths.assetsGraph}>
          <Layers3 aria-hidden="true" />
          去 Assets / Graph
        </Link>
        <Link className="hero-secondary-action" to={studioRoutePaths.storyboard}>
          <Boxes aria-hidden="true" />
          去 Storyboard
        </Link>
      </div>
      <small className="analysis-automation-note">{nextHint}</small>
    </article>
  )
}

function defaultStoryAnalysisTemplateId(): string {
  return storyAnalysisTemplates[0]?.id ?? 'xianxia'
}

function resolveStoryAnalysisTemplateId(templateId?: string | null): string {
  return storyAnalysisTemplates.some((template) => template.id === templateId)
    ? templateId ?? defaultStoryAnalysisTemplateId()
    : defaultStoryAnalysisTemplateId()
}

function readPersistedStoryAnalysisTemplateId(): string {
  if (typeof window === 'undefined') {
    return defaultStoryAnalysisTemplateId()
  }

  return resolveStoryAnalysisTemplateId(
    window.localStorage.getItem(storyAnalysisTemplateStorageKey),
  )
}

function persistStoryAnalysisTemplateId(templateId: string): void {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(
    storyAnalysisTemplateStorageKey,
    resolveStoryAnalysisTemplateId(templateId),
  )
}

function initialFeedbackFilterForHandoff(
  handoff?: StoryAnalysisFollowUpHandoffState | null,
): AgentFeedbackFilter {
  return handoff?.suggestedFilter ?? 'all'
}

function applyAgentFeedbackBatch(
  currentFeedback: Partial<Record<string, AgentFollowUpFeedback>>,
  roles: string[],
  nextFeedback?: AgentFollowUpFeedback,
): Partial<Record<string, AgentFollowUpFeedback>> {
  const next = { ...currentFeedback }

  for (const role of roles) {
    if (nextFeedback) {
      next[role] = nextFeedback
      continue
    }

    delete next[role]
  }

  return next
}

function isReviewCycleComplete(summary: ReturnType<typeof buildAgentFeedbackSummary>): boolean {
  return summary.total > 0 && summary.needs_follow_up === 0 && summary.unmarked === 0
}
