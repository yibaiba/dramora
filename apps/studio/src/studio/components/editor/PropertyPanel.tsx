import { Trash2, Download } from 'lucide-react'
import { useTimelineStore } from '../../lib/editor/timeline-store'

interface PropertyPanelProps {
  onExportClick?: () => void
}

export function PropertyPanel({ onExportClick }: PropertyPanelProps) {
  const currentClip = useTimelineStore((state) => state.currentClip)
  const updateClipProperties = useTimelineStore((state) => state.updateClipProperties)
  const removeClip = useTimelineStore((state) => state.removeClip)
  const timeline = useTimelineStore((state) => state.timeline)

  if (!currentClip) {
    return (
      <div className="property-panel">
        <div className="property-panel-empty">
          <p className="text-sm text-muted">选择片段查看属性</p>
        </div>
      </div>
    )
  }

  const track = timeline.tracks.find((t) => t.id === currentClip.trackId)
  const durationSeconds = currentClip.duration / 1000
  const startSeconds = currentClip.startTime / 1000

  const handleSpeedChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const speed = parseFloat(e.target.value)
    if (!isNaN(speed) && speed > 0) {
      updateClipProperties(currentClip.id, { speed })
    }
  }

  const handleOpacityChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const opacity = parseFloat(e.target.value)
    if (!isNaN(opacity) && opacity >= 0 && opacity <= 1) {
      updateClipProperties(currentClip.id, { opacity })
    }
  }

  const handleDelete = () => {
    if (window.confirm(`确定要删除这个片段吗？`)) {
      removeClip(currentClip.id)
    }
  }

  return (
    <div className="property-panel">
      <div className="property-panel-header">
        <h3>片段属性</h3>
      </div>

      <div className="property-panel-content">
        {/* Clip Info */}
        <div className="property-section">
          <label className="property-label">轨道类型</label>
          <div className="property-value">{track?.type === 'video' ? '视频' : track?.type === 'audio' ? '音频' : '字幕'}</div>
        </div>

        {/* Timing */}
        <div className="property-section">
          <label className="property-label">开始时间</label>
          <div className="property-value">{startSeconds.toFixed(2)}s</div>
        </div>

        <div className="property-section">
          <label className="property-label">持续时间</label>
          <div className="property-value">{durationSeconds.toFixed(2)}s</div>
        </div>

        {/* Speed Control */}
        <div className="property-section">
          <label className="property-label">速度</label>
          <div className="property-input-group">
            <input
              type="range"
              min="0.25"
              max="4"
              step="0.25"
              value={currentClip.properties.speed}
              onChange={handleSpeedChange}
              className="property-slider"
            />
            <span className="property-input-value">{currentClip.properties.speed.toFixed(2)}x</span>
          </div>
        </div>

        {/* Opacity Control (video only) */}
        {track?.type === 'video' && (
          <div className="property-section">
            <label className="property-label">不透明度</label>
            <div className="property-input-group">
              <input
                type="range"
                min="0"
                max="1"
                step="0.05"
                value={currentClip.properties.opacity}
                onChange={handleOpacityChange}
                className="property-slider"
              />
              <span className="property-input-value">{(currentClip.properties.opacity * 100).toFixed(0)}%</span>
            </div>
          </div>
        )}

        {/* Action Buttons */}
        <div className="property-section property-section-buttons">
          <button
            onClick={handleDelete}
            className="property-button property-button-danger"
            type="button"
            title="删除片段"
          >
            <Trash2 size={16} />
            删除
          </button>
          {onExportClick && (
            <button
              onClick={onExportClick}
              className="property-button property-button-primary"
              type="button"
              title="导出视频"
            >
              <Download size={16} />
              导出
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
