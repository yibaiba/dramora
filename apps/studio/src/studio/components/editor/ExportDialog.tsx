import { useState, useEffect } from 'react'
import { X, AlertCircle } from 'lucide-react'
import type { Timeline } from '../../lib/editor/types'
import { initFFmpeg, unloadFFmpeg } from '../../lib/editor/ffmpeg-worker'
import {
  calculateExportSize,
  downloadBlob,
  estimateExportDuration,
  generateExportFilename,
  getQualityDetails,
  getQualityLabel,
  validateClipsForExport,
} from '../../lib/editor/export-helpers'

interface ExportDialogProps {
  isOpen: boolean
  timeline: Timeline
  videoUrl?: string
  videoTitle?: string
  onClose: () => void
}

type Quality = 'low' | 'medium' | 'high' | 'very-high'

export function ExportDialog({ isOpen, timeline, videoTitle = 'video', onClose }: ExportDialogProps) {
  const [quality, setQuality] = useState<Quality>('medium')
  const [isExporting, setIsExporting] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [estimatedTime, setEstimatedTime] = useState(0)

  useEffect(() => {
    if (isOpen) {
      const estimated = estimateExportDuration(timeline, quality)
      setEstimatedTime(estimated)
    }
  }, [isOpen, timeline, quality])

  if (!isOpen) return null

  const validationError = validateClipsForExport(timeline)
  const estimatedSize = calculateExportSize(timeline.duration, quality)
  const qualityDetails = getQualityDetails(quality)

  const handleExport = async () => {
    setError(null)
    setIsExporting(true)
    setProgress(0)

    try {
      // Initialize FFmpeg
      await initFFmpeg()

      // For now, we'll do a simple export of a placeholder
      // In a real scenario, we'd fetch the video and process clips
      // This is a simplified version that demonstrates the flow

      const dummyBlob = new Blob(['mock video data'], { type: 'video/mp4' })

      // Simulate progress
      for (let i = 0; i < 10; i++) {
        await new Promise((resolve) => setTimeout(resolve, (estimatedTime * 1000) / 10))
        setProgress((i + 1) / 10)
      }

      // Create output
      const filename = generateExportFilename(videoTitle, 'mp4')
      downloadBlob(dummyBlob, filename)

      setProgress(1)
      setTimeout(() => {
        unloadFFmpeg()
        onClose()
      }, 1000)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Export failed'
      setError(message)
      setIsExporting(false)
      unloadFFmpeg()
    }
  }

  const handleCancel = () => {
    if (!isExporting) {
      onClose()
    }
  }

  return (
    <div className="export-dialog-overlay" onClick={handleCancel}>
      <div className="export-dialog" onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div className="export-dialog-header">
          <h2>导出视频</h2>
          <button
            onClick={onClose}
            disabled={isExporting}
            className="export-dialog-close"
            type="button"
            aria-label="关闭"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="export-dialog-body">
          {/* Error Message */}
          {error && (
            <div className="export-error-box">
              <AlertCircle size={16} />
              <span>{error}</span>
            </div>
          )}

          {validationError && (
            <div className="export-error-box">
              <AlertCircle size={16} />
              <span>{validationError}</span>
            </div>
          )}

          {/* Quality Selection */}
          {!validationError && (
            <>
              <div className="export-section">
                <label className="export-label">导出质量</label>
                <div className="export-quality-options">
                  {(['low', 'medium', 'high', 'very-high'] as const).map((q) => (
                    <div key={q} className="export-quality-item">
                      <input
                        type="radio"
                        id={`quality-${q}`}
                        name="quality"
                        value={q}
                        checked={quality === q}
                        onChange={(e) => setQuality(e.target.value as Quality)}
                        disabled={isExporting}
                        className="export-quality-radio"
                      />
                      <label htmlFor={`quality-${q}`} className="export-quality-label">
                        <div className="export-quality-name">{getQualityLabel(q)}</div>
                        <div className="export-quality-details">
                          <span>{qualityDetails.resolution}</span>
                          <span>•</span>
                          <span>{qualityDetails.bitrate}</span>
                        </div>
                      </label>
                    </div>
                  ))}
                </div>
              </div>

              {/* Export Info */}
              <div className="export-info-grid">
                <div className="export-info-item">
                  <span className="export-info-label">文件大小</span>
                  <span className="export-info-value">约 {estimatedSize.toFixed(0)} MB</span>
                </div>
                <div className="export-info-item">
                  <span className="export-info-label">预计时间</span>
                  <span className="export-info-value">{estimatedTime}s</span>
                </div>
                <div className="export-info-item">
                  <span className="export-info-label">视频时长</span>
                  <span className="export-info-value">{(timeline.duration / 1000).toFixed(1)}s</span>
                </div>
              </div>

              {/* Progress Bar */}
              {isExporting && (
                <div className="export-progress-section">
                  <div className="export-progress-label">
                    <span>导出中...</span>
                    <span className="export-progress-percent">{Math.round(progress * 100)}%</span>
                  </div>
                  <div className="export-progress-bar">
                    <div
                      className="export-progress-fill"
                      style={{ width: `${Math.round(progress * 100)}%` }}
                    />
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="export-dialog-footer">
          <button
            onClick={handleCancel}
            disabled={isExporting}
            className="export-button export-button-secondary"
            type="button"
          >
            取消
          </button>
          <button
            onClick={handleExport}
            disabled={isExporting || Boolean(validationError)}
            className="export-button export-button-primary"
            type="button"
          >
            {isExporting ? '导出中...' : '开始导出'}
          </button>
        </div>
      </div>
    </div>
  )
}
