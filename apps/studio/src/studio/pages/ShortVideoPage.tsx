import { Clapperboard, Layers3, PlayCircle, Workflow } from 'lucide-react'
import { useMemo, useState } from 'react'
import {
  useBatchSubmissions,
  useCancelBatchSubmission,
  useCreateBatchSubmission,
  useCreateShortVideo,
  useShortVideoTemplates,
  useShortVideos,
} from '../../api/hooks'
import type {
  BatchSubmission,
  CreateBatchSubmissionRequest,
  CreateShortVideoRequest,
  HeyGenAvatarId,
  ShortVideo,
  ShortVideoParameters,
  ShortVideoTemplate,
} from '../../api/types'
import { HEYGEN_AVATARS } from '../../api/types'
import BatchQueueTable from '../components/BatchQueueTable'
import CreateBatchSubmissionDialog from '../components/CreateBatchSubmissionDialog'
import ShortVideoForm from '../components/ShortVideoForm'
import ShortVideoList from '../components/ShortVideoList'
import { StatePlaceholder } from '../components/StatePlaceholder'
import TemplateSelector from '../components/TemplateSelector'
import '../styles/short-video.css'

type ShortVideoTab = 'create' | 'videos' | 'queue'

const EMPTY_SHORT_VIDEOS: ShortVideo[] = []
const EMPTY_BATCHES: BatchSubmission[] = []

