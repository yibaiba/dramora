import {
  Activity,
  BookOpenText,
  Boxes,
  ChevronDown,
  Clapperboard,
  Download,
  Eye,
  Film,
  Home,
  Layers3,
  Library,
  ListFilter,
  Lock,
  Maximize2,
  Music2,
  Play,
  Plus,
  Radio,
  Scissors,
  Search,
  Settings,
  Sparkles,
  Subtitles,
  UserRound,
  Zap,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import {
  useApproveApprovalGate,
  useCreateEpisode,
  useCreateProject,
  useEpisodeApprovalGates,
  useEpisodeAssets,
  useEpisodes,
  useEpisodeTimeline,
  useExport,
  useGenerateShotPromptPack,
  useGenerationJobs,
  useProjects,
  useSaveShotPromptPack,
  useSaveEpisodeTimeline,
  useSeedApprovalGates,
  useSeedEpisodeAssets,
  useSeedStoryboardShots,
  useSeedStoryMap,
  useShotPromptPack,
  useStartEpisodeExport,
  useStartShotVideoGeneration,
  useStartStoryAnalysis,
  useStoryAnalyses,
  useStoryboardShots,
  useStoryMap,
  useUpdateStoryboardShot,
} from './api/hooks'
import { useStudioStore } from './state/studioStore'
import type {
  ApprovalGate,
  Asset,
  Episode,
  GenerationJob,
  Project,
  SaveTimelineRequest,
  ShotPromptPack,
  StoryMap,
  StoryboardShot,
} from './api/types'

type StudioShot = {
  code: string
  description: string
  durationMS: number
  id?: string
  key: string
  progress: number
  prompt: string
  sceneCode: string
  sceneName: string
  status: 'prompt_ready' | 'generating' | 'queued' | 'approved' | 'draft'
  tags: string[]
  thumbnail: string
  title: string
}

type InspectorTab = 'details' | 'prompt' | 'references' | 'notes'
type ViewMode = 'grid' | 'compact'
type ShotDraft = {
  description: string
  durationMS: number
  title: string
}

const navItems = [
  { icon: Home, label: '工作台' },
  { icon: BookOpenText, label: '故事中枢' },
  { icon: UserRound, label: '角色资产' },
  { icon: Layers3, label: '场景图谱' },
  { icon: Boxes, label: '分镜台' },
  { icon: Activity, label: '生成队列' },
  { icon: Film, label: '剪辑时间线' },
  { icon: Download, label: '导出中心' },
]

const demoShots: StudioShot[] = [
  {
    code: '01',
    description: '开场建立镜，交代天门与主角位置。',
    durationMS: 4200,
    key: 'demo-01',
    progress: 100,
    prompt: '航拍视角，云海翻涌，远处天门洞开，金色朝阳洒落，少年独立山崖，衣袂飘扬，史诗感，国漫风格，细节丰富，光影戏剧化。',
    sceneCode: 'SC01',
    sceneName: '云海初现',
    status: 'prompt_ready',
    tags: ['云澜', '白璃', '长老'],
    thumbnail: 'thumb-cloud',
    title: '天门洞开',
  },
  {
    code: '02',
    description: '主角中近景，承接立誓情绪。',
    durationMS: 3600,
    key: 'demo-02',
    progress: 72,
    prompt: '中近景，少年云澜回头凝视远方，风吹乱发，眼神坚定，背景云雾和宗门剪影，电影级景深，稳定角色一致性。',
    sceneCode: 'SC01',
    sceneName: '少年立志',
    status: 'generating',
    tags: ['云澜'],
    thumbnail: 'thumb-hero',
    title: '少年立誓',
  },
  {
    code: '03',
    description: '女主登场镜，强调角色辨识度。',
    durationMS: 4800,
    key: 'demo-03',
    progress: 0,
    prompt: '白璃从桃花雨中出现，银白长发，轻甲飘带，镜头缓慢推进，柔光，高级国漫角色设计，保持面部一致性。',
    sceneCode: 'SC01',
    sceneName: '白璃登场',
    status: 'queued',
    tags: ['白璃'],
    thumbnail: 'thumb-muse',
    title: '白璃回眸',
  },
  {
    code: '04',
    description: '宗门环境大全景，用于世界观铺垫。',
    durationMS: 5500,
    key: 'demo-04',
    progress: 100,
    prompt: '宗门全景，云端建筑群漂浮在山巅，仙鹤穿过云层，大远景，恢弘构图，金紫色晚霞，电影级环境细节。',
    sceneCode: 'SC02',
    sceneName: '宗门全景',
    status: 'prompt_ready',
    tags: ['云澜', '长老', '弟子'],
    thumbnail: 'thumb-temple',
    title: '云端宗门',
  },
  {
    code: '05',
    description: '试炼动作镜，突出剑光和节奏。',
    durationMS: 3900,
    key: 'demo-05',
    progress: 45,
    prompt: '试炼场上剑光爆发，少年拔剑迎敌，蓝紫能量划破画面，快速推轨镜头，动势强烈，角色动作清晰。',
    sceneCode: 'SC02',
    sceneName: '试炼开始',
    status: 'generating',
    tags: ['云澜', '对手'],
    thumbnail: 'thumb-battle',
    title: '剑光破云',
  },
  {
    code: '06',
    description: '威胁登场镜，制造压迫感和悬念。',
    durationMS: 5100,
    key: 'demo-06',
    progress: 0,
    prompt: '黑色异兽从云雾中压境，巨大龙翼遮天，火光映照山门，低角度仰拍，压迫感，暗黑史诗氛围。',
    sceneCode: 'SC03',
    sceneName: '异兽现世',
    status: 'queued',
    tags: ['异兽'],
    thumbnail: 'thumb-beast',
    title: '黑龙压境',
  },
]

const demoReferences = [
  { label: '@image1', thumbnail: 'thumb-cloud' },
  { label: '@image2', thumbnail: 'thumb-hero' },
  { label: '@image3', thumbnail: 'thumb-temple' },
  { label: '@image4', thumbnail: 'thumb-muse' },
]

function filterShots(shots: StudioShot[], query: string): StudioShot[] {
  const keyword = query.trim().toLowerCase()
  if (!keyword) return shots
  return shots.filter((shot) =>
    [
      shot.code,
      shot.description,
      shot.prompt,
      shot.sceneCode,
      shot.sceneName,
      shot.title,
      ...shot.tags,
    ].some((value) => value.toLowerCase().includes(keyword)),
  )
}

function createLocalShot(position: number): StudioShot {
  const code = position.toString().padStart(2, '0')
  return {
    code,
    description: '本地草稿镜头，可先改写提示词和导演备注。',
    durationMS: 4000,
    key: `local-${Date.now()}`,
    progress: 0,
    prompt: '请在右侧提示词面板补充人物、场景、镜头运动、光影和风格约束。',
    sceneCode: `SC${Math.max(1, Math.ceil(position / 3)).toString().padStart(2, '0')}`,
    sceneName: '待定镜头',
    status: 'draft',
    tags: ['草稿'],
    thumbnail: thumbnailByIndex(position),
    title: `草稿镜头 ${code}`,
  }
}

function App() {
  const { data: projects = [], isLoading: projectsLoading } = useProjects()
  const { selectedProjectId, selectProject } = useStudioStore()
  const selectedProject = useMemo(() => projects.find((project) => project.id === selectedProjectId) ?? projects[0], [projects, selectedProjectId])
  const { data: episodes = [] } = useEpisodes(selectedProject?.id)
  const [selectedEpisodeId, setSelectedEpisodeId] = useState<string>()
  const activeEpisode = useMemo(
    () => episodes.find((episode) => episode.id === selectedEpisodeId) ?? episodes[0],
    [episodes, selectedEpisodeId],
  )
  const { data: activeStoryMap } = useStoryMap(activeEpisode?.id)
  const { data: activeAnalyses = [] } = useStoryAnalyses(activeEpisode?.id)
  const { data: storyboardShots = [] } = useStoryboardShots(activeEpisode?.id)
  const canSeedStoryboard = activeAnalyses.length > 0 && hasStoryMapItems(activeStoryMap)
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>('details')
  const [localShots, setLocalShots] = useState<StudioShot[]>([])
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [onlyNeedsWork, setOnlyNeedsWork] = useState(false)
  const [promptDrafts, setPromptDrafts] = useState<Record<string, string>>({})
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedShotKey, setSelectedShotKey] = useState<string>()
  const [shotDrafts, setShotDrafts] = useState<Record<string, ShotDraft>>({})
  const [viewMode, setViewMode] = useState<ViewMode>('grid')
  const savePromptPack = useSaveShotPromptPack()
  const updateStoryboardShot = useUpdateStoryboardShot()
  const displayShots = useMemo(() => [...mapDisplayShots(storyboardShots), ...localShots], [localShots, storyboardShots])
  const filteredShots = useMemo(() => {
    const searchedShots = filterShots(displayShots, searchQuery)
    if (!onlyNeedsWork) return searchedShots
    return searchedShots.filter((shot) => ['draft', 'generating', 'queued'].includes(shot.status))
  }, [displayShots, onlyNeedsWork, searchQuery])
  const selectedShot = useMemo(
    () => displayShots.find((shot) => shot.key === selectedShotKey) ?? filteredShots[0] ?? displayShots[0] ?? demoShots[0],
    [displayShots, filteredShots, selectedShotKey],
  )
  const selectRelativeShot = (offset: number) => {
    const shotList = filteredShots.length > 0 ? filteredShots : displayShots
    if (shotList.length === 0) return
    const currentIndex = Math.max(0, shotList.findIndex((shot) => shot.key === selectedShot.key))
    const nextIndex = (currentIndex + offset + shotList.length) % shotList.length
    setSelectedShotKey(shotList[nextIndex].key)
  }
  const addLocalShot = () => {
    const shot = createLocalShot(displayShots.length + 1)
    setLocalShots((shots) => [...shots, shot])
    setSelectedShotKey(shot.key)
    setInspectorTab('prompt')
  }

  useEffect(() => {
    if (!selectedProjectId && selectedProject) selectProject(selectedProject.id)
  }, [selectProject, selectedProject, selectedProjectId])

  const updateLocalShot = (shotKey: string, values: Partial<StudioShot>) => {
    setLocalShots((shots) => shots.map((shot) => (shot.key === shotKey ? { ...shot, ...values } : shot)))
  }

  const saveSelectedShot = () => {
    const draft = shotDraftValue(selectedShot, shotDrafts[selectedShot.key])
    const prompt = promptDrafts[selectedShot.key] ?? selectedShot.prompt
    if (!selectedShot.id) {
      updateLocalShot(selectedShot.key, {
        description: draft.description,
        durationMS: draft.durationMS,
        prompt,
        title: draft.title,
        sceneName: draft.title,
      })
      return
    }
    updateStoryboardShot.mutate({
      request: {
        description: draft.description,
        duration_ms: draft.durationMS,
        prompt,
        title: draft.title,
      },
      shotId: selectedShot.id,
    })
  }

  const saveSelectedPrompt = () => {
    const directPrompt = promptDrafts[selectedShot.key] ?? selectedShot.prompt
    if (!selectedShot.id) {
      updateLocalShot(selectedShot.key, { prompt: directPrompt })
      return
    }
    savePromptPack.mutate({ request: { direct_prompt: directPrompt }, shotId: selectedShot.id })
  }

  return (
    <main className="cinema-shell">
      <StudioSidebar
        isLoading={projectsLoading}
        onSelectProject={selectProject}
        projects={projects}
        selectedProject={selectedProject}
      />
      <StudioStage
        activeEpisode={activeEpisode}
        canSeedStoryboard={canSeedStoryboard}
        displayShots={filteredShots}
        episodes={episodes}
        onAddLocalShot={addLocalShot}
        onSearchChange={setSearchQuery}
        onSelectEpisode={setSelectedEpisodeId}
        onSelectShot={setSelectedShotKey}
        onToggleNeedsWork={() => setOnlyNeedsWork((value) => !value)}
        onViewModeChange={setViewMode}
        onlyNeedsWork={onlyNeedsWork}
        project={selectedProject}
        searchQuery={searchQuery}
        selectedShotKey={selectedShot.key}
        viewMode={viewMode}
      />
      <ShotInspector
        activeEpisode={activeEpisode}
        displayShots={displayShots}
        inspectorTab={inspectorTab}
        note={notes[selectedShot.key] ?? ''}
        onInspectorTabChange={setInspectorTab}
        onNoteChange={(value) => setNotes((current) => ({ ...current, [selectedShot.key]: value }))}
        onSelectNext={() => selectRelativeShot(1)}
        onSelectPrevious={() => selectRelativeShot(-1)}
        onPromptDraftChange={(value) => setPromptDrafts((current) => ({ ...current, [selectedShot.key]: value }))}
        onSavePrompt={saveSelectedPrompt}
        onSaveShot={saveSelectedShot}
        onShotDraftChange={(draft) => setShotDrafts((current) => ({ ...current, [selectedShot.key]: draft }))}
        project={selectedProject}
        promptDraft={promptDrafts[selectedShot.key]}
        savingPrompt={savePromptPack.isPending}
        savingShot={updateStoryboardShot.isPending}
        selectedShot={selectedShot}
        shotDraft={shotDrafts[selectedShot.key]}
      />
    </main>
  )
}

