import type { ShortVideo } from '../../api/types'
import { useDeleteShortVideo, useShortVideo } from '../../api/hooks'
import { useWebSocketEvent, useWebSocketEvents } from '../../api/websocket-hooks'
import { WebSocketEventType } from '../../types/websocket'
import { Loader2, Trash2, Download, Wifi, WifiOff } from 'lucide-react'
import { useMemo, useState, useEffect } from 'react'
import type { WebSocketMessage } from '../../types/websocket'

interface ShortVideoListProps {
  videos: ShortVideo[]
}

function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins}分钟前`
  if (diffHours < 24) return `${diffHours}小时前`
  if (diffDays < 30) return `${diffDays}天前`
  
  return date.toLocaleDateString('zh-CN')
}

// 单个视频卡片组件，支持轮询和 WebSocket 实时更新
function ShortVideoCard({ video: initialVideo }: { video: ShortVideo }) {
  const deleteMutation = useDeleteShortVideo()
  const [wsProgress, setWsProgress] = useState<number | null>(null)
  const { isConnected: wsIsConnected } = useWebSocketEvents()
  
  // 对于生成中的视频，使用 hook 来轮询状态
  // 如果 WebSocket 未连接，更激进地轮询（每 2 秒），否则减少轮询频率
  const pollingQuery = useShortVideo(
    initialVideo.status === 'generating' ? initialVideo.id : undefined,
    {
      refetchInterval: wsIsConnected ? 5000 : 2000, // 断线时更频繁的轮询
    }
  )
  
  const video = pollingQuery.data || initialVideo

  // 监听 WebSocket 生成进度事件
  useWebSocketEvent(
    WebSocketEventType.GENERATION_PROGRESS,
    (message: WebSocketMessage) => {
      const data = message.data as Record<string, unknown>
      if (data.videoID === video.id && typeof data.progress === 'number') {
        setWsProgress(data.progress)
      }
    }
  )

  // 监听 WebSocket 生成失败事件
  useWebSocketEvent(
    WebSocketEventType.GENERATION_FAILED,
    (message: WebSocketMessage) => {
      const data = message.data as Record<string, unknown>
      if (data.videoID === video.id) {
        // 生成失败，清空进度，让轮询查询获取最新状态
        setWsProgress(null)
      }
    }
  )

  // 重置 ws 状态当视频不再生成时
  useEffect(() => {
    if (video.status !== 'generating') {
      setWsProgress(null)
    }
  }, [video.status])

  // 计算估计的生成进度（基于创建时间）
  const estimatedProgress = useMemo(() => {
    if (video.status !== 'generating') return 0
    
    const createdAt = new Date(video.createdAt).getTime()
    const now = Date.now()
    const elapsed = now - createdAt
    
    // 假设生成通常在 60 秒内完成
    const estimatedTotal = 60000
    const progress = Math.min(90, (elapsed / estimatedTotal) * 100)
    
    return Math.round(progress)
  }, [video.status, video.createdAt])

  // WebSocket 进度优先于估计进度
  const displayProgress = wsProgress ?? estimatedProgress

  const handleDelete = async (videoId: string) => {
    if (confirm('确定要删除这个视频吗？')) {
      try {
        await deleteMutation.mutateAsync(videoId)
      } catch (error) {
        console.error('Failed to delete video:', error)
      }
    }
  }

  const getStatusBadge = (status: ShortVideo['status']) => {
    const statusConfig: Record<string, { label: string; color: string }> = {
      pending: { label: '待处理', color: 'bg-gray-100 text-gray-800' },
      generating: { label: '生成中', color: 'bg-blue-100 text-blue-800' },
      completed: { label: '已完成', color: 'bg-green-100 text-green-800' },
      failed: { label: '失败', color: 'bg-red-100 text-red-800' },
    }
    const config = statusConfig[status] || { label: status, color: 'bg-gray-100 text-gray-800' }
    return (
      <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${config.color}`}>
        {config.label}
      </span>
    )
  }

  return (
    <div key={video.id} className="rounded-lg border border-gray-200 p-6">
      <div className="flex items-start justify-between">
        <div className="space-y-3 flex-1">
          <div className="flex items-center gap-3">
            <h3 className="font-semibold text-gray-900">
              视频 #{video.id.slice(0, 8)}
            </h3>
            {getStatusBadge(video.status)}
          </div>

          <div className="text-sm text-gray-600 space-y-1">
            <p>
              <span className="font-medium">创建时间：</span>
              {formatTimeAgo(video.createdAt)}
            </p>
            {video.heyGenVideoId && (
              <p>
                <span className="font-medium">HeyGen 视频 ID：</span>
                <code className="text-xs bg-gray-100 px-2 py-0.5 rounded ml-1">{video.heyGenVideoId}</code>
              </p>
            )}
            {video.status === 'generating' && (
              <div className="pt-2">
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-blue-500 rounded-full transition-all duration-300"
                      style={{ width: `${displayProgress}%` }}
                    />
                  </div>
                  <span className="text-xs text-blue-600 font-medium whitespace-nowrap">
                    {displayProgress}%
                  </span>
                </div>
              </div>
            )}
            {video.result && (
              <>
                <p>
                  <span className="font-medium">时长：</span>
                  {video.result.duration}秒
                </p>
                <p>
                  <span className="font-medium">文件大小：</span>
                  {(video.result.sizeBytes / 1024 / 1024).toFixed(2)} MB
                </p>
              </>
            )}
            {video.errorMessage && (
              <p className="text-red-600">
                <span className="font-medium">错误：</span>
                {video.errorMessage}
              </p>
            )}
          </div>

          {video.result && (
            <div className="pt-2">
              <img
                src={video.result.thumbURL}
                alt="视频缩略图"
                className="h-16 w-28 rounded object-cover"
              />
            </div>
          )}
        </div>

        <div className="flex flex-col gap-2">
          {video.status === 'completed' && video.result && (
            <button
              onClick={() => {
                const a = document.createElement('a')
                a.href = video.result!.heyGenVideoURL
                a.download = `video-${video.id}.mp4`
                a.click()
              }}
              className="inline-flex items-center justify-center rounded border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              <Download size={16} className="mr-2" />
              下载
            </button>
          )}
          <button
            onClick={() => handleDelete(video.id)}
            disabled={deleteMutation.isPending}
            className="inline-flex items-center justify-center rounded border border-red-300 bg-white px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
          >
            {deleteMutation.isPending ? (
              <Loader2 size={16} className="mr-2 animate-spin" />
            ) : (
              <Trash2 size={16} className="mr-2" />
            )}
            删除
          </button>
        </div>
      </div>
    </div>
  )
}

export default function ShortVideoList({ videos }: ShortVideoListProps) {
  const { isConnected, connectionError } = useWebSocketEvents()

  if (videos.length === 0) {
    return (
      <div className="rounded-lg border border-gray-200 p-12 text-center">
        <p className="text-gray-500">暂无视频，立即创建一个吧</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* WebSocket 连接状态指示器 */}
      {connectionError && (
        <div className="flex items-center gap-2 rounded-lg border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800">
          <WifiOff size={16} className="flex-shrink-0" />
          <span>
            实时更新已断开 - 正在尝试重新连接...
            {connectionError && <span className="ml-1">({connectionError})</span>}
          </span>
        </div>
      )}

      {isConnected && (
        <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          <Wifi size={16} className="flex-shrink-0" />
          <span>实时更新已连接</span>
        </div>
      )}

      {videos.map((video) => (
        <ShortVideoCard key={video.id} video={video} />
      ))}
    </div>
  )
}
