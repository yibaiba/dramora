import type { ShortVideo } from '../../api/types'
import { useDeleteShortVideo } from '../../api/hooks'
import { Loader2, Trash2, Download } from 'lucide-react'

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

export default function ShortVideoList({ videos }: ShortVideoListProps) {
  const deleteMutation = useDeleteShortVideo()

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

  if (videos.length === 0) {
    return (
      <div className="rounded-lg border border-gray-200 p-12 text-center">
        <p className="text-gray-500">暂无视频，立即创建一个吧</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {videos.map((video) => (
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
                    <code className="text-xs bg-gray-100 px-2 py-0.5 rounded">{video.heyGenVideoId}</code>
                  </p>
                )}
                {video.status === 'generating' && (
                  <div className="pt-2">
                    <div className="flex items-center gap-2">
                      <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden">
                        <div className="h-full w-1/2 bg-blue-500 rounded-full animate-pulse"></div>
                      </div>
                      <span className="text-xs text-blue-600 font-medium">生成中...</span>
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
      ))}
    </div>
  )
}
