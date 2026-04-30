import {
  Boxes,
  Home,
  Layers3,
  Library,
  Lock,
  Sparkles,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  useLockAsset,
  useSaveCharacterBible,
  useSeedEpisodeAssets,
  useSeedStoryMap,
  useStoryboardWorkspace,
} from '../../api/hooks'
import type { Asset, AssetStatus, StoryMapItem } from '../../api/types'
import {
  agentFollowUpFeedbackLabel,
  buildStoryAnalysisFollowUpReturnState,
  type AgentOutputHandoffState,
} from '../agentOutput'
import { ActionButton } from '../components/ActionButton'
import { AssetsGraphInspector } from '../components/AssetsGraphInspector'
import { ReviewSummaryChips } from '../components/ReviewSummaryChips'
import {
  cloneCharacterBibleDraft,
  createCharacterBibleDraft,
  isCharacterBibleDraftDirty,
  summarizeCharacterBibleDraft,
  type CharacterBibleDraft,
  type GraphNodeKind,
} from '../components/assetsGraphDrafts'
import { useStudioSelection } from '../hooks/useStudioSelection'
import { studioRoutePaths } from '../routes'
import {
  buildStoryboardCharacterReferences,
  mapWorkspaceDisplayShots,
  selectShotsForStoryMapNode,
} from '../utils'

type GraphGroup = {
  key: 'character' | 'scene' | 'prop'
  label: string
  items: StoryMapItem[]
}

function initialAssetsGraphNodeKey(
  handoff: AgentOutputHandoffState | null,
  groups: GraphGroup[],
): string | undefined {
  if (!handoff?.fromAgentOutput || !handoff.focusNodeKind) {
    return undefined
  }

  const group = groups.find((entry) => entry.key === handoff.focusNodeKind)
  const firstNode = group?.items[0]
  return firstNode ? `${group.key}:${firstNode.code}` : undefined
}

