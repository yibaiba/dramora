import type { Timeline } from './types'

/**
 * Format time from milliseconds to MM:SS.mmm format
 */
export function formatTimeRange(startMs: number, endMs: number): string {
  const formatTime = (ms: number): string => {
    const totalSeconds = ms / 1000
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = Math.floor(totalSeconds % 60)
    const ms_remainder = Math.floor((totalSeconds % 1) * 1000)
    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}.${String(ms_remainder).padStart(3, '0')}`
  }

  return `${formatTime(startMs)} - ${formatTime(endMs)}`
}

/**
 * Format milliseconds to MM:SS format
 */
export function formatDuration(ms: number): string {
  const totalSeconds = ms / 1000
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = Math.floor(totalSeconds % 60)
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

/**
 * Calculate estimated output file size in MB
 * Rough estimation based on video bitrate and duration
 */
export function calculateExportSize(durationMs: number, quality: 'low' | 'medium' | 'high' | 'very-high'): number {
  // Bitrate in kbps based on quality
  const bitrateMap: Record<string, number> = {
    low: 1000, // 1 Mbps
    medium: 2000, // 2 Mbps
    high: 4000, // 4 Mbps
    'very-high': 6000, // 6 Mbps
  }

  const bitrateKbps = bitrateMap[quality]
  const durationSeconds = durationMs / 1000
  const totalKilobytes = (bitrateKbps * durationSeconds) / 8
  return totalKilobytes / 1024 // Convert to MB
}

/**
 * Download a blob as a file
 */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * Generate a filename for export
 */
export function generateExportFilename(title: string, format: 'mp4' | 'fcpxml' = 'mp4'): string {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0]
  const cleanTitle = title.replace(/[^a-zA-Z0-9_-]/g, '_').substring(0, 30)
  const ext = format === 'fcpxml' ? 'fcpxml' : 'mp4'
  return `${cleanTitle}_${timestamp}.${ext}`
}

/**
 * Get quality level label
 */
export function getQualityLabel(quality: 'low' | 'medium' | 'high' | 'very-high'): string {
  const labels: Record<string, string> = {
    low: '低 (480p)',
    medium: '中 (720p)',
    high: '高 (1080p)',
    'very-high': '超高 (1080p+)',
  }
  return labels[quality]
}

/**
 * Get quality details
 */
export function getQualityDetails(quality: 'low' | 'medium' | 'high' | 'very-high'): {
  resolution: string
  bitrate: string
  fileSize: string
} {
  const details: Record<
    string,
    {
      resolution: string
      bitrate: string
      fileSize: string
    }
  > = {
    low: {
      resolution: '480p',
      bitrate: '1 Mbps',
      fileSize: '约 450MB/分钟',
    },
    medium: {
      resolution: '720p',
      bitrate: '2 Mbps',
      fileSize: '约 900MB/分钟',
    },
    high: {
      resolution: '1080p',
      bitrate: '4 Mbps',
      fileSize: '约 1.8GB/分钟',
    },
    'very-high': {
      resolution: '1080p+',
      bitrate: '6 Mbps',
      fileSize: '约 2.7GB/分钟',
    },
  }
  return details[quality]
}

/**
 * Estimate export duration in seconds
 * Rough estimation: base time + complexity based on clip count and speed variations
 */
export function estimateExportDuration(timeline: Timeline, quality: 'low' | 'medium' | 'high' | 'very-high'): number {
  const videoDurationSeconds = timeline.duration / 1000

  // Base estimation: 1/3 real-time for fast preset
  let estimatedSeconds = videoDurationSeconds / 3

  // Adjust based on quality
  const qualityMultiplier: Record<string, number> = {
    low: 0.8,
    medium: 1.0,
    high: 1.3,
    'very-high': 1.6,
  }
  estimatedSeconds *= qualityMultiplier[quality]

  // Count speed variations (slower processing)
  let hasSpeedVariations = false
  for (const track of timeline.tracks) {
    for (const clip of track.clips) {
      if (Math.abs(clip.properties.speed - 1) > 0.01) {
        hasSpeedVariations = true
        break
      }
    }
    if (hasSpeedVariations) break
  }

  if (hasSpeedVariations) {
    estimatedSeconds *= 1.5
  }

  return Math.max(10, Math.ceil(estimatedSeconds)) // Minimum 10 seconds
}

/**
 * Validate that clip can be exported
 */
export function validateClipsForExport(timeline: Timeline): string | null {
  if (timeline.tracks.length === 0) {
    return '没有轨道可以导出'
  }

  const hasClips = timeline.tracks.some((track) => track.clips.length > 0)
  if (!hasClips) {
    return '没有片段可以导出'
  }

  // Check for unreasonable clip properties
  for (const track of timeline.tracks) {
    for (const clip of track.clips) {
      if (clip.properties.speed <= 0 || clip.properties.speed > 10) {
        return `片段速度必须在 0.25x 到 4x 之间`
      }
      if (clip.properties.opacity < 0 || clip.properties.opacity > 1) {
        return `片段不透明度必须在 0 到 1 之间`
      }
    }
  }

  return null
}
