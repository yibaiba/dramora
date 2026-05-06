import { Download, Loader2, Trash2, Wifi, WifiOff } from 'lucide-react'
import { useState } from 'react'
import { useDeleteShortVideo, useShortVideo } from '../../api/hooks'
import { useWebSocketEvent, useWebSocketEvents } from '../../api/websocket-hooks'
import type { ShortVideo } from '../../api/types'
import { WebSocketEventType } from '../../types/websocket'
import type { WebSocketMessage } from '../../types/websocket'
import { StatePlaceholder } from './StatePlaceholder'

interface ShortVideoListProps {
  videos: ShortVideo[]
}

function formatDateTime(dateString: string): string {
  return new Date(dateString).toLocaleString('zh-CN')
}

function statusLabel(status: ShortVideo['status']) {
  switch (status) {
    case 'pending':
      return '待处理'
    case 'generating':
      return '生成中'
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
  }
}

function ShortVideoCard({ video: initialVideo }: { video: ShortVideo }) {
  const deleteMutation = useDeleteShortVideo()
  const [wsProgress, setWsProgress] = useState<number | null>(null)
  const { isConnected: wsIsConnected } = useWebSocketEvents()

  const pollingQuery = useShortVideo(initialVideo.status === 'generating' ? initialVideo.id : undefined, {
    refetchInterval: wsIsConnected ? 5000 : 2000,
  })
  const video = pollingQuery.data ?? initialVideo

  useWebSocketEvent(WebSocketEventType.GENERATION_PROGRESS, (message: WebSocketMessage) => {
    const data = message.data as Record<string, unknown>
    if (data.videoID === video.id && typeof data.progress === 'number') {
      setWsProgress(data.progress)
    }
  })

  useWebSocketEvent(WebSocketEventType.GENERATION_FAILED, (message: WebSocketMessage) => {
    const data = message.data as Record<string, unknown>
    if (data.videoID === video.id) {
      setWsProgress(null)
    }
  })

  const liveProgress = video.status === 'generating' ? wsProgress : null

  const handleDelete = async () => {
    if (!window.confirm('确定要删除这个视频吗？')) return
    await deleteMutation.mutateAsync(video.id)
  }

  return (
    <article className="short-video-result-card">
      <div className="short-video-result-main">
        <div className="short-video-result-header">
          <div>
            <strong>视频 #{video.id.slice(0, 8)}</strong>
            <div className="short-video-result-meta">
              <span className={`short-video-status-badge status-${video.status}`}>{statusLabel(video.status)}</span>
              <span>创建于 {formatDateTime(video.createdAt)}</span>
              {video.heyGenVideoId ? <span>任务 ID {video.heyGenVideoId}</span> : null}
            </div>
          </div>
          {video.result?.thumbURL ? <img src={video.result.thumbURL} alt="短视频缩略图" className="short-video-thumb" /> : null}
        </div>

        {video.status === 'generating' ? (
          <div className="short-video-progress-block">
            <div className="short-video-progress-row">
              <div className="short-video-progress-track">
                <div className="short-video-progress-fill" style={{ width: `${liveProgress ?? 28}%` }} />
              </div>
              <span>{liveProgress !== null ? `${liveProgress}%` : '处理中'}</span>
            </div>
            <small>WebSocket 优先；断线时会自动轮询最新状态。</small>
          </div>
        ) : null}

        {video.result ? (
          <div className="short-video-result-meta">
            <span>时长 {video.result.duration} 秒</span>
            <span>大小 {(video.result.sizeBytes / 1024 / 1024).toFixed(2)} MB</span>
          </div>
        ) : null}

        {video.errorMessage ? <p className="form-error">生成失败：{video.errorMessage}</p> : null}
      </div>

      <div className="short-video-result-actions">
        {video.status === 'completed' && video.result ? (
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => window.open(video.result?.heyGenVideoURL, '_blank', 'noopener,noreferrer')}
          >
            <Download size={16} aria-hidden="true" /> 下载
          </button>
        ) : null}
        <button type="button" className="btn btn-danger" onClick={() => void handleDelete()} disabled={deleteMutation.isPending}>
          {deleteMutation.isPending ? <Loader2 size={16} className="animate-spin" /> : <Trash2 size={16} aria-hidden="true" />}
          删除
        </button>
      </div>
    </article>
  )
}

export default function ShortVideoList({ videos }: ShortVideoListProps) {
  const { connectionError } = useWebSocketEvents()

  if (videos.length === 0) {
    return (
      <StatePlaceholder
        tone="empty"
        title="还没有生成结果"
        description="选择模板并配置主播后，新的电商短视频会出现在这里。"
      />
    )
  }

  return (
    <div className="short-video-results">
      {connectionError ? (
        <div className="short-video-live-banner is-warning" role="status">
          <WifiOff size={16} aria-hidden="true" />
          <span>实时更新已断开，当前改用轮询刷新。{connectionError ? ` (${connectionError})` : ''}</span>
        </div>
      ) : (
        <div className="short-video-live-banner" role="status">
          <Wifi size={16} aria-hidden="true" />
          <span>实时更新已连接，生成进度会自动同步。</span>
        </div>
      )}

      {videos.map((video) => (
        <ShortVideoCard key={video.id} video={video} />
      ))}
    </div>
  )
}
