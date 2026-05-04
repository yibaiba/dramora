import { useState } from 'react'
import { ChevronDown, ChevronUp, Eye, EyeOff, Lock, Unlock, X, Plus } from 'lucide-react'
import type { Track, TrackType } from '../../lib/editor/types'
import { useTimelineStore } from '../../lib/editor/timeline-store'

interface TrackPanelProps {
  tracks: Track[]
  selectedTrackId: string | null
}

const TRACK_TYPE_LABELS: Record<TrackType, string> = {
  video: '视频',
  audio: '音频',
  subtitle: '字幕',
}

const TRACK_TYPE_COLORS: Record<TrackType, string> = {
  video: '#3b82f6', // blue
  audio: '#8b5cf6', // purple
  subtitle: '#10b981', // green
}

export function TrackPanel({ tracks, selectedTrackId }: TrackPanelProps) {
  const [expandedTrackId, setExpandedTrackId] = useState<string | null>(null)
  const {
    selectTrack,
    removeTrack,
    toggleTrackVisibility,
    toggleTrackLock,
  } = useTimelineStore()

  const handleAddTrack = (type: TrackType) => {
    const newTrack: Track = {
      id: `track-${Date.now()}`,
      type,
      name: `${TRACK_TYPE_LABELS[type]} ${tracks.filter((t) => t.type === type).length + 1}`,
      clips: [],
      visible: true,
      locked: false,
      height: type === 'video' ? 100 : type === 'audio' ? 60 : 40,
    }
    useTimelineStore.setState((state) => ({
      timeline: {
        ...state.timeline,
        tracks: [...state.timeline.tracks, newTrack],
      },
    }))
  }

  return (
    <div className="track-panel">
      {/* Panel Header */}
      <div className="track-panel-header">
        <h3 className="track-panel-title">轨道</h3>
        <div className="track-panel-actions">
          <button
            onClick={() => handleAddTrack('video')}
            className="track-add-button"
            title="添加视频轨道"
          >
            <Plus size={14} />
          </button>
          <button
            onClick={() => handleAddTrack('audio')}
            className="track-add-button"
            title="添加音频轨道"
          >
            <Plus size={14} />
          </button>
          <button
            onClick={() => handleAddTrack('subtitle')}
            className="track-add-button"
            title="添加字幕轨道"
          >
            <Plus size={14} />
          </button>
        </div>
      </div>

      {/* Tracks List */}
      <div className="track-list">
        {tracks.length === 0 ? (
          <div className="track-empty">
            <p>没有轨道。点击上方按钮添加轨道。</p>
          </div>
        ) : (
          tracks.map((track) => (
            <div
              key={track.id}
              className={`track-item ${selectedTrackId === track.id ? 'selected' : ''}`}
              onClick={() => selectTrack(track.id)}
            >
              {/* Track Header */}
              <div className="track-item-header">
                {/* Expand/Collapse */}
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    setExpandedTrackId(expandedTrackId === track.id ? null : track.id)
                  }}
                  className="track-expand-button"
                  title={expandedTrackId === track.id ? '折叠' : '展开'}
                >
                  {expandedTrackId === track.id ? (
                    <ChevronDown size={16} />
                  ) : (
                    <ChevronUp size={16} />
                  )}
                </button>

                {/* Track Color Indicator */}
                <div
                  className="track-color-indicator"
                  style={{
                    backgroundColor: TRACK_TYPE_COLORS[track.type],
                  }}
                />

                {/* Track Name */}
                <div className="track-name">
                  <span className="track-type">
                    {TRACK_TYPE_LABELS[track.type]}
                  </span>
                  <span className="track-label">{track.name}</span>
                </div>

                {/* Track Controls */}
                <div className="track-controls">
                  {/* Visibility Toggle */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      toggleTrackVisibility(track.id)
                    }}
                    className="track-control-button"
                    title={track.visible ? '隐藏轨道' : '显示轨道'}
                  >
                    {track.visible ? (
                      <Eye size={14} />
                    ) : (
                      <EyeOff size={14} />
                    )}
                  </button>

                  {/* Lock Toggle */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      toggleTrackLock(track.id)
                    }}
                    className="track-control-button"
                    title={track.locked ? '解锁轨道' : '锁定轨道'}
                  >
                    {track.locked ? (
                      <Lock size={14} />
                    ) : (
                      <Unlock size={14} />
                    )}
                  </button>

                  {/* Delete */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      if (confirm(`确定删除 "${track.name}" 吗？`)) {
                        removeTrack(track.id)
                      }
                    }}
                    className="track-control-button track-control-button--delete"
                    title="删除轨道"
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>

              {/* Track Details (when expanded) */}
              {expandedTrackId === track.id && (
                <div className="track-item-details">
                  <div className="track-detail-row">
                    <label>轨道高度</label>
                    <input
                      type="range"
                      min="40"
                      max="200"
                      value={track.height}
                      onChange={(e) => {
                        useTimelineStore.setState((state) => ({
                          timeline: {
                            ...state.timeline,
                            tracks: state.timeline.tracks.map((t) =>
                              t.id === track.id
                                ? { ...t, height: parseInt(e.target.value) }
                                : t
                            ),
                          },
                        }))
                      }}
                      className="track-height-slider"
                    />
                    <span className="track-height-value">{track.height}px</span>
                  </div>
                  <div className="track-detail-row">
                    <label>片段数</label>
                    <span className="track-clip-count">{track.clips.length}</span>
                  </div>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
