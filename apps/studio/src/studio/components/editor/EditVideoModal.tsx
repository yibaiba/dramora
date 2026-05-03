import { X, RotateCcw, RotateCw, Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTimelineStore } from '../../lib/editor/timeline-store'
import type { Timeline, Track } from '../../lib/editor/types'
import { TimelineCanvas } from './TimelineCanvas'
import { PropertyPanel } from './PropertyPanel'
import { ExportDialog } from './ExportDialog'

interface EditVideoModalProps {
  isOpen: boolean
  videoId?: string
  videoUrl?: string
  videoTitle?: string
  onClose: () => void
  onSave?: (timeline: Timeline) => void
}

export function EditVideoModal({ isOpen, videoUrl, videoTitle, onClose, onSave }: EditVideoModalProps) {
  const timeline = useTimelineStore((state) => state.timeline)
  const undo = useTimelineStore((state) => state.undo)
  const redo = useTimelineStore((state) => state.redo)
  const canUndo = useTimelineStore((state) => state.canUndo())
  const canRedo = useTimelineStore((state) => state.canRedo())
  const reset = useTimelineStore((state) => state.reset)
  const initializeTimeline = useTimelineStore((state) => state.initializeTimeline)

  const [showExportDialog, setShowExportDialog] = useState(false)

  // Initialize timeline with a video track if opening with a URL
  useEffect(() => {
    if (isOpen && videoUrl && timeline.tracks.length === 0) {
      const videoTrack: Track = {
        id: `track-${Date.now()}`,
        type: 'video',
        name: '视频轨',
        clips: [],
        visible: true,
        locked: false,
        height: 60,
      }

      const initialTimeline: Timeline = {
        tracks: [videoTrack],
        duration: 10000, // 10 seconds default
        fps: 30,
      }

      initializeTimeline(initialTimeline)
    }
  }, [isOpen, videoUrl, timeline.tracks.length, initializeTimeline])

  if (!isOpen) return null

  const handleSave = () => {
    if (onSave) {
      onSave(timeline)
    }
    handleClose()
  }

  const handleClose = () => {
    reset()
    setShowExportDialog(false)
    onClose()
  }

  const handleExportClick = () => {
    setShowExportDialog(true)
  }

  return (
    <>
      <div className="edit-modal-overlay" onClick={handleClose}>
        <div className="edit-modal" onClick={(e) => e.stopPropagation()}>
          {/* Header */}
          <div className="edit-modal-header">
            <div>
              <h2>{videoTitle ? `编辑 - ${videoTitle}` : '编辑视频'}</h2>
              {videoUrl && <p className="text-xs text-muted">{videoUrl.split('/').pop()}</p>}
            </div>
            <button
              onClick={handleClose}
              className="edit-modal-close"
              type="button"
              title="关闭编辑器"
              aria-label="关闭"
            >
              <X size={20} />
            </button>
          </div>

          {/* Body */}
          <div className="edit-modal-body">
            {/* Timeline Section */}
            <div className="edit-timeline-section">
              <div className="timeline-header">
                <h3>时间线</h3>
                <span className="timeline-duration">{(timeline.duration / 1000).toFixed(1)}s</span>
              </div>
              <TimelineCanvas />
            </div>

            {/* Property Panel */}
            <div className="edit-property-section">
              <PropertyPanel onExportClick={handleExportClick} />
            </div>
          </div>

          {/* Footer */}
          <div className="edit-modal-footer">
            <div className="edit-footer-left">
              <button
                onClick={undo}
                disabled={!canUndo}
                className="footer-button footer-button-secondary"
                type="button"
                title="撤销"
              >
                <RotateCcw size={16} />
                撤销
              </button>
              <button
                onClick={redo}
                disabled={!canRedo}
                className="footer-button footer-button-secondary"
                type="button"
                title="重做"
              >
                <RotateCw size={16} />
                重做
              </button>
              <button
                onClick={() => setShowExportDialog(true)}
                className="footer-button footer-button-secondary"
                type="button"
                title="导出视频"
              >
                <Download size={16} />
                导出
              </button>
            </div>

            <div className="edit-footer-right">
              <button onClick={handleClose} className="footer-button footer-button-secondary" type="button">
                取消
              </button>
              <button onClick={handleSave} className="footer-button footer-button-primary" type="button">
                保存
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Export Dialog */}
      <ExportDialog
        isOpen={showExportDialog}
        timeline={timeline}
        videoUrl={videoUrl}
        videoTitle={videoTitle}
        onClose={() => setShowExportDialog(false)}
      />
    </>
  )
}
