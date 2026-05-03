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
export function generateExportFilename(title: string, format: 'mp4' | 'fcpxml' | 'prproj' | 'drp' = 'mp4'): string {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').split('T')[0]
  const cleanTitle = title.replace(/[^a-zA-Z0-9_-]/g, '_').substring(0, 30)
  const extMap = {
    mp4: 'mp4',
    fcpxml: 'fcpxml',
    prproj: 'prproj',
    drp: 'drp',
  }
  const ext = extMap[format] || 'mp4'
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

/**
 * Convert milliseconds to frames at given fps
 */
function msToFrames(ms: number, fps: number = 30): number {
  return Math.round((ms / 1000) * fps)
}

/**
 * Escape XML special characters
 */
function escapeXml(str: string): string {
  return str.replace(/[<>&'"]/g, (char) => {
    const entities: Record<string, string> = {
      '<': '&lt;',
      '>': '&gt;',
      '&': '&amp;',
      "'": '&apos;',
      '"': '&quot;',
    }
    return entities[char] || char
  })
}

/**
 * Generate FCPXML format XML string
 */
function generateFCPXMLContent(timeline: Timeline, fps: number = 30): string {
  const videoTracks = timeline.tracks.filter((t) => t.type === 'video')
  if (videoTracks.length === 0) {
    return ''
  }

  const durationFrames = msToFrames(timeline.duration, fps)

  let clipXML = ''
  for (const track of videoTracks) {
    for (const clip of track.clips) {
      const offset = msToFrames(clip.startTime, fps)
      const duration = msToFrames(clip.duration, fps)
      const filename = clip.sourceUrl.split('/').pop() || 'clip.mp4'

      clipXML += `    <clip offset="${offset}s" name="${escapeXml(filename)}" duration="${duration}s">
      <media>
        <video>
          <file src="${escapeXml(clip.sourceUrl)}" />
        </video>
      </media>
      <timeMap>
        <timept value="0s" />
        <timept value="${duration}s" />
      </timeMap>
    </clip>\n`
    }
  }

  return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
  <resources>
    <format id="r1" name="FFmpeg Image2" framerate="${fps}"/>
  </resources>
  <library>
    <event name="Dramora Timeline">
      <project name="Export">
        <sequence format="r1" duration="${durationFrames}s">
          <spine>
${clipXML}          </spine>
        </sequence>
      </project>
    </event>
  </library>
</fcpxml>`
}

/**
 * Export timeline to FCPXML format (generates local XML)
 */
export function exportToFCPXML(timeline: Timeline, videoTitle: string = 'timeline'): void {
  const fcpxmlContent = generateFCPXMLContent(timeline)
  if (!fcpxmlContent) {
    throw new Error('没有视频轨道可以导出')
  }

  const blob = new Blob([fcpxmlContent], { type: 'application/xml' })
  const filename = generateExportFilename(videoTitle, 'fcpxml')
  downloadBlob(blob, filename)
}

/**
 * Generate Premiere Pro XML format
 */
function generatePremiereXML(timeline: Timeline): string {
  const videoTracks = timeline.tracks.filter((t) => t.type === 'video')

  let clipXML = ''
  for (const track of videoTracks) {
    for (const clip of track.clips) {
      const inTime = Math.round(clip.startTime / 1000 * 30) // 30fps
      const duration = Math.round(clip.duration / 1000 * 30)
      const filename = clip.sourceUrl.split('/').pop() || 'clip.mp4'

      clipXML += `  <clip intime="${inTime}" duration="${duration}">
    <name>${escapeXml(filename)}</name>
    <media>
      <video>
        <file path="${escapeXml(clip.sourceUrl)}" />
      </video>
    </media>
  </clip>\n`
    }
  }

  return `<?xml version="1.0" encoding="UTF-8"?>
<PremiereData version="3">
  <project>
    <name>Dramora Export</name>
    <sequence>
      <name>Timeline</name>
      <videoTracks>
${clipXML}      </videoTracks>
    </sequence>
  </project>
</PremiereData>`
}

/**
 * Export timeline to Premiere Pro format
 */
export function exportToPremiere(timeline: Timeline, videoTitle: string = 'timeline'): void {
  const premiereXML = generatePremiereXML(timeline)
  const blob = new Blob([premiereXML], { type: 'application/xml' })
  const filename = generateExportFilename(videoTitle, 'prproj')
  downloadBlob(blob, filename)
}

/**
 * Generate DaVinci Resolve XML format
 */
function generateDaVinciXML(timeline: Timeline): string {
  const videoTracks = timeline.tracks.filter((t) => t.type === 'video')

  let clipXML = ''
  let currentTime = 0
  for (const track of videoTracks) {
    for (const clip of track.clips) {
      const durationSec = clip.duration / 1000
      clipXML += `  <clip name="${escapeXml(clip.sourceUrl.split('/').pop() || 'clip.mp4')}" start="${currentTime.toFixed(2)}" duration="${durationSec.toFixed(2)}">
    <media>
      <video>
        <path>${escapeXml(clip.sourceUrl)}</path>
      </video>
    </media>
  </clip>\n`
      currentTime += durationSec
    }
  }

  return `<?xml version="1.0" encoding="UTF-8"?>
<DavinciResolveProject>
  <timeline>
    <name>Dramora Export</name>
    <framerate>30</framerate>
    <duration>${(timeline.duration / 1000).toFixed(2)}</duration>
    <clips>
${clipXML}    </clips>
  </timeline>
</DavinciResolveProject>`
}

/**
 * Export timeline to DaVinci Resolve format
 */
export function exportToDaVinci(timeline: Timeline, videoTitle: string = 'timeline'): void {
  const davinciXML = generateDaVinciXML(timeline)
  const blob = new Blob([davinciXML], { type: 'application/xml' })
  const filename = generateExportFilename(videoTitle, 'drp')
  downloadBlob(blob, filename)
}
