import { useEffect, useRef, useState, useCallback } from 'react'
import { useTimelineStore } from '../../lib/editor/timeline-store'
import type { Track, Clip } from '../../lib/editor/types'

const TRACK_HEIGHT = 60
const RULER_HEIGHT = 40
const PIXELS_PER_SECOND = 100
const MIN_CLIP_WIDTH = 4

export function TimelineCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const timeline = useTimelineStore((state) => state.timeline)
  const playheadTime = useTimelineStore((state) => state.playheadTime)
  const setPlayheadTime = useTimelineStore((state) => state.setPlayheadTime)
  const setCurrentClip = useTimelineStore((state) => state.setCurrentClip)
  const moveClip = useTimelineStore((state) => state.moveClip)

  const [canvasWidth, setCanvasWidth] = useState(0)
  const [canvasHeight, setCanvasHeight] = useState(0)
  const [draggedClipId, setDraggedClipId] = useState<string | null>(null)
  const [dragOffset, setDragOffset] = useState(0)

  // Update canvas size
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const handleResize = () => {
      const rect = container.getBoundingClientRect()
      setCanvasWidth(rect.width)
      setCanvasHeight(Math.max(400, rect.height))
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

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

  const drawClip = useCallback((ctx: CanvasRenderingContext2D, clip: Clip, yOffset: number, height: number) => {
    const x = (clip.startTime / 1000) * PIXELS_PER_SECOND
    const clipWidth = Math.max(MIN_CLIP_WIDTH, (clip.duration / 1000) * PIXELS_PER_SECOND)
    const padding = 4

    ctx.fillStyle = 'rgba(124, 58, 237, 0.4)'
    ctx.fillRect(x + padding, yOffset + padding, clipWidth - padding * 2, height - padding * 2)

    ctx.strokeStyle = '#7c3aed'
    ctx.lineWidth = 1.5
    ctx.strokeRect(x + padding, yOffset + padding, clipWidth - padding * 2, height - padding * 2)

    if (clipWidth > 40) {
      ctx.fillStyle = '#e5e7eb'
      ctx.font = 'bold 12px Inter, sans-serif'
      ctx.textAlign = 'left'
      ctx.textBaseline = 'middle'
      ctx.fillText(`Clip`, x + padding + 6, yOffset + height / 2)
    }
  }, [])

  const drawTrack = useCallback(
    (ctx: CanvasRenderingContext2D, track: Track, yOffset: number, width: number, height: number) => {
      ctx.fillStyle = track.visible ? 'rgb(19, 22, 33)' : 'rgba(19, 22, 33, 0.5)'
      ctx.fillRect(0, yOffset, width, height)

      ctx.strokeStyle = 'rgba(148, 163, 184, 0.16)'
      ctx.lineWidth = 1
      ctx.strokeRect(0, yOffset, width, height)

      for (const clip of track.clips) {
        drawClip(ctx, clip, yOffset, height)
      }
    },
    [drawClip],
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
      drawTrack(ctx, track, yOffset, canvasWidth, TRACK_HEIGHT)
      yOffset += TRACK_HEIGHT
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
      const trackIndex = Math.floor((y - RULER_HEIGHT) / TRACK_HEIGHT)
      const track = timeline.tracks[trackIndex]
      if (!track) return

      const clickTime = (x / PIXELS_PER_SECOND) * 1000
      const clip = track.clips.find((c) => c.startTime <= clickTime && clickTime <= c.startTime + c.duration)

      if (clip && !track.locked) {
        setCurrentClip(clip)
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
      } else {
        setCurrentClip(null)
      }
    }
  }

  return (
    <div
      ref={containerRef}
      className="timeline-canvas-container"
      style={{
        flex: 1,
        minHeight: '300px',
        position: 'relative',
        overflow: 'hidden',
        backgroundColor: 'rgb(13, 15, 23)',
        borderRadius: '8px',
        border: '1px solid rgba(148, 163, 184, 0.16)',
      }}
    >
      <canvas
        ref={canvasRef}
        onMouseDown={handleCanvasMouseDown}
        style={{
          display: 'block',
          width: '100%',
          height: '100%',
          cursor: draggedClipId ? 'grabbing' : 'default',
        }}
      />
    </div>
  )
}