export function AssetsGraphPage() {
  const { activeEpisode } = useStudioSelection()
  const location = useLocation()
  const { data: storyboardWorkspace } = useStoryboardWorkspace(activeEpisode?.id)
  const lockAsset = useLockAsset()
  const saveCharacterBible = useSaveCharacterBible()
  const seedAssets = useSeedEpisodeAssets()
  const seedStoryMap = useSeedStoryMap()
  const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([])
  const [selectedNodeKey, setSelectedNodeKey] = useState<string | null>()
  const [characterBibleBaselines, setCharacterBibleBaselines] = useState<Record<string, CharacterBibleDraft>>({})
  const [characterBibleDrafts, setCharacterBibleDrafts] = useState<Record<string, CharacterBibleDraft>>({})
  const [focusedReferenceAngles, setFocusedReferenceAngles] = useState<Record<string, string | undefined>>({})
  const [characterBibleSaveMeta, setCharacterBibleSaveMeta] = useState<
    Record<string, { error?: string; lastSavedAt?: string }>
  >({})
  const [statusFilter, setStatusFilter] = useState<'all' | AssetStatus>('all')
  const analysesCount = storyboardWorkspace?.summary.analysis_count ?? 0
  const assets = storyboardWorkspace?.assets ?? []
  const workspaceStoryboardShots = storyboardWorkspace?.storyboard_shots ?? []
  const storyMap = storyboardWorkspace?.story_map
  const storyMapReady = storyboardWorkspace?.summary.story_map_ready ?? false
  const readyAssetsCount = assets.filter((asset) => asset.status === 'ready').length
  const graphGroups = useMemo<GraphGroup[]>(
    () => [
      { items: storyMap?.characters ?? [], key: 'character', label: '角色图谱' },
      { items: storyMap?.scenes ?? [], key: 'scene', label: '场景图谱' },
      { items: storyMap?.props ?? [], key: 'prop', label: '道具图谱' },
    ],
    [storyMap],
  )
  const graphNodesByKey = useMemo(() => {
    const nodes = new Map<string, StoryMapItem>()
    graphGroups.forEach((group) => {
      group.items.forEach((item) => {
        nodes.set(`${group.key}:${item.code}`, item)
      })
    })
    return nodes
  }, [graphGroups])
  const agentOutputHandoff = location.state as AgentOutputHandoffState | null
  const resolvedSelectedNodeKey =
    selectedNodeKey === undefined
      ? initialAssetsGraphNodeKey(agentOutputHandoff, graphGroups)
      : selectedNodeKey ?? undefined
  const assetsByPurpose = useMemo(() => {
    const groups = new Map<string, Asset[]>()
    assets.forEach((asset) => {
      const key = `${asset.kind}:${asset.purpose}`
      groups.set(key, [...(groups.get(key) ?? []), asset])
    })
    return groups
  }, [assets])
  const filteredAssets = useMemo(
    () =>
      assets.filter((asset) => {
        const matchesNode = !resolvedSelectedNodeKey || `${asset.kind}:${asset.purpose}` === resolvedSelectedNodeKey
        const matchesStatus = statusFilter === 'all' || asset.status === statusFilter
        return matchesNode && matchesStatus
      }),
    [assets, resolvedSelectedNodeKey, statusFilter],
  )
  const selectedAssets = useMemo(
    () => assets.filter((asset) => selectedAssetIds.includes(asset.id)),
    [assets, selectedAssetIds],
  )
  const lockableSelectedAssets = useMemo(
    () =>
      selectedAssets.filter(
        (asset) => asset.status !== 'ready' && asset.status !== 'archived',
      ),
    [selectedAssets],
  )
  const totalNodes = graphGroups.reduce((count, group) => count + group.items.length, 0)
  const selectedNode = resolvedSelectedNodeKey ? graphNodesByKey.get(resolvedSelectedNodeKey) : undefined
  const selectedNodeKind = resolvedSelectedNodeKey?.split(':')[0] as GraphNodeKind | undefined
  const selectedNodeAssets = resolvedSelectedNodeKey ? assetsByPurpose.get(resolvedSelectedNodeKey) ?? [] : []
  const selectedReadyReferenceAssets = selectedNodeAssets.filter((asset) => asset.status === 'ready')
  const storyboardDisplayShots = useMemo(
    () => (workspaceStoryboardShots.length > 0 ? mapWorkspaceDisplayShots(workspaceStoryboardShots) : []),
    [workspaceStoryboardShots],
  )
  const selectedCharacterBaseline =
    selectedNode && selectedNodeKind === 'character'
      ? characterBibleBaselines[selectedNode.id] ?? createCharacterBibleDraft(selectedNode)
      : undefined
  const selectedCharacterDraft =
    selectedNode && selectedNodeKind === 'character'
      ? characterBibleDrafts[selectedNode.id] ?? selectedCharacterBaseline
      : undefined
  const selectedCharacterDirty =
    selectedCharacterDraft && selectedCharacterBaseline
      ? isCharacterBibleDraftDirty(selectedCharacterDraft, selectedCharacterBaseline)
      : false
  const selectedCharacterSummary = selectedCharacterDraft
    ? summarizeCharacterBibleDraft(selectedCharacterDraft)
    : undefined
  const previewStoryMap = useMemo(() => {
    if (!storyMap || !selectedNode || selectedNodeKind !== 'character' || !selectedCharacterDraft) {
      return storyMap
    }

    return {
      ...storyMap,
      characters: storyMap.characters.map((character) =>
        character.id === selectedNode.id
          ? {
              ...character,
              character_bible: selectedCharacterDraft,
            }
          : character,
        ),
    }
  }, [selectedCharacterDraft, selectedNode?.id, selectedNodeKind, storyMap])
  const previewCharacterReferences = useMemo(
    () => buildStoryboardCharacterReferences(previewStoryMap, assets),
    [assets, previewStoryMap],
  )
  const selectedReferenceAngleSlots =
    selectedCharacterDraft?.reference_angles.map((angle) => ({
      angle,
      asset: selectedReadyReferenceAssets.find(
        (asset) =>
          asset.id === (selectedCharacterDraft.reference_assets ?? []).find((item) => item.angle === angle)?.asset_id,
      ),
    })) ?? []
  const selectedReferenceAssignmentStats = {
    assignedCount: selectedReferenceAngleSlots.filter((slot) => Boolean(slot.asset)).length,
    totalAngles: selectedReferenceAngleSlots.length,
    unassignedCount: selectedReferenceAngleSlots.filter((slot) => !slot.asset).length,
  }
  const selectedNodeStoryboardLinks = useMemo(
    () =>
      selectShotsForStoryMapNode(
        selectedNode,
        selectedNodeKind,
        storyboardDisplayShots,
        previewCharacterReferences,
      ),
    [previewCharacterReferences, selectedNode, selectedNodeKind, storyboardDisplayShots],
  )
  const selectedNodeStoryboardLinkSummary = {
    coveredCount: selectedNodeStoryboardLinks.filter((link) => link.coverage === 'covered').length,
    linkedCount: selectedNodeStoryboardLinks.length,
  }
  const selectedFocusedReferenceAngle =
    selectedNode && selectedNodeKind === 'character'
      ? focusedReferenceAngles[selectedNode.id] ??
        (selectedCharacterDraft?.reference_angles.length ? selectedCharacterDraft.reference_angles[0] : undefined)
      : undefined
  const selectedCharacterSaveMeta = selectedNode ? characterBibleSaveMeta[selectedNode.id] : undefined
  const selectedCharacterSavePending =
    saveCharacterBible.isPending &&
    selectedNodeKind === 'character' &&
    saveCharacterBible.variables?.characterId === selectedNode?.id
  const selectedCharacterSaveState =
    selectedCharacterSavePending
      ? 'saving'
      : selectedCharacterSaveMeta?.error
        ? 'error'
        : selectedCharacterDirty
          ? 'dirty'
          : selectedCharacterSaveMeta?.lastSavedAt || selectedNode?.character_bible
            ? 'saved'
            : 'draft'
  const selectedCharacterSaveMessage =
    selectedCharacterSaveMeta?.error
      ? selectedCharacterSaveMeta.error
      : selectedCharacterDirty
        ? '当前有未保存修改，保存后 Storyboard 与 Assets / Graph 会一起刷新。'
        : selectedCharacterSaveMeta?.lastSavedAt
          ? `最近一次保存于 ${formatSaveTime(selectedCharacterSaveMeta.lastSavedAt)}。`
          : selectedNode?.character_bible
            ? '当前 Character Bible 已与后端保持一致。'
            : '当前内容仍是本地草稿，保存后才能跨页复用。'

  const toggleAssetSelection = (assetId: string) => {
    setSelectedAssetIds((current) =>
      current.includes(assetId)
        ? current.filter((item) => item !== assetId)
        : [...current, assetId],
    )
  }

  const batchLockSelectedAssets = async () => {
    if (!activeEpisode || lockableSelectedAssets.length === 0) return

    const results = await Promise.allSettled(
      lockableSelectedAssets.map((asset) =>
        lockAsset.mutateAsync({
          assetId: asset.id,
          episodeId: activeEpisode.id,
        }),
      ),
    )
    const successfulIds = new Set(
      lockableSelectedAssets
        .filter((_, index) => results[index]?.status === 'fulfilled')
        .map((asset) => asset.id),
    )
    setSelectedAssetIds((current) =>
      current.filter((assetId) => !successfulIds.has(assetId)),
    )
  }

  const updateCharacterDraft = (
    node: StoryMapItem,
    updater: (draft: CharacterBibleDraft) => CharacterBibleDraft,
  ) => {
    setCharacterBibleDrafts((current) => {
      const base = current[node.id] ?? createCharacterBibleDraft(node)
      return {
        ...current,
        [node.id]: updater(base),
      }
    })
    setCharacterBibleSaveMeta((current) => ({
      ...current,
      [node.id]: {
        ...current[node.id],
        error: undefined,
      },
    }))
  }

  return (
    <section className="studio-page assets-graph-page" aria-labelledby="assets-graph-title">
      <div className="board-header">
        <div>
          <h1 id="assets-graph-title">Assets / Graph</h1>
          <span>聚焦故事图谱、候选资产池和可锁定参考资产。</span>
        </div>
        <div className="board-actions">
          <Link className="hero-secondary-action" to={studioRoutePaths.storyboard}>
            <Boxes aria-hidden="true" />
            回到 Storyboard
          </Link>
          <Link className="hero-secondary-action" to={studioRoutePaths.home}>
            <Home aria-hidden="true" />
            返回 Home
          </Link>
        </div>
      </div>

      <div className="dashboard-grid">
        <article className="surface-card">
          <span className="section-kicker">Graph nodes</span>
          <strong>{totalNodes} 个图谱节点</strong>
          <p>角色、场景、道具节点会驱动候选资产生成和后续参考图锁定。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Candidate assets</span>
          <strong>{assets.length} 个候选资产</strong>
          <p>每个节点默认对应一批候选素材，可在此页直接锁定为参考资产。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Locked references</span>
          <strong>{readyAssetsCount} 个已锁定</strong>
          <p>被锁定的资产会进入提示词包和后续视频生成链路。</p>
        </article>
      </div>

      <article className="surface-card graph-board">
        <div className="panel-title-row">
          <div>
            <span>Graph workspace</span>
            <strong>故事图谱与资产生成入口</strong>
          </div>
          <div className="section-actions">
            <ActionButton
              disabled={!activeEpisode || analysesCount === 0 || seedStoryMap.isPending}
              disabledReason={
                analysesCount === 0 ? '先完成故事解析，才能生成角色、场景和道具图谱。' : undefined
              }
              icon={Layers3}
              label={storyMapReady ? '资产图谱就绪' : '生成资产图谱'}
              onClick={() => activeEpisode && seedStoryMap.mutate(activeEpisode.id)}
            />
            <ActionButton
              disabled={!activeEpisode || !storyMapReady || seedAssets.isPending}
              disabledReason={!storyMapReady ? '先生成资产图谱，才能创建候选资产。' : undefined}
              icon={Sparkles}
              label={assets.length > 0 ? '补齐候选资产' : '生成候选资产'}
              onClick={() => activeEpisode && seedAssets.mutate(activeEpisode.id)}
            />
          </div>
        </div>
        {agentOutputHandoff?.fromAgentOutput ? (
          <>
            <div className="board-notice timeline-handoff-notice">
              已从 {agentOutputHandoff.agentLabel} 跳转到 Assets / Graph
              {resolvedSelectedNodeKey
                ? ` · 当前默认聚焦 ${resolvedSelectedNodeKey.replace(':', ' · ')}`
                : ' · 当前剧集还没有对应图谱节点'}
              {agentOutputHandoff.followUpFeedback
                ? ` · 当前标记 ${agentFollowUpFeedbackLabel(agentOutputHandoff.followUpFeedback)}`
                : ''}
              {agentOutputHandoff.reviewContext?.assetsGraphPendingCount
                ? ` · Assets / Graph 侧剩余 ${agentOutputHandoff.reviewContext.assetsGraphPendingCount} 个待跟进`
                : ''}
              {agentOutputHandoff.reviewContext?.storyboardPendingCount
                ? ` · Storyboard 侧还有 ${agentOutputHandoff.reviewContext.storyboardPendingCount} 个待跟进`
                : ''}
              {agentOutputHandoff.reviewContext?.assetsGraphReturnedCount
                ? ` · Assets / Graph 已回传 ${agentOutputHandoff.reviewContext.assetsGraphReturnedCount} 条`
                : ''}
              {agentOutputHandoff.followUpFeedback === 'needs_follow_up'
                ? ' · 完成锁定或补图后建议回到 Story Analysis 收口反馈'
                : ''}
              <Link
                className="ghost-action"
                to={studioRoutePaths.storyAnalysis}
                state={buildStoryAnalysisFollowUpReturnState(agentOutputHandoff, 'Assets / Graph')}
              >
                回到解析
              </Link>
              {agentOutputHandoff.followUpFeedback === 'needs_follow_up' ? (
                <Link
                  className="ghost-action"
                  to={studioRoutePaths.storyAnalysis}
                  state={buildStoryAnalysisFollowUpReturnState(
                    agentOutputHandoff,
                    'Assets / Graph',
                    'adopted',
                    '已在 Assets / Graph 完成锁定或补图，可回到解析确认收口。',
                  )}
                >
                  处理完成并回到解析
                </Link>
              ) : null}
              {agentOutputHandoff.reviewContext?.storyboardPendingCount ? (
                <Link className="ghost-action" to={studioRoutePaths.storyboard}>
                  去 Storyboard 继续处理
                </Link>
              ) : null}
            </div>
            {agentOutputHandoff.reviewContext ? (
              <ReviewSummaryChips
                currentSide="assetsGraph"
                storyboardPendingCount={agentOutputHandoff.reviewContext.storyboardPendingCount}
                assetsGraphPendingCount={agentOutputHandoff.reviewContext.assetsGraphPendingCount}
                totalReturnedCount={agentOutputHandoff.reviewContext.totalReturnedCount}
                storyAnalysisLinkState={buildStoryAnalysisFollowUpReturnState(agentOutputHandoff, 'Assets / Graph')}
              />
            ) : null}
          </>
        ) : null}

        {!activeEpisode ? (
          <div className="board-notice">先选择一个剧集，Assets / Graph 页面才会展示对应的图谱和候选资产。</div>
        ) : null}

        {activeEpisode && !storyMapReady ? (
          <div className="board-notice">
            {analysesCount > 0
              ? '当前已有故事解析，但图谱还未生成；先生成资产图谱，再继续锁定候选资产。'
              : '请先回到 Story Analysis 完成解析，再生成资产图谱和候选资产。'}
          </div>
        ) : null}

        <div className="graph-column-grid">
          {graphGroups.map((group) => (
            <section className="graph-column" key={group.key}>
              <header className="graph-column-header">
                <span className="section-kicker">{group.label}</span>
                <strong>{group.items.length} 个节点</strong>
              </header>
              {group.items.length === 0 ? (
                <div className="graph-empty">当前还没有可展示的 {assetKindLabel(group.key)} 节点。</div>
              ) : (
                <div className="graph-item-stack">
                  {group.items.map((item) => {
                    const nodeKey = `${group.key}:${item.code}`
                    const relatedAssets = assetsByPurpose.get(`${group.key}:${item.code}`) ?? []
                    const relatedReadyAssets = relatedAssets.filter((asset) => asset.status === 'ready').length

                    return (
                      <button
                        className={
                          resolvedSelectedNodeKey === nodeKey
                            ? 'graph-item-card filter-active'
                            : 'graph-item-card'
                        }
                        key={item.id}
                        onClick={() =>
                          setSelectedNodeKey((current) =>
                            (current === undefined ? resolvedSelectedNodeKey : current) === nodeKey ? null : nodeKey,
                          )
                        }
                        type="button"
                      >
                        <div className="graph-item-header">
                          <span className="scene-chip">{item.code}</span>
                          <strong>{item.name}</strong>
                        </div>
                        <p>{item.description}</p>
                        <div className="approval-summary-row">
                          <span>候选 {relatedAssets.length}</span>
                          <span>锁定 {relatedReadyAssets}</span>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </section>
          ))}
        </div>
      </article>

      <AssetsGraphInspector
        draft={selectedCharacterDraft}
        hasPersistedDraft={Boolean(selectedNode?.character_bible)}
        kind={selectedNodeKind}
        node={selectedNode}
        focusedReferenceAngle={selectedFocusedReferenceAngle}
        onAssignReferenceAsset={(assetId) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          selectedFocusedReferenceAngle &&
          updateCharacterDraft(selectedNode, (draft) => ({
            ...draft,
            reference_assets: [
              ...(draft.reference_assets ?? []).filter((item) => item.angle !== selectedFocusedReferenceAngle),
              { angle: selectedFocusedReferenceAngle, asset_id: assetId },
            ],
          }))
        }
        onChangeReferenceAngle={(angle) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          (() => {
            const nextReferenceAngles = selectedCharacterDraft?.reference_angles.includes(angle)
              ? selectedCharacterDraft.reference_angles.filter((item) => item !== angle)
              : [...(selectedCharacterDraft?.reference_angles ?? []), angle]

            updateCharacterDraft(selectedNode, (draft) => ({
              ...draft,
              reference_angles: nextReferenceAngles,
              reference_assets: (draft.reference_assets ?? []).filter(
                (item) => item.angle !== angle || nextReferenceAngles.includes(angle),
              ),
            }))
            setFocusedReferenceAngles((current) => ({
              ...current,
              [selectedNode.id]:
                nextReferenceAngles.includes(current[selectedNode.id] ?? '')
                  ? current[selectedNode.id] ?? nextReferenceAngles[0]
                  : nextReferenceAngles[0],
            }))
          })()
        }
        onClearReferenceAssignments={() =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({
            ...draft,
            reference_assets: [],
          }))
        }
        onAutoFillReferenceAssignments={() =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          selectedReadyReferenceAssets.length > 0 &&
          updateCharacterDraft(selectedNode, (draft) =>
            buildBatchReferenceAssignments(draft, selectedReadyReferenceAssets, 'fill-empty'),
          )
        }
        onChangeAnchor={(value) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({ ...draft, anchor: value }))
        }
        onChangeExpression={(expression) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({
            ...draft,
            expressions: draft.expressions.includes(expression)
              ? draft.expressions.filter((item) => item !== expression)
              : [...draft.expressions, expression],
          }))
        }
        onChangeNotes={(value) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({ ...draft, notes: value }))
        }
        onChangePalette={(key, value) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({
            ...draft,
            palette: {
              ...draft.palette,
              [key]: value,
            },
          }))
        }
        onChangeWardrobe={(value) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({ ...draft, wardrobe: value }))
        }
        onClearSelection={() => setSelectedNodeKey(null)}
        onFocusReferenceAngle={(angle) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          setFocusedReferenceAngles((current) => ({
            ...current,
            [selectedNode.id]: angle,
          }))
        }
        onRemoveReferenceAsset={(angle) =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          updateCharacterDraft(selectedNode, (draft) => ({
            ...draft,
            reference_assets: (draft.reference_assets ?? []).filter((item) => item.angle !== angle),
          }))
        }
        onReassignReferenceAssignments={() =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          selectedReadyReferenceAssets.length > 0 &&
          updateCharacterDraft(selectedNode, (draft) =>
            buildBatchReferenceAssignments(draft, selectedReadyReferenceAssets, 'reassign-all'),
          )
        }
        onResetDraft={() =>
          selectedNode &&
          selectedNodeKind === 'character' &&
          (() => {
            const resetDraft = selectedCharacterBaseline ?? createCharacterBibleDraft(selectedNode)
            setCharacterBibleDrafts((current) => ({
              ...current,
              [selectedNode.id]: cloneCharacterBibleDraft(resetDraft),
            }))
            setCharacterBibleSaveMeta((current) => ({
              ...current,
              [selectedNode.id]: {
                ...current[selectedNode.id],
                error: undefined,
              },
            }))
          })()
        }
        onSaveDraft={() =>
          activeEpisode &&
          selectedNode &&
          selectedNodeKind === 'character' &&
          selectedCharacterDraft &&
          saveCharacterBible.mutate(
            {
              characterId: selectedNode.id,
              episodeId: activeEpisode.id,
              request: {
                character_bible: selectedCharacterDraft,
              },
            },
            {
              onSuccess: (storyMapItem) => {
                const nextDraft = createCharacterBibleDraft(storyMapItem)
                setCharacterBibleBaselines((current) => ({
                  ...current,
                  [storyMapItem.id]: cloneCharacterBibleDraft(nextDraft),
                }))
                setCharacterBibleDrafts((current) => ({
                  ...current,
                  [storyMapItem.id]: cloneCharacterBibleDraft(nextDraft),
                }))
                setCharacterBibleSaveMeta((current) => ({
                  ...current,
                  [storyMapItem.id]: {
                    error: undefined,
                    lastSavedAt: new Date().toISOString(),
                  },
                }))
              },
              onError: (error) => {
                setCharacterBibleSaveMeta((current) => ({
                  ...current,
                  [selectedNode.id]: {
                    ...current[selectedNode.id],
                    error: error instanceof Error ? error.message : '保存 Character Bible 失败，请重试。',
                  },
                }))
              },
            },
          )
        }
        relatedAssets={selectedNodeAssets}
        referenceAssignments={selectedReferenceAngleSlots}
        referenceAssignmentStats={selectedReferenceAssignmentStats}
        saveMessage={selectedCharacterSaveMessage}
        savePending={selectedCharacterSavePending}
        saveState={selectedCharacterSaveState}
        storyboardLinks={selectedNodeStoryboardLinks.slice(0, 4)}
        storyboardLinkSummary={selectedNodeStoryboardLinkSummary}
        summary={selectedCharacterSummary}
      />

      <article className="surface-card">
        <div className="panel-title-row">
          <div>
            <span>Candidate pool</span>
            <strong>候选资产池</strong>
          </div>
          <div className="section-actions">
            <ActionButton
              disabled={lockableSelectedAssets.length === 0 || lockAsset.isPending}
              icon={Lock}
              label={
                lockAsset.isPending
                  ? '批量锁定中...'
                  : `批量锁定 ${lockableSelectedAssets.length} 项`
              }
              onClick={() => {
                void batchLockSelectedAssets()
              }}
            />
          </div>
        </div>
        <div className="asset-filter-toolbar" aria-label="资产筛选与选择">
          <div className="asset-filter-row">
            {assetStatusFilters.map((filter) => (
              <button
                className={
                  statusFilter === filter.value
                    ? 'asset-filter-chip active'
                    : 'asset-filter-chip'
                }
                key={filter.value}
                onClick={() => setStatusFilter(filter.value)}
                type="button"
              >
                {filter.label}
              </button>
            ))}
          </div>
          <div className="asset-filter-row">
            <span className="asset-filter-summary">
              当前筛选：{resolvedSelectedNodeKey ? resolvedSelectedNodeKey.replace(':', ' · ') : '全部节点'} ·{' '}
              {statusFilter === 'all' ? '全部状态' : assetStatusLabel(statusFilter)}
            </span>
            {resolvedSelectedNodeKey ? (
              <button
                className="asset-filter-chip"
                onClick={() => setSelectedNodeKey(null)}
                type="button"
              >
                清除节点筛选
              </button>
            ) : null}
            {selectedAssetIds.length > 0 ? (
              <button
                className="asset-filter-chip"
                onClick={() => setSelectedAssetIds([])}
                type="button"
              >
                清空已选 {selectedAssetIds.length}
              </button>
            ) : null}
          </div>
        </div>

        {assets.length === 0 ? (
          <div className="empty-board asset-empty-board">
            <Library aria-hidden="true" />
            <div>
              <strong>还没有候选资产</strong>
              <p>生成资产图谱并创建候选资产后，这里会出现可锁定的参考素材卡片。</p>
            </div>
          </div>
        ) : filteredAssets.length === 0 ? (
          <div className="graph-empty">当前筛选条件下没有资产，请切换节点或状态筛选。</div>
        ) : (
          <div className="asset-candidate-grid">
            {filteredAssets.map((asset) => (
              <AssetCandidateCard
                activeEpisodeId={activeEpisode?.id}
                asset={asset}
                graphNode={graphNodesByKey.get(`${asset.kind}:${asset.purpose}`)}
                key={asset.id}
                lockAsset={lockAsset}
                onToggleSelect={() => toggleAssetSelection(asset.id)}
                selected={selectedAssetIds.includes(asset.id)}
              />
            ))}
          </div>
        )}
      </article>
    </section>
  )
}

