import { useEffect, useRef, useState, useCallback } from 'react'
import { Eye, EyeOff, Lock, LockOpen } from 'lucide-react'
import { useTimelineStore } from '../../lib/editor/timeline-store'
import type { Track, Clip } from '../../lib/editor/types'

const RULER_HEIGHT = 40
const PIXELS_PER_SECOND = 100
const MIN_CLIP_WIDTH = 4
const TRACK_HEIGHTS: Record<string, number> = {
  video: 120,
  audio: 80,
  subtitle: 60,
}
const TRACK_HEADER_WIDTH = 180

export function TimelineCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const timeline = useTimelineStore((state) => state.timeline)
  const playheadTime = useTimelineStore((state) => state.playheadTime)
  const setPlayheadTime = useTimelineStore((state) => state.setPlayheadTime)
  const setCurrentClip = useTimelineStore((state) => state.setCurrentClip)
  const moveClip = useTimelineStore((state) => state.moveClip)
  const toggleTrackVisibility = useTimelineStore((state) => state.toggleTrackVisibility)
  const toggleTrackLock = useTimelineStore((state) => state.toggleTrackLock)
  const trimClip = useTimelineStore((state) => state.trimClip)

  const [canvasWidth, setCanvasWidth] = useState(0)
  const [canvasHeight, setCanvasHeight] = useState(0)
  const [draggedClipId, setDraggedClipId] = useState<string | null>(null)
  const [dragOffset, setDragOffset] = useState(0)
  const [selectedClipId, setSelectedClipId] = useState<string | null>(null)
  const [hoveredClipId, setHoveredClipId] = useState<string | null>(null)
  const [trimMode, setTrimMode] = useState<'start' | 'end' | null>(null)
  const [cursorStyle, setCursorStyle] = useState('default')

  // Update canvas size
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const handleResize = () => {
      const rect = container.getBoundingClientRect()
      setCanvasWidth(rect.width - TRACK_HEADER_WIDTH)
      const totalHeight = RULER_HEIGHT + timeline.tracks.reduce((sum, track) => sum + TRACK_HEIGHTS[track.type], 0)
      setCanvasHeight(Math.max(400, totalHeight))
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [timeline.tracks])

  const drawRuler = useCallback((ctx: CanvasRenderingContext2D, width: number, height: number) => {
    ctx.fillStyle = 'rgb(19, 22, 33)'
    ctx.fillRect(0, 0, width, height)

    ctx.strokeStyle = 'rgba(148, 163, 184, 0.16)'
    ctx.lineWidth = 1
    ctx.strokeRect(0, 0, width, height)

    ctx.fillStyle = '#94a3b8'
    ctx.font = '12px Inter, sans-serif'
    ctx.textAlign = 'left'
    ctx.textBaseline = 'top'

    const secondWidth = PIXELS_PER_SECOND
    const majorTick = 5

    for (let i = 0; i < Math.ceil(timeline.duration / 1000) + 1; i++) {
      const x = i * secondWidth

      if (i % majorTick === 0) {
        ctx.fillRect(x, height - 20, 2, 16)
        ctx.fillText(`${i}s`, x + 6, height - 18)
      } else {
        ctx.fillRect(x, height - 12, 1, 8)
      }
    }
  }, [timeline.duration])

  const drawClip = useCallback((ctx: CanvasRenderingContext2D, clip: Clip, yOffset: number, height: number, isSelected: boolean, isHovered: boolean) => {
    const x = (clip.startTime / 1000) * PIXELS_PER_SECOND
    const clipWidth = Math.max(MIN_CLIP_WIDTH, (clip.duration / 1000) * PIXELS_PER_SECOND)
    const padding = 4

    // Clip background color based on selection/hover state
    if (isSelected) {
      ctx.fillStyle = 'rgba(148, 163, 184, 0.7)'
    } else if (isHovered) {
      ctx.fillStyle = 'rgba(124, 58, 237, 0.55)'
    } else {
      ctx.fillStyle = 'rgba(124, 58, 237, 0.4)'
    }
    ctx.fillRect(x + padding, yOffset + padding, clipWidth - padding * 2, height - padding * 2)

    // Clip border
    ctx.strokeStyle = isSelected ? '#94a3b8' : isHovered ? '#a78bfa' : '#7c3aed'
    ctx.lineWidth = isSelected ? 2 : isHovered ? 2 : 1.5
    ctx.strokeRect(x + padding, yOffset + padding, clipWidth - padding * 2, height - padding * 2)

    // Clip label with duration
    if (clipWidth > 40) {
      ctx.fillStyle = '#e5e7eb'
      ctx.font = 'bold 12px Inter, sans-serif'
      ctx.textAlign = 'left'
      ctx.textBaseline = 'middle'

      // Format duration as MM:SS
      const durationSeconds = clip.duration / 1000
      const minutes = Math.floor(durationSeconds / 60)
      const seconds = Math.floor(durationSeconds % 60)
      const durationStr = `${minutes}:${seconds.toString().padStart(2, '0')}`

      const labelText = clipWidth > 80 ? `Clip ${durationStr}` : 'Clip'
      ctx.fillText(labelText, x + padding + 6, yOffset + height / 2)
    }

    // Draw trim handles when selected
    if (isSelected) {
      const handleWidth = 6
      ctx.fillStyle = '#38bdf8'

      // Start trim handle
      ctx.fillRect(x, yOffset + padding, handleWidth, height - padding * 2)

      // End trim handle
      ctx.fillRect(x + clipWidth - handleWidth, yOffset + padding, handleWidth, height - padding * 2)
    }
  }, [])

  const drawTrack = useCallback(
    (ctx: CanvasRenderingContext2D, track: Track, yOffset: number, width: number, height: number) => {
      // Track background - varies by type
      const trackTypeColor = {
        video: 'rgb(30, 35, 50)',
        audio: 'rgb(25, 30, 45)',
        subtitle: 'rgb(20, 25, 40)',
      }
      ctx.fillStyle = track.visible ? trackTypeColor[track.type] : 'rgba(19, 22, 33, 0.5)'
      ctx.fillRect(0, yOffset, width, height)

      // Track border
      ctx.strokeStyle = 'rgba(148, 163, 184, 0.16)'
      ctx.lineWidth = 1
      ctx.strokeRect(0, yOffset, width, height)

      // Draw clips
      for (const clip of track.clips) {
        const isSelected = clip.id === selectedClipId
        const isHovered = clip.id === hoveredClipId
        drawClip(ctx, clip, yOffset, height, isSelected, isHovered)
      }
    },
    [drawClip, selectedClipId, hoveredClipId],
  )

  const drawPlayhead = useCallback((ctx: CanvasRenderingContext2D, time: number, pixelsPerSecond: number, height: number) => {
    const x = (time / 1000) * pixelsPerSecond

    ctx.strokeStyle = '#38bdf8'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, height)
    ctx.stroke()

    ctx.fillStyle = '#38bdf8'
    ctx.fillRect(x - 4, 0, 8, 10)
  }, [])

  // Draw canvas
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || canvasWidth === 0) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    canvas.width = canvasWidth
    canvas.height = canvasHeight

    ctx.fillStyle = 'rgb(13, 15, 23)'
    ctx.fillRect(0, 0, canvasWidth, canvasHeight)

    drawRuler(ctx, canvasWidth, RULER_HEIGHT)

    let yOffset = RULER_HEIGHT
    for (const track of timeline.tracks) {
      const height = TRACK_HEIGHTS[track.type]
      drawTrack(ctx, track, yOffset, canvasWidth, height)
      yOffset += height
    }

    drawPlayhead(ctx, playheadTime, PIXELS_PER_SECOND, canvasHeight)
  }, [canvasWidth, canvasHeight, timeline, playheadTime, drawRuler, drawTrack, drawPlayhead])

  const handleCanvasMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current
    if (!canvas) return

    const rect = canvas.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    const playheadX = (playheadTime / 1000) * PIXELS_PER_SECOND
    if (Math.abs(x - playheadX) < 10 && y < RULER_HEIGHT) {
      const handleMouseMove = (moveE: MouseEvent) => {
        const moveX = moveE.clientX - rect.left
        const newTime = Math.max(0, (moveX / PIXELS_PER_SECOND) * 1000)
        setPlayheadTime(newTime)
      }

      const handleMouseUp = () => {
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
      }

      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
      return
    }

    if (y > RULER_HEIGHT) {
      // Find which track was clicked
      let trackIndex = 0
      let yOffset = RULER_HEIGHT
      for (let i = 0; i < timeline.tracks.length; i++) {
        const track = timeline.tracks[i]
        const trackHeight = TRACK_HEIGHTS[track.type]
        if (y >= yOffset && y < yOffset + trackHeight) {
          trackIndex = i
          break
        }
        yOffset += trackHeight
      }

      const track = timeline.tracks[trackIndex]
      if (!track) return

      const clickTime = (x / PIXELS_PER_SECOND) * 1000
      const clip = track.clips.find((c) => c.startTime <= clickTime && clickTime <= c.startTime + c.duration)

      if (clip && !track.locked) {
        // Check if clicking on clip edges for trimming
        const clipStartX = (clip.startTime / 1000) * PIXELS_PER_SECOND
        const clipEndX = ((clip.startTime + clip.duration) / 1000) * PIXELS_PER_SECOND
        const TRIM_HANDLE_WIDTH = 6

        const isStartEdge = Math.abs(x - clipStartX) < TRIM_HANDLE_WIDTH
        const isEndEdge = Math.abs(x - clipEndX) < TRIM_HANDLE_WIDTH

        if (isStartEdge || isEndEdge) {
          // Trim mode
          const mode = isStartEdge ? 'start' : 'end'
          setTrimMode(mode)
          setCurrentClip(clip)
          setSelectedClipId(clip.id)

          const handleMouseMove = (moveE: MouseEvent) => {
            const moveX = moveE.clientX - rect.left
            const newTime = Math.max(0, (moveX / PIXELS_PER_SECOND) * 1000)

            if (mode === 'start') {
              const maxNewStart = clip.startTime + clip.duration - 100 // Prevent zero-length clips
              const adjustedStart = Math.min(newTime, maxNewStart)
              const newDuration = clip.startTime + clip.duration - adjustedStart
              trimClip(clip.id, adjustedStart, newDuration)
            } else {
              const minNewEnd = clip.startTime + 100 // Prevent zero-length clips
              const adjustedEnd = Math.max(newTime, minNewEnd)
              const newDuration = adjustedEnd - clip.startTime
              trimClip(clip.id, clip.startTime, newDuration)
            }
          }

          const handleMouseUp = () => {
            setTrimMode(null)
            document.removeEventListener('mousemove', handleMouseMove)
            document.removeEventListener('mouseup', handleMouseUp)
          }

          document.addEventListener('mousemove', handleMouseMove)
          document.addEventListener('mouseup', handleMouseUp)
        } else {
          // Normal drag mode (move clip)
          setCurrentClip(clip)
          setSelectedClipId(clip.id)
          setDraggedClipId(clip.id)
          setDragOffset(clickTime - clip.startTime)

          const handleMouseMove = (moveE: MouseEvent) => {
            const moveX = moveE.clientX - rect.left
            const newTime = (moveX / PIXELS_PER_SECOND) * 1000
            const newStartTime = newTime - dragOffset

            if (newStartTime !== clip.startTime) {
              moveClip(clip.id, newStartTime)
            }
          }

          const handleMouseUp = () => {
            setDraggedClipId(null)
            document.removeEventListener('mousemove', handleMouseMove)
            document.removeEventListener('mouseup', handleMouseUp)
          }

          document.addEventListener('mousemove', handleMouseMove)
          document.addEventListener('mouseup', handleMouseUp)
        }
      } else {
        setCurrentClip(null)
        setSelectedClipId(null)
      }
    }
  }

  const handleCanvasMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current
    if (!canvas || draggedClipId) {
      setCursorStyle('default')
      setHoveredClipId(null)
      return
    }

    const rect = canvas.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    let hoveredId: string | null = null

    // Check if hovering over any clip
    if (y > RULER_HEIGHT) {
      let yOffset = RULER_HEIGHT
      for (let i = 0; i < timeline.tracks.length; i++) {
        const track = timeline.tracks[i]
        const trackHeight = TRACK_HEIGHTS[track.type]
        if (y >= yOffset && y < yOffset + trackHeight) {
          for (const clip of track.clips) {
            const clipStartX = (clip.startTime / 1000) * PIXELS_PER_SECOND
            const clipEndX = ((clip.startTime + clip.duration) / 1000) * PIXELS_PER_SECOND
            if (x >= clipStartX && x <= clipEndX) {
              hoveredId = clip.id
              break
            }
          }
          break
        }
        yOffset += trackHeight
      }
    }

    setHoveredClipId(hoveredId)

    if (y > RULER_HEIGHT && selectedClipId) {
      // Find the selected clip
      let foundClip: Clip | null = null
      for (const track of timeline.tracks) {
        const clip = track.clips.find((c) => c.id === selectedClipId)
        if (clip) {
          foundClip = clip
          break
        }
      }

      if (foundClip) {
        const clipStartX = (foundClip.startTime / 1000) * PIXELS_PER_SECOND
        const clipEndX = ((foundClip.startTime + foundClip.duration) / 1000) * PIXELS_PER_SECOND
        const TRIM_HANDLE_WIDTH = 6

        if (Math.abs(x - clipStartX) < TRIM_HANDLE_WIDTH || Math.abs(x - clipEndX) < TRIM_HANDLE_WIDTH) {
          setCursorStyle('ew-resize')
          return
        }
      }
    }

    setCursorStyle('default')
  }

  return (
    <div
      ref={containerRef}
      className="timeline-canvas-wrapper"
      style={{
        flex: 1,
        minHeight: '300px',
        position: 'relative',
        overflow: 'auto',
        backgroundColor: 'rgb(13, 15, 23)',
        borderRadius: '8px',
        border: '1px solid rgba(148, 163, 184, 0.16)',
        display: 'flex',
      }}
    >
      {/* Track Headers */}
      <div
        className="timeline-track-headers"
        style={{
          width: `${TRACK_HEADER_WIDTH}px`,
          flexShrink: 0,
          borderRight: '1px solid rgba(148, 163, 184, 0.16)',
          display: 'flex',
          flexDirection: 'column',
          backgroundColor: 'rgb(15, 18, 28)',
        }}
      >
        {/* Ruler spacer */}
        <div
          style={{
            height: `${RULER_HEIGHT}px`,
            borderBottom: '1px solid rgba(148, 163, 184, 0.16)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '12px',
            color: '#94a3b8',
            fontWeight: '500',
          }}
        >
          轨道
        </div>

        {/* Track headers */}
        {timeline.tracks.map((track) => (
          <div
            key={track.id}
            className="timeline-track-header"
            style={{
              height: `${TRACK_HEIGHTS[track.type]}px`,
              padding: '8px',
              borderBottom: '1px solid rgba(148, 163, 184, 0.16)',
              display: 'flex',
              flexDirection: 'column',
              gap: '4px',
              justifyContent: 'space-between',
            }}
          >
            <div style={{ fontSize: '12px', fontWeight: '600', color: '#e2e8f0' }}>
              {track.type === 'video' ? '视频' : track.type === 'audio' ? '音频' : '字幕'}
            </div>
            <div style={{ display: 'flex', gap: '4px', justifyContent: 'flex-start' }}>
              <button
                onClick={() => toggleTrackVisibility(track.id)}
                title={track.visible ? '隐藏' : '显示'}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: '2px',
                  display: 'flex',
                  alignItems: 'center',
                  color: track.visible ? '#94a3b8' : '#64748b',
                }}
              >
                {track.visible ? <Eye size={14} /> : <EyeOff size={14} />}
              </button>
              <button
                onClick={() => toggleTrackLock(track.id)}
                title={track.locked ? '解锁' : '锁定'}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: '2px',
                  display: 'flex',
                  alignItems: 'center',
                  color: track.locked ? '#f87171' : '#94a3b8',
                }}
              >
                {track.locked ? <Lock size={14} /> : <LockOpen size={14} />}
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Canvas */}
      <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
        <canvas
          ref={canvasRef}
          onMouseDown={handleCanvasMouseDown}
          onMouseMove={handleCanvasMouseMove}
          style={{
            display: 'block',
            width: '100%',
            height: '100%',
            cursor: draggedClipId ? 'grabbing' : trimMode ? 'ew-resize' : cursorStyle,
          }}
        />
      </div>
    </div>
  )
}
