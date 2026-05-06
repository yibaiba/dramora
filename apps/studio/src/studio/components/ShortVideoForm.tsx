import type {
  ShortVideoParameters,
  ShortVideoParameterValue,
  ShortVideoTemplateConfig,
  ShortVideoTemplateField,
} from '../../api/types'

interface ShortVideoFormProps {
  config: ShortVideoTemplateConfig
  parameters: ShortVideoParameters
  onChange: (parameters: ShortVideoParameters) => void
}

function updateParameters(
  parameters: ShortVideoParameters,
  fieldName: string,
  value: ShortVideoParameterValue,
) {
  return {
    ...parameters,
    [fieldName]: value,
  }
}

function renderField(
  field: ShortVideoTemplateField,
  value: ShortVideoParameterValue,
  onValueChange: (next: ShortVideoParameterValue) => void,
) {
  const sharedProps = {
    id: field.name,
    required: field.required,
  }

  switch (field.type) {
    case 'textarea':
      return (
        <textarea
          {...sharedProps}
          rows={4}
          placeholder={field.placeholder}
          value={typeof value === 'string' ? value : ''}
          onChange={(event) => onValueChange(event.target.value)}
        />
      )
    case 'number':
      return (
        <input
          {...sharedProps}
          type="number"
          min={field.min}
          max={field.max}
          placeholder={field.placeholder}
          value={typeof value === 'number' ? value : ''}
          onChange={(event) => {
            const nextValue = event.target.value.trim()
            onValueChange(nextValue === '' ? null : Number(nextValue))
          }}
        />
      )
    case 'select':
      return (
        <select
          {...sharedProps}
          value={typeof value === 'string' ? value : ''}
          onChange={(event) => onValueChange(event.target.value)}
        >
          <option value="">{field.placeholder ?? '请选择'}</option>
          {(field.options ?? []).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      )
    default:
      return (
        <input
          {...sharedProps}
          type="text"
          placeholder={field.placeholder}
          value={typeof value === 'string' ? value : ''}
          onChange={(event) => onValueChange(event.target.value)}
        />
      )
  }
}

export default function ShortVideoForm({
  config,
  parameters,
  onChange,
}: ShortVideoFormProps) {
  const fields = config.fields ?? []

  if (fields.length === 0) {
    return <p className="short-video-form-empty">该模板暂无可配置参数。</p>
  }

  return (
    <div className="short-video-form-grid">
      {fields.map((field) => (
        <label key={field.name} className={`short-video-form-field${field.type === 'textarea' ? ' is-full' : ''}`} htmlFor={field.name}>
          <span>
            {field.label}
            {field.required ? <em> *</em> : null}
          </span>
          {renderField(field, parameters[field.name] ?? null, (nextValue) =>
            onChange(updateParameters(parameters, field.name, nextValue)),
          )}
          {field.description ? <small>{field.description}</small> : null}
        </label>
      ))}
    </div>
  )
}
