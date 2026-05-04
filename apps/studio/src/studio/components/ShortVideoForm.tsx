interface ShortVideoFormProps {
  config: Record<string, any>
  parameters: Record<string, any>
  onChange: (parameters: Record<string, any>) => void
}

export default function ShortVideoForm({
  config,
  parameters,
  onChange,
}: ShortVideoFormProps) {
  const fields = config?.fields || []

  const handleFieldChange = (fieldName: string, value: any) => {
    onChange({
      ...parameters,
      [fieldName]: value,
    })
  }

  if (fields.length === 0) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-gray-500">该模板暂无可配置参数</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {fields.map((field: any) => (
        <div key={field.name} className="space-y-2">
          <label htmlFor={field.name} className="text-sm font-medium text-gray-700">
            {field.label}
            {field.required && <span className="text-red-500"> *</span>}
          </label>

          {field.type === 'text' && (
            <input
              id={field.name}
              type="text"
              placeholder={field.placeholder}
              value={parameters[field.name] || ''}
              onChange={(e) => handleFieldChange(field.name, e.target.value)}
              required={field.required}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          )}

          {field.type === 'number' && (
            <input
              id={field.name}
              type="number"
              placeholder={field.placeholder}
              value={parameters[field.name] || ''}
              onChange={(e) => handleFieldChange(field.name, parseFloat(e.target.value))}
              min={field.min}
              max={field.max}
              required={field.required}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          )}

          {field.type === 'textarea' && (
            <textarea
              id={field.name}
              placeholder={field.placeholder}
              value={parameters[field.name] || ''}
              onChange={(e) => handleFieldChange(field.name, e.target.value)}
              rows={4}
              required={field.required}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          )}

          {field.type === 'select' && (
            <select
              value={parameters[field.name] || ''}
              onChange={(e) => handleFieldChange(field.name, e.target.value)}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="">{field.placeholder}</option>
              {(field.options || []).map((option: any) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          )}

          {field.description && (
            <p className="text-xs text-gray-500">{field.description}</p>
          )}
        </div>
      ))}
    </div>
  )
}
