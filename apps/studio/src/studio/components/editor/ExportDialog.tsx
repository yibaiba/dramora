import { useState, useEffect, useMemo } from 'react'
import { X, AlertCircle } from 'lucide-react'
import type { Timeline } from '../../lib/editor/types'
import { initFFmpeg, unloadFFmpeg, encodeToMP4 } from '../../lib/editor/ffmpeg-worker'
import {
  calculateExportSize,
  downloadBlob,
  estimateExportDuration,
  generateExportFilename,
  getQualityDetails,
  getQualityLabel,
  validateClipsForExport,
  exportToFCPXML,
  exportToPremiere,
  exportToDaVinci,
} from '../../lib/editor/export-helpers'

interface ExportDialogProps {
  isOpen: boolean
  timeline: Timeline
  videoUrl?: string
  videoTitle?: string
  onClose: () => void
}

async function fetchVideoBlob(url: string): Promise<Blob> {
  const response = await fetch(url, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error(`Failed to fetch video: ${response.statusText}`)
  }
  return response.blob()
}

type Quality = 'low' | 'medium' | 'high' | 'very-high'
type Format = 'mp4' | 'fcpxml' | 'premiere' | 'davinci'

export function ExportDialog({ isOpen, timeline, videoUrl, videoTitle = 'video', onClose }: ExportDialogProps) {
  const [quality, setQuality] = useState<Quality>('medium')
  const [format, setFormat] = useState<Format>('mp4')
  const [isExporting, setIsExporting] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)

  const estimatedTime = useMemo(() => {
    if (!isOpen) return 0
    return format === 'mp4' ? estimateExportDuration(timeline, quality) : 2
  }, [isOpen, format, quality, timeline])

  useEffect(() => {
    if (!isOpen || !isExporting) return

    // Cleanup on unmount or when dialog closes
    return () => {
      unloadFFmpeg()
    }
  }, [isOpen, isExporting])

  if (!isOpen) return null

  const validationError = validateClipsForExport(timeline)
  const estimatedSize = format === 'mp4' ? calculateExportSize(timeline.duration, quality) : 5
  const qualityDetails = format === 'mp4' ? getQualityDetails(quality) : null

  const handleExport = async () => {
    setError(null)
    setIsExporting(true)
    setProgress(0)

    try {
      const validationError = validateClipsForExport(timeline)
      if (validationError) {
        setError(validationError)
        setIsExporting(false)
        return
      }

      if (format === 'fcpxml') {
        exportToFCPXML(timeline, videoTitle)
        setProgress(1)
      } else if (format === 'premiere') {
        exportToPremiere(timeline, videoTitle)
        setProgress(1)
      } else if (format === 'davinci') {
        exportToDaVinci(timeline, videoTitle)
        setProgress(1)
      } else {
        // MP4 export with FFmpeg
        await initFFmpeg()

        let videoBlob: Blob
        if (videoUrl) {
          // Fetch actual video from URL
          try {
            setProgress(0.1)
            videoBlob = await fetchVideoBlob(videoUrl)
            setProgress(0.2)
          } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to fetch video'
            throw new Error(`Video fetch error: ${message}`, { cause: err })
          }
        } else {
          // Fallback if no video URL (shouldn't happen in normal flow)
          throw new Error(
            'No video source available. Please open a video from Gallery to edit.',
            { cause: undefined }
          )
        }

        // Encode to MP4 with real FFmpeg
        const filename = generateExportFilename(videoTitle, 'mp4')
        const encodedBlob = await encodeToMP4(videoBlob, quality, (currentProgress, _) => {
          // Map progress from 0.2-0.95 for encoding phase
          const encodingProgress = 0.2 + currentProgress * 0.75
          setProgress(Math.min(0.95, encodingProgress))
        })

        setProgress(1)
        downloadBlob(encodedBlob, filename)
        unloadFFmpeg()
      }

      setTimeout(() => {
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

          {!validationError && (
            <>
              {/* Format Selection */}
              <div className="export-section">
                <label className="export-label">导出格式</label>
                <div className="export-format-options">
                  {(
                    [
                      { value: 'mp4', label: 'MP4', desc: '通用视频格式' },
                      { value: 'fcpxml', label: 'FCPXML', desc: 'Final Cut Pro' },
                      { value: 'premiere', label: 'Premiere', desc: 'Adobe Premiere Pro' },
                      { value: 'davinci', label: 'DaVinci', desc: 'DaVinci Resolve' },
                    ] as const
                  ).map((opt) => (
                    <div key={opt.value} className="export-format-item">
                      <input
                        type="radio"
                        id={`format-${opt.value}`}
                        name="format"
                        value={opt.value}
                        checked={format === opt.value}
                        onChange={(e) => setFormat(e.target.value as Format)}
                        disabled={isExporting}
                        className="export-format-radio"
                      />
                      <label htmlFor={`format-${opt.value}`} className="export-format-label">
                        <div className="export-format-name">{opt.label}</div>
                        <div className="export-format-desc">{opt.desc}</div>
                      </label>
                    </div>
                  ))}
                </div>
              </div>

              {/* Quality Selection (MP4 only) */}
              {format === 'mp4' && (
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
                          {qualityDetails && (
                            <div className="export-quality-details">
                              <span>{qualityDetails.resolution}</span>
                              <span>•</span>
                              <span>{qualityDetails.bitrate}</span>
                            </div>
                          )}
                        </label>
                      </div>
                    ))}
                  </div>
                </div>
              )}

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