function StudioSidebar({
  isLoading,
  onSelectProject,
  projects,
  selectedProject,
}: {
  isLoading: boolean
  onSelectProject: (projectId: string) => void
  projects: Project[]
  selectedProject?: Project
}) {
  const createProject = useCreateProject()
  const [projectName, setProjectName] = useState('')
  const submitProject = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    createProject.mutate(
      { description: 'AI 漫剧生产项目', name: projectName },
      { onSuccess: (project) => onSelectProject(project.id) },
    )
    setProjectName('')
  }

  return (
    <aside className="cinema-sidebar" aria-label="漫幕主导航">
      <div className="manmu-logo">
        <span className="logo-glyph">M</span>
        <div>
          <strong>漫幕 Manmu</strong>
          <small>AI 漫剧工场</small>
        </div>
      </div>
      <ProjectSwitcher isLoading={isLoading} onSelect={onSelectProject} projects={projects} selectedProject={selectedProject} />
      <nav className="cinema-nav">
        {navItems.map((item, index) => (
          <button className={index === 4 ? 'cinema-nav-item active' : 'cinema-nav-item'} key={item.label} type="button">
            <item.icon aria-hidden="true" />
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
      <form className="quick-create" onSubmit={submitProject}>
        <label>
          <span>新建项目</span>
          <input
            minLength={1}
            onChange={(event) => setProjectName(event.target.value)}
            placeholder="九霄之上"
            required
            value={projectName}
          />
        </label>
        <button disabled={createProject.isPending} type="submit">
          <Plus aria-hidden="true" />
          创建项目
        </button>
      </form>
      <div className="owner-card">
        <div className="avatar-orb">LY</div>
        <div>
          <strong>Lin Yifei</strong>
          <small>主理人</small>
        </div>
        <Settings aria-hidden="true" />
      </div>
    </aside>
  )
}

function ProjectSwitcher({
  isLoading,
  onSelect,
  projects,
  selectedProject,
}: {
  isLoading: boolean
  onSelect: (projectId: string) => void
  projects: Project[]
  selectedProject?: Project
}) {
  return (
    <section className="project-switcher" aria-busy={isLoading} aria-label="项目切换">
      <span className="section-kicker">当前项目</span>
      <button className="project-current" disabled={projects.length === 0} type="button">
        <span className="project-cover thumb-cloud" aria-hidden="true" />
        <span>
          <strong>{selectedProject?.name ?? '新建一个漫剧项目'}</strong>
          <small>{selectedProject?.description || '九霄之上 · 天门试炼'}</small>
        </span>
        <ChevronDown aria-hidden="true" />
      </button>
      {projects.length > 1 ? (
        <div className="project-mini-list">
          {projects.slice(0, 4).map((project) => (
            <button className={project.id === selectedProject?.id ? 'mini-project active' : 'mini-project'} key={project.id} onClick={() => onSelect(project.id)} type="button">
              {project.name}
            </button>
          ))}
        </div>
      ) : null}
    </section>
  )
}

function StudioStage({
  activeEpisode,
  canSeedStoryboard,
  displayShots,
  episodes,
  onAddLocalShot,
  onSearchChange,
  onSelectEpisode,
  onSelectShot,
  onToggleNeedsWork,
  onViewModeChange,
  onlyNeedsWork,
  project,
  searchQuery,
  selectedShotKey,
  viewMode,
}: {
  activeEpisode?: Episode
  canSeedStoryboard: boolean
  displayShots: StudioShot[]
  episodes: Episode[]
  onAddLocalShot: () => void
  onSearchChange: (query: string) => void
  onSelectEpisode: (episodeId: string) => void
  onSelectShot: (shotKey: string) => void
  onToggleNeedsWork: () => void
  onViewModeChange: (mode: ViewMode) => void
  onlyNeedsWork: boolean
  project?: Project
  searchQuery: string
  selectedShotKey: string
  viewMode: ViewMode
}) {
  return (
    <section className="studio-stage" aria-label="分镜与剪辑工作区">
      <StudioTopBar
        activeEpisode={activeEpisode}
        canSeedStoryboard={canSeedStoryboard}
        episodes={episodes}
        onSearchChange={onSearchChange}
        onSelectEpisode={onSelectEpisode}
        project={project}
        searchQuery={searchQuery}
      />
      <StoryboardWorkspace
        activeEpisode={activeEpisode}
        displayShots={displayShots}
        onAddLocalShot={onAddLocalShot}
        onSelectShot={onSelectShot}
        onToggleNeedsWork={onToggleNeedsWork}
        onViewModeChange={onViewModeChange}
        onlyNeedsWork={onlyNeedsWork}
        project={project}
        selectedShotKey={selectedShotKey}
        viewMode={viewMode}
      />
      <TimelineDock activeEpisode={activeEpisode} displayShots={displayShots} onAddLocalShot={onAddLocalShot} />
    </section>
  )
}

function StudioTopBar({
  activeEpisode,
  canSeedStoryboard,
  episodes,
  onSearchChange,
  onSelectEpisode,
  project,
  searchQuery,
}: {
  activeEpisode?: Episode
  canSeedStoryboard: boolean
  episodes: Episode[]
  onSearchChange: (query: string) => void
  onSelectEpisode: (episodeId: string) => void
  project?: Project
  searchQuery: string
}) {
  const createEpisode = useCreateEpisode(project?.id)
  const seedStoryboard = useSeedStoryboardShots()
  const [episodeTitle, setEpisodeTitle] = useState('')
  const submitEpisode = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    createEpisode.mutate({ title: episodeTitle }, { onSuccess: (episode) => {
      onSelectEpisode(episode.id)
      setEpisodeTitle('')
    } })
  }

  return (
    <header className="studio-topbar">
      <div className="title-cluster">
        <strong>{project?.name ?? '九霄之上 · 天门试炼'}</strong>
        <span className="pro-badge">Pro</span>
        <small>刚刚自动保存</small>
      </div>
      <label className="episode-pill">
        <Clapperboard aria-hidden="true" />
        <span className="sr-only">当前剧集</span>
        <select disabled={episodes.length === 0} onChange={(event) => onSelectEpisode(event.target.value)} value={activeEpisode?.id ?? ''}>
          {episodes.length === 0 ? <option value="">还没有剧集</option> : null}
          {episodes.map((episode) => (
            <option key={episode.id} value={episode.id}>
              第 {episode.number.toString().padStart(2, '0')} 集 · {episode.title}
            </option>
          ))}
        </select>
        <ChevronDown aria-hidden="true" />
      </label>
      <label className="search-box">
        <Search aria-hidden="true" />
        <span className="search-box-label">检索</span>
        <input onChange={(event) => onSearchChange(event.target.value)} placeholder="搜索镜头、场景、角色编号..." value={searchQuery} />
      </label>
      <div className="ops-status">
        <Radio aria-hidden="true" />
        <span>模型状态</span>
        <strong>全部可用</strong>
      </div>
      <div className="budget-card">
        <span>本集预算</span>
        <strong>¥ 12,450.00</strong>
        <small>68%</small>
      </div>
      <button className="primary-generate" disabled={!activeEpisode || !canSeedStoryboard || seedStoryboard.isPending} onClick={() => activeEpisode && seedStoryboard.mutate(activeEpisode.id)} type="button">
        <Sparkles aria-hidden="true" />
        生成下一镜
      </button>
      <form className="episode-create" onSubmit={submitEpisode}>
        <label>
          <span className="sr-only">剧集标题</span>
          <input disabled={!project} onChange={(event) => setEpisodeTitle(event.target.value)} placeholder="新建剧集标题" required value={episodeTitle} />
        </label>
        <button disabled={!project || createEpisode.isPending} type="submit">
          <Plus aria-hidden="true" />
        </button>
      </form>
      <span className="episode-count">{episodes.length} 集</span>
    </header>
  )
}

function StoryboardWorkspace({
  activeEpisode,
  displayShots,
  onAddLocalShot,
  onSelectShot,
  onToggleNeedsWork,
  onViewModeChange,
  onlyNeedsWork,
  project,
  selectedShotKey,
  viewMode,
}: {
  activeEpisode?: Episode
  displayShots: StudioShot[]
  onAddLocalShot: () => void
  onSelectShot: (shotKey: string) => void
  onToggleNeedsWork: () => void
  onViewModeChange: (mode: ViewMode) => void
  onlyNeedsWork: boolean
  project?: Project
  selectedShotKey: string
  viewMode: ViewMode
}) {
  const seedStoryMap = useSeedStoryMap()
  const seedAssets = useSeedEpisodeAssets()
  const seedStoryboard = useSeedStoryboardShots()
  const startStoryAnalysis = useStartStoryAnalysis()
  const { data: storyMap } = useStoryMap(activeEpisode?.id)
  const { data: analyses = [] } = useStoryAnalyses(activeEpisode?.id)
  const { data: assets = [] } = useEpisodeAssets(activeEpisode?.id)
  const { data: gates = [] } = useEpisodeApprovalGates(activeEpisode?.id)
  const { data: jobs = [] } = useGenerationJobs()
  const storyMapReady = hasStoryMapItems(storyMap)
  const hasAnalysis = analyses.length > 0
  const episodeJobs = useMemo(
    () => jobs.filter((job) => job.episode_id === activeEpisode?.id).slice(0, 5),
    [activeEpisode?.id, jobs],
  )

  return (
    <section className="storyboard-workspace" aria-labelledby="storyboard-board-title">
      <div className="board-header">
        <div>
          <h1 id="storyboard-board-title">分镜看板</h1>
          <span>{displayShots.length} 个镜头</span>
        </div>
        <div className="board-actions">
          <ActionButton disabled={!activeEpisode || startStoryAnalysis.isPending} icon={BookOpenText} label={`故事解析 ${analyses.length}`} onClick={() => activeEpisode && startStoryAnalysis.mutate(activeEpisode.id)} />
          <ActionButton disabled={!activeEpisode || !hasAnalysis || seedStoryMap.isPending} icon={Layers3} label={storyMapReady ? '资产图谱就绪' : '生成资产图谱'} onClick={() => activeEpisode && seedStoryMap.mutate(activeEpisode.id)} />
          <ActionButton disabled={!activeEpisode || !storyMapReady || seedAssets.isPending} icon={Library} label="生成候选资产" onClick={() => activeEpisode && seedAssets.mutate(activeEpisode.id)} />
          <ActionButton disabled={!activeEpisode || !hasAnalysis || !storyMapReady || seedStoryboard.isPending} icon={Boxes} label="生成分镜卡" onClick={() => activeEpisode && seedStoryboard.mutate(activeEpisode.id)} />
          <ActionButton icon={ListFilter} label={onlyNeedsWork ? `待处理 ${gates.length}` : `审批点 ${gates.length}`} onClick={onToggleNeedsWork} />
          <div className="view-toggle" aria-label="切换分镜视图">
            <button className={viewMode === 'grid' ? 'active' : ''} onClick={() => onViewModeChange('grid')} type="button">宫格</button>
            <button className={viewMode === 'compact' ? 'active' : ''} onClick={() => onViewModeChange('compact')} type="button">紧凑</button>
          </div>
        </div>
      </div>
      {!project || !activeEpisode ? (
        <div className="board-notice">
          正在展示演示分镜。创建项目和剧集后，故事解析、资产图谱、生成队列和导出动作会接入当前剧集。
        </div>
      ) : null}
      <ProductionFlowPanel
        analysesCount={analyses.length}
        assetsCount={assets.length}
        gatesCount={gates.length}
        jobs={episodeJobs}
        shotsCount={displayShots.filter((shot) => Boolean(shot.id)).length}
        storyMapReady={storyMapReady}
      />
      <ShotGrid
        displayShots={displayShots}
        onAddLocalShot={onAddLocalShot}
        onSelectShot={onSelectShot}
        selectedShotKey={selectedShotKey}
        viewMode={viewMode}
      />
    </section>
  )
}

function ActionButton({ disabled, icon: Icon, label, onClick }: { disabled?: boolean; icon: typeof Activity; label: string; onClick: () => void }) {
  return (
    <button className="ghost-action" disabled={disabled} onClick={onClick} type="button">
      <Icon aria-hidden="true" />
      {label}
    </button>
  )
}

function ProductionFlowPanel({
  analysesCount,
  assetsCount,
  gatesCount,
  jobs,
  shotsCount,
  storyMapReady,
}: {
  analysesCount: number
  assetsCount: number
  gatesCount: number
  jobs: GenerationJob[]
  shotsCount: number
  storyMapReady: boolean
}) {
  const steps = [
    { label: '故事解析', ready: analysesCount > 0, value: `${analysesCount} 份` },
    { label: '资产图谱', ready: storyMapReady, value: storyMapReady ? '已生成' : '待生成' },
    { label: '候选资产', ready: assetsCount > 0, value: `${assetsCount} 个` },
    { label: '分镜卡', ready: shotsCount > 0, value: `${shotsCount} 镜` },
    { label: '人审关卡', ready: gatesCount > 0, value: `${gatesCount} 个` },
  ]
  return (
    <section className="production-flow" aria-label="真实生产流程状态">
      <div className="flow-steps">
        {steps.map((step) => (
          <div className={step.ready ? 'flow-step ready' : 'flow-step'} key={step.label}>
            <strong>{step.label}</strong>
            <span>{step.value}</span>
          </div>
        ))}
      </div>
      <div className="job-rail" aria-label="当前剧集生成队列">
        <strong>生成队列</strong>
        {jobs.length === 0 ? (
          <span>暂无任务，先启动故事解析或镜头生成。</span>
        ) : (
          jobs.map((job) => (
            <span className={`job-pill ${job.status}`} key={job.id}>
              {job.task_type} · {jobStatusLabel(job.status)}
            </span>
          ))
        )}
      </div>
    </section>
  )
}

function ShotGrid({
  displayShots,
  onAddLocalShot,
  onSelectShot,
  selectedShotKey,
  viewMode,
}: {
  displayShots: StudioShot[]
  onAddLocalShot: () => void
  onSelectShot: (shotKey: string) => void
  selectedShotKey: string
  viewMode: ViewMode
}) {
  return (
    <div className={viewMode === 'compact' ? 'shot-grid compact' : 'shot-grid'} aria-label="分镜镜头卡片">
      {displayShots.map((shot) => (
        <ShotCard key={shot.key} onSelect={() => onSelectShot(shot.key)} selected={shot.key === selectedShotKey} shot={shot} />
      ))}
      <button className="add-shot-card" onClick={onAddLocalShot} type="button">
        <Plus aria-hidden="true" />
        <span>添加草稿镜头</span>
        <kbd>⌘ N</kbd>
      </button>
    </div>
  )
}

function ShotCard({ onSelect, selected, shot }: { onSelect: () => void; selected: boolean; shot: StudioShot }) {
  const generatePromptPack = useGenerateShotPromptPack()
  const canGeneratePack = Boolean(shot.id)
  return (
    <article className={selected ? 'shot-card selected' : 'shot-card'} onClick={onSelect}>
      <button className="shot-select-button" onClick={onSelect} type="button">
        <span className="sr-only">选中第 {shot.code} 镜</span>
      </button>
      <div className={`shot-thumbnail ${shot.thumbnail}`} aria-hidden="true">
        <span>{shot.code}</span>
        <small>{formatDuration(shot.durationMS)}</small>
      </div>
      <div className="shot-card-body">
        <div>
          <span className="scene-chip">{shot.sceneCode}</span>
          <strong>{shot.sceneName}</strong>
        </div>
        <p>{shot.description}</p>
        <div className="tag-row">
          {shot.tags.map((tag) => (
            <span key={tag}>{tag}</span>
          ))}
        </div>
      </div>
      <div className="shot-status-row">
        <span className={`status-dot ${shot.status}`}>{statusLabel(shot.status)}</span>
        <small>{shot.progress}%</small>
      </div>
      <div className="progress-track">
        <span style={{ width: `${shot.progress}%` }} />
      </div>
      <button className="shot-inline-action" disabled={!canGeneratePack || generatePromptPack.isPending} onClick={(event) => {
        event.stopPropagation()
        if (shot.id) generatePromptPack.mutate(shot.id)
      }} type="button">
        <Zap aria-hidden="true" />
        生成提示词包
      </button>
    </article>
  )
}

function ShotInspector({
  activeEpisode,
  displayShots,
  inspectorTab,
  note,
  onInspectorTabChange,
  onNoteChange,
  onPromptDraftChange,
  onSavePrompt,
  onSaveShot,
  onSelectNext,
  onSelectPrevious,
  onShotDraftChange,
  project,
  promptDraft,
  savingPrompt,
  savingShot,
  selectedShot,
  shotDraft,
}: {
  activeEpisode?: Episode
  displayShots: StudioShot[]
  inspectorTab: InspectorTab
  note: string
  onInspectorTabChange: (tab: InspectorTab) => void
  onNoteChange: (value: string) => void
  onPromptDraftChange: (value: string) => void
  onSavePrompt: () => void
  onSaveShot: () => void
  onSelectNext: () => void
  onSelectPrevious: () => void
  onShotDraftChange: (draft: ShotDraft) => void
  project?: Project
  promptDraft?: string
  savingPrompt: boolean
  savingShot: boolean
  selectedShot: StudioShot
  shotDraft?: ShotDraft
}) {
  const { data: promptPack } = useShotPromptPack(selectedShot.id)
  const { data: assets = [] } = useEpisodeAssets(activeEpisode?.id)
  const { data: gates = [] } = useEpisodeApprovalGates(activeEpisode?.id)
  const approveGate = useApproveApprovalGate()

  return (
    <aside className="shot-inspector" aria-label="当前镜头检查器">
      <InspectorHeader onSelectNext={onSelectNext} onSelectPrevious={onSelectPrevious} selectedShot={selectedShot} />
      <InspectorTabs activeTab={inspectorTab} onChange={onInspectorTabChange} />
      {inspectorTab === 'details' ? (
        <SceneTimingCard
          draft={shotDraftValue(selectedShot, shotDraft)}
          onDraftChange={onShotDraftChange}
          onSave={onSaveShot}
          saving={savingShot}
          selectedShot={selectedShot}
        />
      ) : null}
      {inspectorTab === 'prompt' ? (
        <PromptPackCard
          onPromptDraftChange={onPromptDraftChange}
          onSavePrompt={onSavePrompt}
          pack={promptPack}
          promptDraft={promptDraft}
          savingPrompt={savingPrompt}
          selectedShot={selectedShot}
        />
      ) : null}
      {inspectorTab === 'references' ? <ReferenceTokens assets={assets} pack={promptPack} /> : null}
      {inspectorTab === 'notes' ? <ShotNotes note={note} onNoteChange={onNoteChange} selectedShot={selectedShot} /> : null}
      <ModelPresetCard />
      <ApprovalStatusCard activeEpisode={activeEpisode} approveGate={approveGate} gates={gates} />
      <InspectorActions activeEpisode={activeEpisode} disabled={!project || !selectedShot.id} displayShots={displayShots} selectedShot={selectedShot} />
    </aside>
  )
}

function InspectorHeader({
  onSelectNext,
  onSelectPrevious,
  selectedShot,
}: {
  onSelectNext: () => void
  onSelectPrevious: () => void
  selectedShot: StudioShot
}) {
  return (
    <header className="inspector-header">
      <div>
        <strong>第 {selectedShot.code} 镜</strong>
        <span>已选中</span>
      </div>
      <div className="inspector-nav">
        <button aria-label="上一镜" onClick={onSelectPrevious} type="button">‹</button>
        <button aria-label="下一镜" onClick={onSelectNext} type="button">›</button>
      </div>
    </header>
  )
}

function InspectorTabs({ activeTab, onChange }: { activeTab: InspectorTab; onChange: (tab: InspectorTab) => void }) {
  const tabs: { key: InspectorTab; label: string }[] = [
    { key: 'details', label: '镜头信息' },
    { key: 'prompt', label: '提示词' },
    { key: 'references', label: '参考资产' },
    { key: 'notes', label: '导演备注' },
  ]
  return (
    <nav className="inspector-tabs" aria-label="镜头详情分区">
      {tabs.map((tab) => (
        <button className={activeTab === tab.key ? 'active' : ''} key={tab.key} onClick={() => onChange(tab.key)} type="button">{tab.label}</button>
      ))}
    </nav>
  )
}

function SceneTimingCard({
  draft,
  onDraftChange,
  onSave,
  saving,
  selectedShot,
}: {
  draft: ShotDraft
  onDraftChange: (draft: ShotDraft) => void
  onSave: () => void
  saving: boolean
  selectedShot: StudioShot
}) {
  return (
    <section className="inspector-section" aria-labelledby="scene-timing-title">
      <div className="section-title-row">
        <h2 id="scene-timing-title">场景与节奏</h2>
        <button disabled={saving} onClick={onSave} type="button">保存镜头</button>
      </div>
      <label className="field-editor">
        <span>镜头标题</span>
        <input onChange={(event) => onDraftChange({ ...draft, title: event.target.value })} value={draft.title} />
      </label>
      <label className="field-editor">
        <span>镜头说明</span>
        <textarea onChange={(event) => onDraftChange({ ...draft, description: event.target.value })} value={draft.description} />
      </label>
      <label className="field-editor">
        <span>时长 ms</span>
        <input min={1} onChange={(event) => onDraftChange({ ...draft, durationMS: Number(event.target.value) })} type="number" value={draft.durationMS} />
      </label>
      <dl className="detail-list">
        <div><dt>场景</dt><dd><span className="scene-chip">{selectedShot.sceneCode}</span> {selectedShot.sceneName}</dd></div>
        <div><dt>镜头时长</dt><dd>{formatDuration(selectedShot.durationMS)}</dd></div>
        <div><dt>帧率</dt><dd>24 fps</dd></div>
        <div><dt>画幅</dt><dd>16:9 宽银幕</dd></div>
      </dl>
    </section>
  )
}

function PromptPackCard({
  onPromptDraftChange,
  onSavePrompt,
  pack,
  promptDraft,
  savingPrompt,
  selectedShot,
}: {
  onPromptDraftChange: (value: string) => void
  onSavePrompt: () => void
  pack?: ShotPromptPack
  promptDraft?: string
  savingPrompt: boolean
  selectedShot: StudioShot
}) {
  const generatePromptPack = useGenerateShotPromptPack()
  const promptText = promptDraft ?? pack?.direct_prompt ?? selectedShot.prompt
  return (
    <section className="inspector-section prompt-section" aria-labelledby="prompt-pack-title">
      <div className="section-title-row">
        <h2 id="prompt-pack-title">SD2 镜头提示词包</h2>
        <div className="section-actions">
          <button disabled={savingPrompt} onClick={onSavePrompt} type="button">保存提示词</button>
          <button disabled={!selectedShot.id || generatePromptPack.isPending} onClick={() => selectedShot.id && generatePromptPack.mutate(selectedShot.id)} type="button">
            重新生成
          </button>
        </div>
      </div>
      <label className="prompt-editor">
        <span>可编辑提示词</span>
        <textarea onChange={(event) => onPromptDraftChange(event.target.value)} value={promptText} />
      </label>
      <small>{promptText.length}/2000</small>
    </section>
  )
}

function ReferenceTokens({ assets, pack }: { assets: Asset[]; pack?: ShotPromptPack }) {
  const refs = pack?.reference_bindings.map((ref, index) => ({ label: ref.token, thumbnail: thumbnailByIndex(index) })) ?? demoReferences
  return (
    <section className="inspector-section" aria-labelledby="reference-token-title">
      <div className="section-title-row">
        <h2 id="reference-token-title">参考资产令牌</h2>
        <span>{refs.length} / {Math.max(10, assets.length)}</span>
      </div>
      <div className="reference-grid">
        {refs.slice(0, 4).map((ref) => (
          <div className="reference-card" key={ref.label}>
            <span className={`reference-thumb ${ref.thumbnail}`} aria-hidden="true" />
            <strong>{ref.label}</strong>
          </div>
        ))}
      </div>
    </section>
  )
}

function ModelPresetCard() {
  return (
    <section className="inspector-section" aria-labelledby="model-preset-title">
      <h2 id="model-preset-title">生成模型预设</h2>
      <button className="model-select" type="button">
        <span><strong>Seedance Fast</strong><small>优先保证出片速度与动作连续性</small></span>
        <span className="model-badge">SD2.1</span>
        <ChevronDown aria-hidden="true" />
      </button>
    </section>
  )
}

function ApprovalStatusCard({
  activeEpisode,
  approveGate,
  gates,
}: {
  activeEpisode?: Episode
  approveGate: ReturnType<typeof useApproveApprovalGate>
  gates: ApprovalGate[]
}) {
  const seedApprovalGates = useSeedApprovalGates()
  const pendingGate = gates.find((gate) => gate.status === 'pending')
  const approvedGate = gates.find((gate) => gate.status === 'approved')
  return (
    <section className="inspector-section approval-status" aria-labelledby="approval-status-title">
      <h2 id="approval-status-title">人审状态</h2>
      <span className={approvedGate ? 'approved-pill' : 'pending-pill'}>{approvedGate ? '已通过' : '等待确认'}</span>
      <small>{approvedGate?.reviewed_by ? `${approvedGate.reviewed_by} 已确认` : '进入昂贵生成前，需要人工确认。'}</small>
      <div className="approval-actions">
        <button disabled={!activeEpisode || seedApprovalGates.isPending} onClick={() => activeEpisode && seedApprovalGates.mutate(activeEpisode.id)} type="button">刷新审批点</button>
        <button disabled={!pendingGate || approveGate.isPending} onClick={() => pendingGate && approveGate.mutate({ episodeId: pendingGate.episode_id, gateId: pendingGate.id })} type="button">确认通过</button>
      </div>
    </section>
  )
}

function ShotNotes({ note, onNoteChange, selectedShot }: { note: string; onNoteChange: (value: string) => void; selectedShot: StudioShot }) {
  return (
    <section className="inspector-section prompt-section" aria-labelledby="shot-notes-title">
      <h2 id="shot-notes-title">导演备注</h2>
      <label className="prompt-editor">
        <span>给后续生成和剪辑的备注</span>
        <textarea
          onChange={(event) => onNoteChange(event.target.value)}
          placeholder={`例如：第 ${selectedShot.code} 镜需要保留云海开阔感，人物不要贴脸。`}
          value={note}
        />
      </label>
    </section>
  )
}

function InspectorActions({
  activeEpisode,
  disabled,
  displayShots,
  selectedShot,
}: {
  activeEpisode?: Episode
  disabled: boolean
  displayShots: StudioShot[]
  selectedShot: StudioShot
}) {
  const generatePromptPack = useGenerateShotPromptPack()
  const saveTimeline = useSaveEpisodeTimeline()
  const startVideoGeneration = useStartShotVideoGeneration()
  const regenerate = () => {
    if (!selectedShot.id) return
    generatePromptPack.mutate(selectedShot.id, { onSuccess: () => startVideoGeneration.mutate(selectedShot.id ?? '') })
  }

  return (
    <div className="inspector-actions">
      <button className="primary-inspector-action" disabled={disabled || startVideoGeneration.isPending || generatePromptPack.isPending} onClick={regenerate} type="button">
        <Sparkles aria-hidden="true" />
        重新出片
      </button>
      <button disabled={disabled || generatePromptPack.isPending} onClick={() => selectedShot.id && generatePromptPack.mutate(selectedShot.id)} type="button">重写提示词</button>
      <button disabled={disabled || !activeEpisode || saveTimeline.isPending} onClick={() => activeEpisode && saveTimeline.mutate({ episodeId: activeEpisode.id, request: buildTimelineRequest(displayShots) })} type="button">送入时间线</button>
    </div>
  )
}

function TimelineDock({
  activeEpisode,
  displayShots,
  onAddLocalShot,
}: {
  activeEpisode?: Episode
  displayShots: StudioShot[]
  onAddLocalShot: () => void
}) {
  const { data: timeline } = useEpisodeTimeline(activeEpisode?.id)
  const saveTimeline = useSaveEpisodeTimeline()
  const startExport = useStartEpisodeExport()
  const exportQuery = useExport(startExport.data?.id)
  const activeExport = exportQuery.data ?? startExport.data
  const duration = displayShots.reduce((total, shot) => total + shot.durationMS, 0)
  const saveDraft = () => {
    if (!activeEpisode) return
    saveTimeline.mutate({ episodeId: activeEpisode.id, request: buildTimelineRequest(displayShots) })
  }

  return (
    <section className="timeline-dock" aria-labelledby="timeline-title">
      <div className="timeline-toolbar">
        <h2 id="timeline-title">剪辑时间线</h2>
        <div className="transport-controls" aria-label="播放控制">
          <button aria-label="切开片段" type="button"><Scissors aria-hidden="true" /></button>
          <button aria-label="播放预览" type="button"><Play aria-hidden="true" /></button>
          <button aria-label="全屏预览" type="button"><Maximize2 aria-hidden="true" /></button>
        </div>
        <span className="timecode">00:00:11:12</span>
        <button className="ghost-action" disabled={!activeEpisode || saveTimeline.isPending} onClick={saveDraft} type="button">保存剪辑</button>
        <button className="ghost-action" disabled={!activeEpisode || !timeline || startExport.isPending} onClick={() => activeEpisode && startExport.mutate(activeEpisode.id)} type="button">开始导出</button>
      </div>
      <TimelineRuler />
      <TimelineTracks displayShots={displayShots} onAddLocalShot={onAddLocalShot} />
      <footer className="timeline-footer">
        <span>总时长 {formatTimecode(duration)}</span>
        <span>{displayShots.length} 镜 · {Math.round(duration / 1000)} 秒</span>
        <span>导出预设 1080p · H.264 · 24fps</span>
        <span>导出状态 {activeExport ? exportStatusLabel(activeExport.status) : '可预览'}</span>
      </footer>
    </section>
  )
}

function TimelineRuler() {
  return (
    <div className="timeline-ruler" aria-hidden="true">
      {['00:00:00', '00:00:05', '00:00:10', '00:00:15', '00:00:20', '00:00:25', '00:00:30'].map((mark) => (
        <span key={mark}>{mark}</span>
      ))}
    </div>
  )
}

function TimelineTracks({ displayShots, onAddLocalShot }: { displayShots: StudioShot[]; onAddLocalShot: () => void }) {
  return (
    <div className="timeline-tracks" aria-label="剪辑轨道">
      <TrackLabel icon={Film} label="V1" name="画面" />
      <div className="timeline-strip video-strip">
        {displayShots.map((shot) => <TimelineClip key={shot.key} shot={shot} />)}
        <button className="add-clip" onClick={onAddLocalShot} type="button"><Plus aria-hidden="true" /> 添加片段</button>
      </div>
      <TrackLabel icon={Music2} label="A1" name="配乐" />
      <div className="audio-wave">BGM_九霄之上_Main Theme.wav</div>
      <TrackLabel icon={Subtitles} label="S1" name="字幕" />
      <div className="subtitle-strip">{displayShots.map((shot) => <span key={shot.key}>{subtitleForShot(shot.code)}</span>)}</div>
      <TrackLabel icon={Scissors} label="T1" name="转场" />
      <div className="transition-strip"><span>云雾叠化 00:00:15</span><span>雷光闪切 00:00:10</span></div>
    </div>
  )
}

function TrackLabel({ icon: Icon, label, name }: { icon: typeof Activity; label: string; name: string }) {
  return (
    <div className="track-label">
      <span>{label}</span>
      <Icon aria-hidden="true" />
      <strong>{name}</strong>
      <Lock aria-hidden="true" />
      <Eye aria-hidden="true" />
    </div>
  )
}

function TimelineClip({ shot }: { shot: StudioShot }) {
  return (
    <article className="timeline-clip">
      <span className={`clip-thumb ${shot.thumbnail}`} aria-hidden="true" />
      <strong>{shot.code}</strong>
      <small>{formatDuration(shot.durationMS)}</small>
    </article>
  )
}

function mapDisplayShots(shots: StoryboardShot[]): StudioShot[] {
  if (shots.length === 0) return demoShots
  return shots.map((shot, index) => ({
    code: shot.code || `${index + 1}`.padStart(2, '0'),
    description: shot.description || '系统生成的分镜镜头',
    durationMS: shot.duration_ms,
    id: shot.id,
    key: shot.id,
    progress: index % 3 === 1 ? 72 : index % 3 === 2 ? 0 : 100,
    prompt: shot.prompt,
    sceneCode: `SC${((index % 3) + 1).toString().padStart(2, '0')}`,
    sceneName: shot.title,
    status: index % 3 === 1 ? 'generating' : index % 3 === 2 ? 'queued' : 'prompt_ready',
    tags: ['云澜', '白璃', '长老'].slice(0, (index % 3) + 1),
    thumbnail: thumbnailByIndex(index),
    title: shot.title,
  }))
}

function buildTimelineRequest(shots: StudioShot[]): SaveTimelineRequest {
  return {
    duration_ms: shots.reduce((total, shot) => total + shot.durationMS, 0),
    tracks: [{
      clips: shots.map((shot, index) => ({
        asset_id: undefined,
        duration_ms: shot.durationMS,
        kind: 'shot',
        start_ms: shots.slice(0, index).reduce((total, item) => total + item.durationMS, 0),
        trim_start_ms: 0,
      })),
      kind: 'video',
      name: '分镜视频轨',
      position: 1,
    }],
  }
}

function statusLabel(status: StudioShot['status']): string {
  const labels: Record<StudioShot['status'], string> = {
    approved: '已通过',
    draft: '草稿',
    generating: '生成中',
    prompt_ready: '提示词就绪',
    queued: '排队中',
  }
  return labels[status]
}

function shotDraftValue(shot: StudioShot, draft?: ShotDraft): ShotDraft {
  return draft ?? {
    description: shot.description,
    durationMS: shot.durationMS,
    title: shot.title || shot.sceneName,
  }
}

function hasStoryMapItems(storyMap?: StoryMap): boolean {
  if (!storyMap) return false
  return storyMap.characters.length + storyMap.scenes.length + storyMap.props.length > 0
}

function jobStatusLabel(status: GenerationJob['status']): string {
  const labels: Record<GenerationJob['status'], string> = {
    blocked: '已阻塞',
    canceled: '已取消',
    canceling: '取消中',
    downloading: '下载中',
    draft: '草稿',
    failed: '失败',
    needs_review: '待人审',
    polling: '轮询中',
    postprocessing: '后处理',
    preflight: '预检中',
    queued: '排队中',
    submitting: '提交中',
    submitted: '已提交',
    succeeded: '已完成',
    timed_out: '超时',
  }
  return labels[status] ?? status.replaceAll('_', ' ')
}

function exportStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    canceled: '已取消',
    completed: '已完成',
    failed: '导出失败',
    processing: '导出中',
    queued: '排队中',
    rendering: '渲染中',
    requested: '已提交',
    succeeded: '已完成',
  }
  return labels[status] ?? status.replaceAll('_', ' ')
}

function thumbnailByIndex(index: number): string {
  const thumbnails = ['thumb-cloud', 'thumb-hero', 'thumb-muse', 'thumb-temple', 'thumb-battle', 'thumb-beast']
  return thumbnails[index % thumbnails.length]
}

function formatDuration(durationMS: number): string {
  return `00:${(durationMS / 1000).toFixed(1).padStart(4, '0')}`
}

function formatTimecode(durationMS: number): string {
  const totalSeconds = Math.round(durationMS / 1000)
  return `00:00:${totalSeconds.toString().padStart(2, '0')}:11`
}

function subtitleForShot(code: string): string {
  const subtitles: Record<string, string> = {
    '01': '云海之上，天门将开。',
    '02': '我云澜，必登九霄！',
    '03': '白璃师姐？',
    '04': '天玄宗，外门广场。',
    '05': '今日试炼，开始！',
    '06': '这...这是什么？',
  }
  return subtitles[code] ?? '镜头对白待生成。'
}

export default App
