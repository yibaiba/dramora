import type { Timeline, Asset } from '../api/types'

const FRAMERATE = 30
const MS_PER_FRAME = 1000 / FRAMERATE

/**
 * 将毫秒转换为帧数（基于 30fps）
 */
function msToFrames(ms: number): number {
  return Math.round(ms / MS_PER_FRAME)
}

/**
 * 生成 FCPXML 格式字符串
 * @param timeline 时间线数据
 * @param assets 资源映射
 * @returns FCPXML 格式的 XML 字符串
 */
export function generateFCPXML(timeline: Timeline, assets: Map<string, Asset> = new Map()): string {
  const totalDurationFrames = msToFrames(timeline.duration_ms)
  const videoTracks = timeline.tracks.filter((track) => track.kind === 'video')

  // 构建 clips XML
  let clipsXml = ''

  videoTracks.forEach((track) => {
    track.clips.forEach((clip) => {
      const asset = assets.get(clip.asset_id)
      const durationFrames = msToFrames(clip.duration_ms)
      const startFrames = msToFrames(clip.start_ms)

      // 获取媒体路径（如果有资源）
      const mediaPath = asset?.uri ?? `asset://${clip.asset_id}`

      clipsXml += `
    <clip name="${escapeXml(clip.asset_id)}" duration="${durationFrames}s" start="${startFrames}s">
      <media>
        <video>
          <media-rep path="${escapeXml(mediaPath)}"/>
        </video>
      </media>
    </clip>`
    })
  })

  // 生成完整的 FCPXML 文档
  const fcpxml = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
  <resources>
    <format id="r1" name="FFmpeg Image2" framerate="30"/>
  </resources>
  <library>
    <event name="Dramora Timeline">
      <project name="Export">
        <sequence format="r1" duration="${totalDurationFrames}s">
          <spine>${clipsXml}
          </spine>
        </sequence>
      </project>
    </event>
  </library>
</fcpxml>`

  return fcpxml
}

/**
 * 转义 XML 特殊字符
 */
function escapeXml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}