export default function ShortVideoPage() {
  const templatesQuery = useShortVideoTemplates()
  const videosQuery = useShortVideos()
  const batchesQuery = useBatchSubmissions()
  const createVideoMutation = useCreateShortVideo()
  const createBatchMutation = useCreateBatchSubmission()
  const cancelBatchMutation = useCancelBatchSubmission()

  const [selectedTemplate, setSelectedTemplate] = useState<ShortVideoTemplate | null>(null)
  const [parameters, setParameters] = useState<ShortVideoParameters>({})
  const [selectedAvatarId, setSelectedAvatarId] = useState<HeyGenAvatarId>('avatar_001')
  const [activeTab, setActiveTab] = useState<ShortVideoTab>('create')
  const [isCreateBatchDialogOpen, setIsCreateBatchDialogOpen] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const videos = videosQuery.data?.videos ?? EMPTY_SHORT_VIDEOS
  const batches = batchesQuery.data?.data ?? EMPTY_BATCHES

  const stats = useMemo(
    () => ({
      templates: templatesQuery.data?.length ?? 0,
      videos: videos.length,
      generating: videos.filter((video) => video.status === 'generating').length,
      batches: batches.length,
    }),
    [batches.length, templatesQuery.data?.length, videos],
  )

  const handleSelectTemplate = (template: ShortVideoTemplate) => {
    setSelectedTemplate(template)
    setParameters(template.config.fields?.length ? {} : {})
  }

  const handleCreateVideo = async () => {
    if (!selectedTemplate) {
      setErrorMessage('请先选择一个模板。')
      return
    }

    setErrorMessage(null)
    const request: CreateShortVideoRequest = {
      templateId: selectedTemplate.id,
      parameters,
      heyGenAvatarId: selectedAvatarId,
    }

    try {
      await createVideoMutation.mutateAsync(request)
      setSelectedTemplate(null)
      setParameters({})
      setSelectedAvatarId('avatar_001')
      setActiveTab('videos')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '创建短视频失败')
    }
  }

  const handleCreateBatch = async (request: CreateBatchSubmissionRequest) => {
    setErrorMessage(null)
    await createBatchMutation.mutateAsync(request)
    setIsCreateBatchDialogOpen(false)
    setActiveTab('queue')
  }

  const handleCancelBatch = async (batchId: string) => {
    try {
      setErrorMessage(null)
      await cancelBatchMutation.mutateAsync(batchId)
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '取消批量任务失败')
    }
  }

  return (
    <section className="short-video-page" aria-labelledby="short-video-title">
      <header className="short-video-header">
        <div>
          <span className="section-kicker">电商视频工场</span>
          <h1 id="short-video-title">电商短视频</h1>
          <p className="page-subtitle">模板选型、虚拟主播、视频结果与批量队列统一收敛在一个导演台工作面板。</p>
        </div>
      </header>

      <div className="short-video-stats">
        <article className="short-video-stat-card">
          <span className="section-kicker">模板数量</span>
          <strong>{stats.templates}</strong>
          <p>可用于商品主推、促销转化、场景种草的模板数。</p>
        </article>
        <article className="short-video-stat-card">
          <span className="section-kicker">视频结果</span>
          <strong>{stats.videos}</strong>
          <p>当前组织下的短视频记录总量。</p>
        </article>
        <article className="short-video-stat-card">
          <span className="section-kicker">生成中</span>
          <strong>{stats.generating}</strong>
          <p>正在通过 HeyGen / 队列生成的视频数。</p>
        </article>
        <article className="short-video-stat-card">
          <span className="section-kicker">批量任务</span>
          <strong>{stats.batches}</strong>
          <p>已提交的批量生成任务数量。</p>
        </article>
      </div>

      <div className="short-video-tabs" role="tablist" aria-label="电商短视频视图">
        <button type="button" className={activeTab === 'create' ? 'is-active' : ''} onClick={() => setActiveTab('create')}>
          <Clapperboard size={16} aria-hidden="true" /> 创建
        </button>
        <button type="button" className={activeTab === 'videos' ? 'is-active' : ''} onClick={() => setActiveTab('videos')}>
          <PlayCircle size={16} aria-hidden="true" /> 视频结果
        </button>
        <button type="button" className={activeTab === 'queue' ? 'is-active' : ''} onClick={() => setActiveTab('queue')}>
          <Workflow size={16} aria-hidden="true" /> 批量队列
        </button>
      </div>

      {errorMessage ? <div className="form-error">{errorMessage}</div> : null}

      {activeTab === 'create' ? (
        <div className="short-video-layout">
          <article className="short-video-panel">
            <header className="short-video-panel-header">
              <Layers3 size={18} aria-hidden="true" />
              <div>
                <h2>模板库</h2>
                <p>挑选适合当前商品投放目标的视频模板。</p>
              </div>
            </header>
            {templatesQuery.isLoading ? (
              <StatePlaceholder tone="loading" title="正在加载模板..." />
            ) : templatesQuery.error ? (
              <StatePlaceholder
                tone="error"
                title="模板加载失败"
                description={templatesQuery.error instanceof Error ? templatesQuery.error.message : '未知错误'}
              />
            ) : (
              <TemplateSelector
                templates={templatesQuery.data ?? []}
                selectedTemplate={selectedTemplate}
                onSelectTemplate={handleSelectTemplate}
              />
            )}
          </article>

          <article className="short-video-panel is-wide">
            <header className="short-video-panel-header">
              <Clapperboard size={18} aria-hidden="true" />
              <div>
                <h2>生成配置</h2>
                <p>围绕模板参数与虚拟主播完成一次完整的视频创建。</p>
              </div>
            </header>

            {selectedTemplate ? (
              <div className="short-video-composer">
                <div className="short-video-template-hero">
                  <span className="short-video-pill">{selectedTemplate.category}</span>
                  <strong>{selectedTemplate.name}</strong>
                  <p>{selectedTemplate.description}</p>
                </div>

                <ShortVideoForm config={selectedTemplate.config} parameters={parameters} onChange={setParameters} />

                <section className="short-video-avatar-section">
                  <header>
                    <strong>选择虚拟主播</strong>
                    <p>主播风格会影响口播语气与电商镜头气质。</p>
                  </header>
                  <div className="short-video-avatar-grid">
                    {Object.values(HEYGEN_AVATARS).map((avatar) => (
                      <button
                        key={avatar.id}
                        type="button"
                        className={`short-video-avatar-card${selectedAvatarId === avatar.id ? ' is-selected' : ''}`}
                        onClick={() => setSelectedAvatarId(avatar.id)}
                      >
                        <strong>{avatar.name}</strong>
                        <p>{avatar.description}</p>
                      </button>
                    ))}
                  </div>
                </section>

                <div className="short-video-actions">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={() => {
                      setSelectedTemplate(null)
                      setParameters({})
                    }}
                  >
                    清空
                  </button>
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={() => void handleCreateVideo()}
                    disabled={createVideoMutation.isPending}
                  >
                    {createVideoMutation.isPending ? '生成中...' : '生成视频'}
                  </button>
                </div>
              </div>
            ) : (
              <StatePlaceholder
                tone="empty"
                title="先从左侧选择模板"
                description="选中模板后，这里会显示参数配置、主播选择和生成操作。"
              />
            )}
          </article>
        </div>
      ) : null}

      {activeTab === 'videos' ? <ShortVideoList videos={videos} /> : null}

      {activeTab === 'queue' ? (
        <article className="short-video-panel">
          <header className="short-video-panel-header">
            <Workflow size={18} aria-hidden="true" />
            <div>
              <h2>批量队列</h2>
              <p>对已有短视频记录批量排队、跟踪执行与取消异常任务。</p>
            </div>
            <button type="button" className="btn btn-primary" onClick={() => setIsCreateBatchDialogOpen(true)}>
              新建批量任务
            </button>
          </header>

          <BatchQueueTable
            batches={batches}
            isLoading={batchesQuery.isLoading}
            onCancel={(batchId) => void handleCancelBatch(batchId)}
          />
        </article>
      ) : null}

      <CreateBatchSubmissionDialog
        isOpen={isCreateBatchDialogOpen}
        onClose={() => setIsCreateBatchDialogOpen(false)}
        onSubmit={handleCreateBatch}
        availableVideoIds={videos.map((video) => video.id)}
      />
    </section>
  )
}
