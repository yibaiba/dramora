import { AlertCircle, Boxes, CheckCircle2, Eye, Layers3, Palette, RotateCcw, Sparkles, UserRound } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { Asset, StoryMapItem } from '../../api/types'
import {
  characterBibleExpressions,
  characterBibleReferenceAngles,
  type CharacterBibleDraft,
  type GraphNodeKind,
} from './assetsGraphDrafts'
import { studioRoutePaths } from '../routes'
import type { StoryMapNodeLinkedShot } from '../utils'

type AssetsGraphInspectorProps = {
  draft?: CharacterBibleDraft
  focusedReferenceAngle?: string
  hasPersistedDraft: boolean
  kind?: GraphNodeKind
  node?: StoryMapItem
  onAssignReferenceAsset: (assetId: string) => void
  onChangeAnchor: (value: string) => void
  onChangeExpression: (expression: string) => void
  onChangeNotes: (value: string) => void
  onChangePalette: (key: keyof CharacterBibleDraft['palette'], value: string) => void
  onChangeReferenceAngle: (angle: string) => void
  onAutoFillReferenceAssignments: () => void
  onClearReferenceAssignments: () => void
  onChangeWardrobe: (value: string) => void
  onClearSelection: () => void
  onFocusReferenceAngle: (angle: string) => void
  onRemoveReferenceAsset: (angle: string) => void
  onReassignReferenceAssignments: () => void
  onResetDraft: () => void
  onSaveDraft: () => void
  relatedAssets: Asset[]
  referenceAssignments: Array<{ angle: string; asset?: Asset }>
  referenceAssignmentStats: {
    assignedCount: number
    totalAngles: number
    unassignedCount: number
  }
  saveMessage?: string
  savePending: boolean
  saveState: 'draft' | 'dirty' | 'error' | 'saved' | 'saving'
  storyboardLinks: StoryMapNodeLinkedShot[]
  storyboardLinkSummary: {
    coveredCount: number
    linkedCount: number
  }
  summary?: {
    anchorReady: boolean
    expressionCount: number
    paletteFilled: number
    referenceAngleCount: number
  }
}

