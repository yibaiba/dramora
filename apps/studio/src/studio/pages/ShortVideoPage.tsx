import { useShortVideoTemplates, useCreateShortVideo, useShortVideos, useBatchSubmissions, useCreateBatchSubmission, useCancelBatchSubmission } from '../../api/hooks'
import type { ShortVideoTemplate, CreateShortVideoRequest, HeyGenAvatarId, CreateBatchSubmissionRequest } from '../../api/types'
import { HEYGEN_AVATARS } from '../../api/types'
import { useCallback, useState } from 'react'
import TemplateSelector from '../components/TemplateSelector'
import ShortVideoForm from '../components/ShortVideoForm'
import ShortVideoList from '../components/ShortVideoList'
import BatchQueueTable from '../components/BatchQueueTable'
import CreateBatchSubmissionDialog from '../components/CreateBatchSubmissionDialog'
import { Loader2 } from 'lucide-react'

export default function ShortVideoPage() {
  const templatesQuery = useShortVideoTemplates()
  const videosQuery = useShortVideos()
  const batchesQuery = useBatchSubmissions()
  const createMutation = useCreateShortVideo()
  const createBatchMutation = useCreateBatchSubmission()
  const cancelBatchMutation = useCancelBatchSubmission()

  const [selectedTemplate, setSelectedTemplate] = useState<ShortVideoTemplate | null>(null)
  const [parameters, setParameters] = useState<Record<string, unknown>>({})
  const [selectedAvatarId, setSelectedAvatarId] = useState<HeyGenAvatarId>('avatar_001')
  const [activeTab, setActiveTab] = useState<'create' | 'videos' | 'queue'>('create')
  const [isCreateBatchDialogOpen, setIsCreateBatchDialogOpen] = useState(false)

  const handleSelectTemplate = useCallback((template: ShortVideoTemplate) => {
    setSelectedTemplate(template)
    setParameters(template.config || {})
  }, [])

  const handleParametersChange = useCallback((newParams: Record<string, unknown>) => {
    setParameters(newParams)
  }, [])

  const handleCreateVideo = useCallback(async () => {
    if (!selectedTemplate) {
      return
    }

    const request: CreateShortVideoRequest = {
      templateId: selectedTemplate.id,
      parameters,
      heyGenAvatarId: selectedAvatarId,
    }

    try {
      await createMutation.mutateAsync(request)
      setSelectedTemplate(null)
      setParameters({})
      setSelectedAvatarId('avatar_001')
      setActiveTab('videos')
    } catch (error) {
      console.error('Failed to create short video:', error)
    }
  }, [selectedTemplate, parameters, selectedAvatarId, createMutation])

  const handleCreateBatch = useCallback(async (req: CreateBatchSubmissionRequest) => {
    try {
      await createBatchMutation.mutateAsync(req)
      setIsCreateBatchDialogOpen(false)
    } catch (error) {
      console.error('Failed to create batch submission:', error)
    }
  }, [createBatchMutation])

  const handleCancelBatch = useCallback(async (batchId: string) => {
    try {
      await cancelBatchMutation.mutateAsync(batchId)
    } catch (error) {
      console.error('Failed to cancel batch:', error)
    }
  }, [cancelBatchMutation])

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">电商短视频</h1>
          <p className="mt-2 text-gray-600">
            使用 AI 生成高质量短视频，支持多种模板和自定义参数
          </p>
        </div>
      </div>

      <div className="space-y-4">
        <div className="flex gap-2 border-b border-gray-200">
          <button
            onClick={() => setActiveTab('create')}
            className={`px-4 py-2 font-medium text-sm border-b-2 ${
              activeTab === 'create'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            创建视频
          </button>
          <button
            onClick={() => setActiveTab('videos')}
            className={`px-4 py-2 font-medium text-sm border-b-2 ${
              activeTab === 'videos'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            我的视频
          </button>
          <button
            onClick={() => setActiveTab('queue')}
            className={`px-4 py-2 font-medium text-sm border-b-2 ${
              activeTab === 'queue'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            队列管理
          </button>
        </div>

        {activeTab === 'create' && (
          <div className="space-y-6">
            {templatesQuery.isLoading ? (
              <div className="rounded-lg border border-gray-200 p-12 flex items-center justify-center">
                <Loader2 className="animate-spin text-gray-400" size={40} />
              </div>
            ) : templatesQuery.error ? (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                <p className="text-sm text-red-700">
                  加载模板失败：{templatesQuery.error instanceof Error ? templatesQuery.error.message : '未知错误'}
                </p>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                <div>
                  <h2 className="mb-4 text-lg font-semibold text-gray-900">选择模板</h2>
                  <TemplateSelector
                    templates={templatesQuery.data || []}
                    selectedTemplate={selectedTemplate}
                    onSelectTemplate={handleSelectTemplate}
                  />
                </div>

                <div className="lg:col-span-2">
                  {selectedTemplate ? (
                    <div className="space-y-4">
                      <div className="rounded-lg border border-gray-200 p-6">
                        <h3 className="mb-4 text-lg font-semibold text-gray-900">
                          {selectedTemplate.name}
                        </h3>
                        <p className="mb-6 text-sm text-gray-600">
                          {selectedTemplate.description}
                        </p>

                        <ShortVideoForm
                          config={selectedTemplate.config}
                          parameters={parameters}
                          onChange={handleParametersChange}
                        />

                        <div className="mt-6 space-y-4 border-t border-gray-200 pt-6">
                          <div>
                            <h4 className="text-sm font-medium text-gray-900 mb-3">选择虚拟主播</h4>
                            <div className="grid grid-cols-3 gap-3">
                              {Object.values(HEYGEN_AVATARS).map((avatar) => (
                                <button
                                  key={avatar.id}
                                  onClick={() => setSelectedAvatarId(avatar.id)}
                                  className={`p-3 rounded-lg border-2 transition-all text-left ${
                                    selectedAvatarId === avatar.id
                                      ? 'border-blue-500 bg-blue-50'
                                      : 'border-gray-200 bg-white hover:border-gray-300'
                                  }`}
                                >
                                  <div className="font-medium text-sm text-gray-900">{avatar.name}</div>
                                  <div className="text-xs text-gray-500 mt-1">{avatar.description}</div>
                                </button>
                              ))}
                            </div>
                          </div>
                        </div>

                        <div className="mt-6 flex justify-end gap-3">
                          <button
                            onClick={() => {
                              setSelectedTemplate(null)
                              setParameters({})
                            }}
                            className="inline-flex items-center justify-center rounded border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                          >
                            取消
                          </button>
                          <button
                            onClick={handleCreateVideo}
                            disabled={createMutation.isPending}
                            className="inline-flex items-center justify-center rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                          >
                            {createMutation.isPending && (
                              <Loader2 className="mr-2 animate-spin" size={16} />
                            )}
                            {createMutation.isPending ? '生成中...' : '生成视频'}
                          </button>
                        </div>
                      </div>

                      {createMutation.error && (
                        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                          <p className="text-sm text-red-700">
                            生成失败：{createMutation.error instanceof Error ? createMutation.error.message : '未知错误'}
                          </p>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="rounded-lg border border-gray-200 p-12 flex items-center justify-center text-center">
                      <div>
                        <p className="text-gray-500">选择左侧的模板开始创建视频</p>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'videos' && (
          <div>
            {videosQuery.isLoading ? (
              <div className="rounded-lg border border-gray-200 p-12 flex items-center justify-center">
                <Loader2 className="animate-spin text-gray-400" size={40} />
              </div>
            ) : videosQuery.error ? (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                <p className="text-sm text-red-700">
                  加载视频失败：{videosQuery.error instanceof Error ? videosQuery.error.message : '未知错误'}
                </p>
              </div>
            ) : (
              <ShortVideoList videos={videosQuery.data?.videos || []} />
            )}
          </div>
        )}

        {activeTab === 'queue' && (
          <div className="space-y-4">
            <div className="flex justify-end">
              <button
                onClick={() => setIsCreateBatchDialogOpen(true)}
                className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
              >
                新建批量任务
              </button>
            </div>

            {batchesQuery.isLoading ? (
              <div className="rounded-lg border border-gray-200 p-12 flex items-center justify-center">
                <Loader2 className="animate-spin text-gray-400" size={40} />
              </div>
            ) : batchesQuery.error ? (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                <p className="text-sm text-red-700">
                  加载队列失败：{batchesQuery.error instanceof Error ? batchesQuery.error.message : '未知错误'}
                </p>
              </div>
            ) : (
              <BatchQueueTable
                batches={batchesQuery.data?.data || []}
                onCancel={handleCancelBatch}
              />
            )}

            <CreateBatchSubmissionDialog
              isOpen={isCreateBatchDialogOpen}
              onClose={() => setIsCreateBatchDialogOpen(false)}
              onSubmit={handleCreateBatch}
              availableVideoIds={(videosQuery.data?.videos || []).map(v => v.id)}
            />
          </div>
        )}
      </div>
    </div>
  )
}