function AssetCandidateCard({
  activeEpisodeId,
  asset,
  graphNode,
  lockAsset,
  onToggleSelect,
  selected,
}: {
  activeEpisodeId?: string
  asset: Asset
  graphNode?: StoryMapItem
  lockAsset: ReturnType<typeof useLockAsset>
  onToggleSelect: () => void
  selected: boolean
}) {
  const tone = assetStatusTone(asset.status)
  const canLock = activeEpisodeId && asset.status !== 'ready' && asset.status !== 'archived'

  return (
    <article className={selected ? `asset-card ${tone} selected` : `asset-card ${tone}`}>
      <div className="asset-card-header">
        <div>
          <span className="section-kicker">{assetKindLabel(asset.kind)}</span>
          <strong>{graphNode?.name ?? asset.purpose}</strong>
        </div>
        <div className="asset-card-topbar">
          <span className={`asset-status-chip ${tone}`}>{assetStatusLabel(asset.status)}</span>
          <button
            className={selected ? 'asset-select-toggle active' : 'asset-select-toggle'}
            onClick={onToggleSelect}
            type="button"
          >
            {selected ? '已选' : '选择'}
          </button>
        </div>
      </div>
      <p>{graphNode?.description ?? `${assetKindLabel(asset.kind)}候选资产 ${asset.purpose}`}</p>
      <div className="asset-meta-list">
        <span>节点 {asset.purpose}</span>
        <span className="asset-uri">{asset.uri.replace('manmu://', '')}</span>
      </div>
      <div className="asset-card-footer">
        <small>{graphNode ? `${graphNode.code} · ${graphNode.name}` : '等待图谱命名'}</small>
        <button
          disabled={!canLock || lockAsset.isPending}
          onClick={() =>
            activeEpisodeId &&
            lockAsset.mutate({
              assetId: asset.id,
              episodeId: activeEpisodeId,
            })
          }
          type="button"
        >
          {asset.status === 'ready' ? (
            <>
              <Lock aria-hidden="true" />
              已锁定
            </>
          ) : lockAsset.isPending ? (
            '锁定中...'
          ) : (
            <>
              <Lock aria-hidden="true" />
              锁定为参考资产
            </>
          )}
        </button>
      </div>
    </article>
  )
}