export function AssetsGraphInspector({
  draft,
  focusedReferenceAngle,
  hasPersistedDraft,
  kind,
  node,
  onAssignReferenceAsset,
  onChangeAnchor,
  onChangeExpression,
  onChangeNotes,
  onChangePalette,
  onChangeReferenceAngle,
  onAutoFillReferenceAssignments,
  onClearReferenceAssignments,
  onChangeWardrobe,
  onClearSelection,
  onFocusReferenceAngle,
  onRemoveReferenceAsset,
  onReassignReferenceAssignments,
  onResetDraft,
  onSaveDraft,
  relatedAssets,
  referenceAssignments,
  referenceAssignmentStats,
  saveMessage,
  savePending,
  saveState,
  storyboardLinks,
  storyboardLinkSummary,
  summary,
}: AssetsGraphInspectorProps) {
  if (!node || !kind) {
    return (
      <article className="surface-card assets-graph-inspector">
        <div className="panel-title-row">
          <div>
            <span>Node preview</span>
            <strong>Character Bible / 节点预览</strong>
          </div>
        </div>
        <div className="graph-empty">点击左侧图谱节点后，这里会展示节点预览；角色节点还会展开 Character Bible 编辑视图。</div>
      </article>
    )
  }

  const readyAssets = relatedAssets.filter((asset) => asset.status === 'ready')
  const assignedAnglesByAssetId = referenceAssignments.reduce<Record<string, string[]>>((current, slot) => {
    if (!slot.asset) {
      return current
    }
    return {
      ...current,
      [slot.asset.id]: [...(current[slot.asset.id] ?? []), slot.angle],
    }
  }, {})

  return (
    <article className="surface-card assets-graph-inspector">
      <div className="panel-title-row">
        <div>
          <span>Node preview</span>
          <strong>Character Bible / 节点预览</strong>
        </div>
        <div className="section-actions">
          {kind === 'character' ? (
            <button className="asset-filter-chip" onClick={onResetDraft} type="button">
              <RotateCcw aria-hidden="true" />
              重置草稿
            </button>
          ) : null}
          {kind === 'character' ? (
            <button
              className="primary-inline-action"
              disabled={!draft || draft.anchor.trim() === '' || savePending}
              onClick={onSaveDraft}
              type="button"
            >
              {savePending ? '保存中...' : '保存 Character Bible'}
            </button>
          ) : null}
          <button className="asset-filter-chip" onClick={onClearSelection} type="button">
            清除节点选择
          </button>
        </div>
      </div>

      <div className="asset-node-inspector-grid">
        <section className="asset-node-preview">
          <div className="asset-node-preview-header">
            <div>
              <span className="scene-chip">{node.code}</span>
              <strong>{node.name}</strong>
            </div>
            <span className="asset-status-chip draft">{kindLabel(kind)}</span>
          </div>
          <p>{node.description || '当前节点还没有详细描述，可先通过 Character Bible 草稿补齐。'}</p>
          <div className="asset-node-stat-row">
            <span>候选资产 {relatedAssets.length}</span>
            <span>已锁定 {readyAssets.length}</span>
            <span>创建于 {formatDate(node.created_at)}</span>
          </div>
          <div className="blackboard-chip-row">
            <span className="blackboard-chip">{promptAnchorLabel(kind, node)}</span>
            <span className="blackboard-chip">URI 节点 {`${kind}:${node.code}`}</span>
          </div>
          <div className="node-preview-list">
            {relatedAssets.length === 0 ? (
              <div className="graph-empty">该节点还没有候选资产，可先在上方生成或补齐资产。</div>
            ) : (
              relatedAssets.slice(0, 4).map((asset) => (
                <div className="node-preview-list-item" key={asset.id}>
                  <strong>{asset.uri.replace('manmu://episodes/', '')}</strong>
                  <span>{asset.status === 'ready' ? '已锁定参考' : '待选择候选'}</span>
                </div>
              ))
            )}
          </div>
          <div className="character-bible-section asset-cross-link-section">
            <div className="character-bible-section-header">
              <Boxes aria-hidden="true" />
              <strong>跨页联动</strong>
            </div>
            <small>
              已命中 {storyboardLinkSummary.linkedCount} 个 Storyboard 镜头
              {storyboardLinkSummary.coveredCount > 0
                ? ` · 其中 ${storyboardLinkSummary.coveredCount} 个已具备 Prompt 引用图覆盖`
                : ' · 当前还没有 Prompt 引用图覆盖'}
            </small>
            {storyboardLinks.length === 0 ? (
              <div className="graph-empty">当前 Storyboard 里还没有命中这个节点的镜头；生成更多分镜或补齐镜头文案后，这里会自动联动。</div>
            ) : (
              <div className="node-preview-list">
                {storyboardLinks.map((link) => (
                  <div className="node-preview-list-item node-link-preview-item" key={link.shot.key}>
                    <div className="node-link-preview-head">
                      <div className="node-link-preview-copy">
                        <strong>第 {link.shot.code} 镜 · {link.shot.title}</strong>
                        <span>{link.shot.sceneCode} · {link.shot.sceneName}</span>
                      </div>
                      <span className={link.coverage === 'covered' ? 'asset-status-chip ready' : 'asset-status-chip draft'}>
                        {link.coverage === 'covered' ? 'Prompt 已覆盖' : '镜头已命中'}
                      </span>
                    </div>
                    <span>{link.reason}</span>
                    <Link
                      className="asset-filter-chip node-preview-inline-action"
                      to={studioRoutePaths.storyboard}
                      state={{
                        fromAssetsGraph: true,
                        selectedNodeCode: node.code,
                        selectedNodeKind: kind,
                        selectedNodeName: node.name,
                        selectedShotCode: link.shot.code,
                      }}
                    >
                      前往 Storyboard 查看
                    </Link>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>

        {kind === 'character' && draft ? (
          <section className="character-bible-card">
            <div className="character-bible-header">
              <div>
                <span>Character Bible</span>
                <strong>角色一致性锚点</strong>
              </div>
              <small>{hasPersistedDraft ? '已接入后端角色节点' : '当前为本地草稿，保存后才会跨页复用'}</small>
            </div>

            <div className="character-bible-status-row" aria-live="polite">
              <span className={`bible-state-chip ${saveState}`}>
                {saveState === 'saving'
                  ? '保存中'
                  : saveState === 'error'
                    ? '保存失败'
                    : saveState === 'dirty'
                      ? '未保存修改'
                      : saveState === 'saved'
                        ? '已同步保存'
                        : '本地草稿'}
              </span>
              {summary ? (
                <div className="character-bible-metrics">
                  <span className={summary.anchorReady ? 'bible-metric-chip active' : 'bible-metric-chip'}>锚点</span>
                  <span className="bible-metric-chip">色板 {summary.paletteFilled}/5</span>
                  <span className="bible-metric-chip">表情 {summary.expressionCount}/6</span>
                  <span className="bible-metric-chip">角度 {summary.referenceAngleCount}/7</span>
                </div>
              ) : null}
            </div>

            {saveMessage ? (
              <div className={saveState === 'error' ? 'bible-save-note error' : 'bible-save-note'}>
                {saveState === 'error' ? <AlertCircle aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                <span>{saveMessage}</span>
              </div>
            ) : null}

            <div className="character-bible-callout">
              <strong>导演提示</strong>
              <p>主描述锚点只写角色外观和稳定识别特征，不写镜头动作、场景或情绪演法，这样后续提示词更稳。</p>
            </div>

            <label className="character-bible-field">
              <span>主描述锚点</span>
              <textarea
                onChange={(event) => onChangeAnchor(event.target.value)}
                rows={4}
                value={draft.anchor}
              />
              <small className={draft.anchor.trim() ? 'character-bible-field-hint' : 'character-bible-field-hint warning'}>
                {draft.anchor.trim()
                  ? `当前 ${draft.anchor.trim().length} 字，建议保留年龄、体态、发型、眼神与标志性服装特征。`
                  : '主描述锚点必填；缺少锚点时不能保存。'}
              </small>
            </label>

            <div className="character-bible-palette-grid">
              {paletteFields.map((field) => (
                <label className="character-bible-field" key={field.key}>
                  <span>{field.label}</span>
                  <div className="palette-input-row">
                    <span
                      className="palette-swatch"
                      style={{ background: isHexColor(draft.palette[field.key]) ? draft.palette[field.key] : 'transparent' }}
                    />
                    <input
                      onChange={(event) => onChangePalette(field.key, event.target.value)}
                      placeholder="#000000"
                      value={draft.palette[field.key]}
                    />
                  </div>
                </label>
              ))}
            </div>

            <div className="character-bible-section">
              <div className="character-bible-section-header">
                <Palette aria-hidden="true" />
                <strong>表情锚点</strong>
              </div>
              <div className="blackboard-chip-row">
                {characterBibleExpressions.map((expression) => (
                  <button
                    className={draft.expressions.includes(expression) ? 'asset-filter-chip active' : 'asset-filter-chip'}
                    key={expression}
                    onClick={() => onChangeExpression(expression)}
                    type="button"
                  >
                    {expression}
                  </button>
                ))}
              </div>
            </div>

            <div className="character-bible-section">
              <div className="character-bible-section-header">
                <Eye aria-hidden="true" />
                <strong>标准角度参考</strong>
              </div>
              <div className="reference-angle-grid">
                {characterBibleReferenceAngles.map((angle) => (
                  <button
                    className={
                      draft.reference_angles.includes(angle)
                        ? 'reference-angle-chip active'
                        : 'reference-angle-chip'
                    }
                    key={angle}
                    onClick={() => onChangeReferenceAngle(angle)}
                    type="button"
                  >
                    {angle}
                  </button>
                ))}
              </div>
              <small>{readyAssets.length > 0 ? `当前已有 ${readyAssets.length} 个锁定参考资产，可在后续版本映射到具体角度。` : '当前尚未有锁定参考资产，建议至少先锁定 1 个角色参考。'}</small>
            </div>

            <div className="character-bible-section reference-workflow-section">
              <div className="character-bible-section-header">
                <Sparkles aria-hidden="true" />
                <strong>引用图工作流</strong>
              </div>
              <small>先选择一个角度槽位，再从下方已锁定参考资产里挂载引用图。保存 Character Bible 后，角度-引用图映射会随 `reference_assets` 一起持久化到角色节点。</small>
              <div className="reference-workflow-toolbar">
                <button
                  className="asset-filter-chip"
                  disabled={referenceAssignmentStats.unassignedCount === 0 || readyAssets.length === 0}
                  onClick={onAutoFillReferenceAssignments}
                  type="button"
                >
                  自动补齐空槽
                </button>
                <button
                  className="asset-filter-chip"
                  disabled={referenceAssignmentStats.totalAngles === 0 || readyAssets.length === 0}
                  onClick={onReassignReferenceAssignments}
                  type="button"
                >
                  按顺序重排
                </button>
                <button
                  className="asset-filter-chip"
                  disabled={referenceAssignments.length === 0}
                  onClick={onClearReferenceAssignments}
                  type="button"
                >
                  清空角度挂载
                </button>
                <span className="asset-filter-summary">
                  已绑定 {referenceAssignmentStats.assignedCount}/{referenceAssignmentStats.totalAngles} · 空槽 {referenceAssignmentStats.unassignedCount} · 当前焦点：{focusedReferenceAngle ?? '先选择一个角度'}
                </span>
              </div>

              <div className="reference-slot-grid">
                {referenceAssignments.length === 0 ? (
                  <div className="graph-empty">先在上方启用至少一个标准角度，下面才会出现可挂载的引用图槽位。</div>
                ) : (
                  referenceAssignments.map((slot) => (
                    <article
                      className={
                        focusedReferenceAngle === slot.angle
                          ? 'reference-slot-card active'
                          : 'reference-slot-card'
                      }
                      key={slot.angle}
                    >
                      <button
                        className="reference-slot-select"
                        onClick={() => onFocusReferenceAngle(slot.angle)}
                        type="button"
                      >
                        <div>
                          <span className="scene-chip">{slot.angle}</span>
                          <strong>{slot.asset ? '已挂载引用图' : '等待挂载引用图'}</strong>
                        </div>
                        <span className="asset-status-chip ready">
                          {slot.asset ? '已绑定' : '待绑定'}
                        </span>
                      </button>
                      {slot.asset ? (
                        <div className="reference-slot-meta">
                          <strong>{slot.asset.uri.replace('manmu://episodes/', '')}</strong>
                          <div className="reference-slot-actions">
                            <span>{slot.asset.status === 'ready' ? '锁定参考资产' : '候选资产'}</span>
                            <button
                              className="asset-filter-chip"
                              onClick={() => onRemoveReferenceAsset(slot.angle)}
                              type="button"
                            >
                              移除
                            </button>
                          </div>
                        </div>
                      ) : (
                        <div className="reference-slot-empty">
                          选择此角度后，点击下方任意一个已锁定资产即可挂载到这个槽位。
                        </div>
                      )}
                    </article>
                  ))
                )}
              </div>

              <div className="reference-bank-grid">
                {readyAssets.length === 0 ? (
                  <div className="graph-empty">当前还没有可用的锁定参考资产；先去候选资产池锁定至少 1 个角色资产。</div>
                ) : (
                  readyAssets.map((asset) => {
                    const assignedAngles = assignedAnglesByAssetId[asset.id] ?? []
                    const isAssignedToFocus =
                      focusedReferenceAngle !== undefined &&
                      assignedAngles.includes(focusedReferenceAngle)

                    return (
                      <button
                        className={isAssignedToFocus ? 'reference-bank-card active' : 'reference-bank-card'}
                        disabled={!focusedReferenceAngle}
                        key={asset.id}
                        onClick={() => onAssignReferenceAsset(asset.id)}
                        type="button"
                      >
                        <div className="reference-bank-header">
                          <span className="section-kicker">{asset.status === 'ready' ? 'Reference asset' : 'Candidate asset'}</span>
                          <strong>{asset.uri.replace('manmu://episodes/', '')}</strong>
                        </div>
                        <span className="reference-bank-meta">
                          {assignedAngles.length > 0
                            ? `已挂到 ${assignedAngles.join(' / ')}`
                            : focusedReferenceAngle
                              ? `点击挂到 ${focusedReferenceAngle}`
                              : '先选择一个角度槽位'}
                        </span>
                      </button>
                    )
                  })
                )}
              </div>
            </div>

            <label className="character-bible-field">
              <span>服装变体</span>
              <textarea
                onChange={(event) => onChangeWardrobe(event.target.value)}
                rows={3}
                value={draft.wardrobe}
              />
            </label>

            <label className="character-bible-field">
              <span>备注</span>
              <textarea
                onChange={(event) => onChangeNotes(event.target.value)}
                rows={3}
                value={draft.notes}
              />
            </label>

            <div className="draft-note">
              <Sparkles aria-hidden="true" />
              <span>保存后会写入角色节点的 `character_bible` 字段，并通过 `story-map` / `storyboard-workspace` 读模型重新回流到前端。</span>
            </div>
          </section>
        ) : (
          <section className="character-bible-card compact">
            <div className="character-bible-header">
              <div>
                <span>{kindLabel(kind)}</span>
                <strong>节点预览建议</strong>
              </div>
            </div>
            <div className="character-bible-section">
              <div className="character-bible-section-header">
                <Layers3 aria-hidden="true" />
                <strong>建议作为提示词锚点</strong>
              </div>
              <p>{promptAnchorLabel(kind, node)}</p>
            </div>
            <div className="draft-note">
              <UserRound aria-hidden="true" />
              <span>当前节点预览已可辅助筛选与锁定参考资产；Character Bible 编辑视图优先对角色节点开放。</span>
            </div>
          </section>
        )}
      </div>
    </article>
  )
}

const paletteFields: Array<{ key: keyof CharacterBibleDraft['palette']; label: string }> = [
  { key: 'skin', label: '肤色' },
  { key: 'hair', label: '发色' },
  { key: 'accent', label: '挑染 / 点缀' },
  { key: 'eyes', label: '眼睛' },
  { key: 'costume', label: '服装主色' },
]

function kindLabel(kind: GraphNodeKind): string {
  const labels: Record<GraphNodeKind, string> = {
    character: '角色节点',
    prop: '道具节点',
    scene: '场景节点',
  }
  return labels[kind]
}

function promptAnchorLabel(kind: GraphNodeKind, node: StoryMapItem): string {
  if (kind === 'character') {
    return `保持 ${node.name} 的外观、发型、眼神与服装主色一致`
  }
  if (kind === 'scene') {
    return `保持 ${node.name} 的空间结构、主光源和场景色调一致`
  }
  return `保持 ${node.name} 的造型比例、材质与关键细节一致`
}

function formatDate(value: string): string {
  if (!value) return '未知时间'
  return value.slice(0, 10)
}

function isHexColor(value: string): boolean {
  return /^#([0-9a-fA-F]{6})$/.test(value)
}
