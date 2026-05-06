import { Film } from 'lucide-react'
import type { ShortVideoTemplate } from '../../api/types'
import { StatePlaceholder } from './StatePlaceholder'

interface TemplateSelectorProps {
  templates: ShortVideoTemplate[]
  selectedTemplate: ShortVideoTemplate | null
  onSelectTemplate: (template: ShortVideoTemplate) => void
}

const CATEGORY_LABELS: Record<ShortVideoTemplate['category'], string> = {
  'product-focus': '产品主推',
  promotion: '促销转化',
  'usage-scenario': '场景种草',
}

export default function TemplateSelector({
  templates,
  selectedTemplate,
  onSelectTemplate,
}: TemplateSelectorProps) {
  if (templates.length === 0) {
    return (
      <StatePlaceholder
        tone="empty"
        title="暂无可用模板"
        description="先补充模板配置，再开始批量生成电商短视频。"
        icon={Film}
      />
    )
  }

  return (
    <div className="short-video-template-list" role="list" aria-label="短视频模板列表">
      {templates.map((template) => {
        const selected = selectedTemplate?.id === template.id
        return (
          <button
            key={template.id}
            type="button"
            className={`short-video-template-card${selected ? ' is-selected' : ''}`}
            onClick={() => onSelectTemplate(template)}
          >
            <div className="short-video-template-topline">
              <span className="short-video-pill">{CATEGORY_LABELS[template.category]}</span>
              <span className="short-video-template-meta">{new Date(template.updatedAt).toLocaleDateString()}</span>
            </div>
            <strong>{template.name}</strong>
            <p>{template.description}</p>
          </button>
        )
      })}
    </div>
  )
}