function assetKindLabel(kind: string) {
  const labels: Record<string, string> = {
    character: '角色资产',
    prop: '道具资产',
    scene: '场景资产',
  }
  return labels[kind] ?? kind
}

function formatSaveTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '刚刚'
  }
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function assetStatusLabel(status: AssetStatus) {
  const labels: Record<AssetStatus, string> = {
    archived: '已归档',
    draft: '待锁定',
    failed: '生成失败',
    generating: '生成中',
    ready: '已锁定',
  }
  return labels[status]
}

function assetStatusTone(status: AssetStatus) {
  const tones: Record<AssetStatus, AssetStatus> = {
    archived: 'archived',
    draft: 'draft',
    failed: 'failed',
    generating: 'generating',
    ready: 'ready',
  }
  return tones[status]
}

function buildBatchReferenceAssignments(
  draft: CharacterBibleDraft,
  readyAssets: Asset[],
  mode: 'fill-empty' | 'reassign-all',
): CharacterBibleDraft {
  const enabledAngles = draft.reference_angles
  if (enabledAngles.length === 0 || readyAssets.length === 0) {
    return draft
  }

  const existingAssignments =
    mode === 'fill-empty'
      ? (draft.reference_assets ?? []).filter(
          (item) =>
            enabledAngles.includes(item.angle) &&
            readyAssets.some((asset) => asset.id === item.asset_id),
        )
      : []

  const usedAssetIds = new Set(existingAssignments.map((item) => item.asset_id))
  const availableAssets = readyAssets.filter((asset) => !usedAssetIds.has(asset.id))
  const assignedAngles = new Set(existingAssignments.map((item) => item.angle))
  const nextAssignments = [...existingAssignments]

  enabledAngles.forEach((angle) => {
    if (assignedAngles.has(angle)) {
      return
    }

    const asset = availableAssets.shift()
    if (!asset) {
      return
    }

    nextAssignments.push({ angle, asset_id: asset.id })
  })

  return {
    ...draft,
    reference_assets: nextAssignments,
  }
}

const assetStatusFilters: Array<{ label: string; value: 'all' | AssetStatus }> = [
  { label: '全部', value: 'all' },
  { label: '待锁定', value: 'draft' },
  { label: '生成中', value: 'generating' },
  { label: '已锁定', value: 'ready' },
]
