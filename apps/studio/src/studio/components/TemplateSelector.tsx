import type { ShortVideoTemplate } from '../../api/types'

interface TemplateSelectorProps {
  templates: ShortVideoTemplate[]
  selectedTemplate: ShortVideoTemplate | null
  onSelectTemplate: (template: ShortVideoTemplate) => void
}

export default function TemplateSelector({
  templates,
  selectedTemplate,
  onSelectTemplate,
}: TemplateSelectorProps) {
  if (templates.length === 0) {
    return (
      <div className="rounded-lg border border-gray-200 p-6 text-center text-gray-500">
        暂无可用模板
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {templates.map((template) => (
        <div
          key={template.id}
          className={`rounded-lg border-2 p-4 cursor-pointer transition-all ${
            selectedTemplate?.id === template.id
              ? 'border-blue-500 bg-blue-50'
              : 'border-gray-200 hover:border-gray-300'
          }`}
          onClick={() => onSelectTemplate(template)}
        >
          <div className="space-y-2">
            <h3 className="font-semibold text-gray-900">{template.name}</h3>
            <p className="text-sm text-gray-600 line-clamp-2">
              {template.description}
            </p>
            <div className="inline-block rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700">
              {template.category}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
