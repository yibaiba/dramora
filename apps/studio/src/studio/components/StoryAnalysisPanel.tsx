import { BookOpenText } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useCreateStorySource, useStorySources } from '../../api/hooks'
import type { Episode, StoryAnalysis } from '../../api/types'
import type { AnalysisTemplate } from './analysisTemplates'
import { agentRoleLabel } from '../utils'

type StoryAnalysisPanelProps = {
  activeEpisode?: Episode
  analyses: StoryAnalysis[]
  draftNotice?: string | null
  onDraftNoticeChange: (value: string | null) => void
  onSourceTextChange: (value: string) => void
  sourceComposerFocusToken: number
  sourceText: string
  selectedTemplate?: AnalysisTemplate
}

export function StoryAnalysisPanel({
  activeEpisode,
  analyses,
  draftNotice,
  onDraftNoticeChange,
  onSourceTextChange,
  sourceComposerFocusToken,
  sourceText,
  selectedTemplate,
}: StoryAnalysisPanelProps) {
  const createSource = useCreateStorySource(activeEpisode?.id)
  const { data: sources = [] } = useStorySources(activeEpisode?.id)
  const sourceTextRef = useRef<HTMLTextAreaElement | null>(null)
  const [sourceTitle, setSourceTitle] = useState('')
  const latestAnalysis = analyses[0]
  const latestSource = sources[0]

  useEffect(() => {
    if (sourceComposerFocusToken === 0) return
    requestAnimationFrame(() => sourceTextRef.current?.focus())
  }, [sourceComposerFocusToken])

  const submitSource = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!activeEpisode || sourceText.trim() === '') return
    createSource.mutate(
      {
        content_text: sourceText,
        language: 'zh-CN',
        source_type: 'novel',
        title: sourceTitle,
      },
      {
        onSuccess: () => {
          onDraftNoticeChange(null)
          onSourceTextChange('')
          setSourceTitle('')
        },
      },
    )
  }

  const appendTemplateHint = (hint: string) => {
    const nextText = (() => {
      const nextHint = `# 模板提示\n${hint}`
      if (sourceText.includes(hint)) return sourceText
      if (sourceText.trim() === '') return nextHint
      return `${sourceText.trim()}\n\n${nextHint}`
    })()
    onSourceTextChange(nextText)
  }

  return (
    <section className="novel-analysis-panel" aria-label="小说多 Agent 解析">
      <form className="novel-source-card" onSubmit={submitSource}>
        <div className="panel-title-row">
          <div>
            <span>小说输入</span>
            <strong>多 Agent 剧情拆解</strong>
          </div>
          <small>{latestSource ? `最新：${latestSource.title || '未命名素材'}` : '等待原文'}</small>
        </div>
        {selectedTemplate ? (
          <div className="template-guidance-card">
            <div className="panel-title-row">
              <div>
                <span>当前模板</span>
                <strong>{selectedTemplate.name}</strong>
              </div>
              <small>{selectedTemplate.tone}</small>
            </div>
            <p>{selectedTemplate.description}</p>
            <div className="template-guidance-row">
              {selectedTemplate.hints.map((hint) => (
                <button
                  className="template-hint-chip"
                  key={hint}
                  onClick={() => appendTemplateHint(hint)}
                  type="button"
                >
                  {hint}
                </button>
              ))}
            </div>
          </div>
        ) : null}
        {draftNotice ? <div className="draft-note">{draftNotice}</div> : null}
        <label>
          <span>素材标题</span>
          <input
            disabled={!activeEpisode || createSource.isPending}
            onChange={(event) => setSourceTitle(event.target.value)}
            placeholder="例如：天门试炼 第一章"
            value={sourceTitle}
          />
        </label>
        <label>
          <span>小说或故事原文</span>
          <textarea
            disabled={!activeEpisode || createSource.isPending}
            onChange={(event) => onSourceTextChange(event.target.value)}
            placeholder={`粘贴本集小说片段，保存后启动故事解析；当前可参考 ${selectedTemplate?.name ?? '通用'} 模板提示补齐风格、节奏和镜头意图。`}
            ref={sourceTextRef}
            rows={6}
            value={sourceText}
          />
        </label>
        <button
          className="primary-inline-action"
          disabled={!activeEpisode || sourceText.trim() === '' || createSource.isPending}
          type="submit"
        >
          <BookOpenText aria-hidden="true" />
          保存故事源
        </button>
      </form>
      <AnalysisResultCard analysis={latestAnalysis} />
    </section>
  )
}

export function AnalysisResultCard({ analysis }: { analysis?: StoryAnalysis }) {
  if (!analysis) {
    return (
      <div className="analysis-result-card empty">
        <strong>等待解析结果</strong>
        <span>
          保存故事源并启动故事解析后，这里会展示大纲、人物、场景、道具和 Agent 输出。
        </span>
      </div>
    )
  }

  return (
    <div className="analysis-result-card">
      <div className="panel-title-row">
        <div>
          <span>解析结果 v{analysis.version}</span>
          <strong>{analysis.summary}</strong>
        </div>
      </div>
      <div className="analysis-columns">
        <AnalysisList
          title="剧情大纲"
          values={analysis.outline.map((beat) => `${beat.code} ${beat.title}：${beat.summary}`)}
        />
        <AnalysisList title="人物" values={analysis.character_seeds} />
        <AnalysisList title="场景" values={analysis.scene_seeds} />
        <AnalysisList title="道具" values={analysis.prop_seeds} />
      </div>
      <div className="agent-output-row">
        {analysis.agent_outputs.map((agent) => (
          <span className="agent-chip" key={agent.role}>
            {agentRoleLabel(agent.role)} · {agent.status}
          </span>
        ))}
      </div>
    </div>
  )
}

function AnalysisList({ title, values }: { title: string; values: string[] }) {
  return (
    <div className="analysis-list">
      <strong>{title}</strong>
      {values.length === 0 ? <span>等待输出</span> : null}
      {values.slice(0, 4).map((value) => (
        <span key={value}>{value}</span>
      ))}
    </div>
  )
}
