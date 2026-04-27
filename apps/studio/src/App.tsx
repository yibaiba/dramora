import {
  Activity,
  BookOpenText,
  Boxes,
  CircleDollarSign,
  Clapperboard,
  Film,
  Layers3,
  Library,
  ListChecks,
  Play,
  Plus,
  Radio,
  Sparkles,
  WandSparkles,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import {
  useCreateEpisode,
  useCreateProject,
  useEpisodeTimeline,
  useEpisodeAssets,
  useEpisodes,
  useGenerationJobs,
  useLockAsset,
  useProjects,
  useSaveEpisodeTimeline,
  useGenerateShotPromptPack,
  useSeedEpisodeAssets,
  useSeedStoryboardShots,
  useSeedStoryMap,
  useShotPromptPack,
  useStoryAnalyses,
  useStoryboardShots,
  useStoryMap,
  useStartEpisodeExport,
  useStartStoryAnalysis,
} from './api/hooks'
import { useStudioStore } from './state/studioStore'
import type {
  Episode,
  Asset,
  GenerationJob,
  GenerationJobStatus,
  Project,
  SaveTimelineRequest,
  ShotPromptPack,
  StoryAnalysis,
  StoryMap,
  StoryMapItem,
  StoryboardShot,
  TimelineTrack,
} from './api/types'

const productionSteps = [
  { name: 'Producer', taskType: 'production_plan', detail: '目标、预算、审批点' },
  { name: 'Story Analyst', taskType: 'story_analysis', detail: '故事分析与角色/场景/道具抽取' },
  { name: 'Character Designer', taskType: 'character_design', detail: '人物卡、三视图、表情与姿势' },
  { name: 'Scene Designer', taskType: 'scene_design', detail: '场景卡、氛围、光线与概念图' },
  { name: 'Storyboard', taskType: 'storyboard', detail: '镜头卡、首尾帧与 prompt pack' },
  { name: 'Continuity Supervisor', taskType: 'continuity_review', detail: '人物、服装、场景、道具连续性检查' },
]

type AgentStepStatus = 'ready' | 'queued' | 'running' | 'succeeded' | 'waiting' | 'blocked'

type AgentStep = {
  detail: string
  jobCount: number
  latestJob?: GenerationJob
  name: string
  status: AgentStepStatus
  statusLabel: string
}

const activeGenerationStatuses: GenerationJobStatus[] = [
  'queued',
  'submitting',
  'submitted',
  'polling',
  'downloading',
  'postprocessing',
]

const blockedGenerationStatuses: GenerationJobStatus[] = ['blocked', 'failed', 'timed_out', 'canceling', 'canceled']

const storyboardColumns = [
  { title: 'Planned', count: 4, tone: 'blue' },
  { title: 'Prompt ready', count: 0, tone: 'violet' },
  { title: 'Generating', count: 0, tone: 'orange' },
  { title: 'Review', count: 0, tone: 'green' },
]

function App() {
  const { data: projects = [], isLoading: projectsLoading } = useProjects()
  const { selectedProjectId, selectProject, logEvent } = useStudioStore()
  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId) ?? projects[0],
    [projects, selectedProjectId],
  )
  const { data: selectedEpisodes = [] } = useEpisodes(selectedProject?.id)
  const activeEpisode = selectedEpisodes[0]

  useEffect(() => {
    if (!selectedProjectId && selectedProject) {
      selectProject(selectedProject.id)
    }
  }, [selectProject, selectedProject, selectedProjectId])

  return (
    <main className="studio-shell">
      <Sidebar />
      <section className="studio-main" aria-label="Manmu Studio workspace">
        <StudioHeader project={selectedProject} />
        <div className="workspace-grid">
          <ProjectPanel
            projects={projects}
            selectedProjectId={selectedProject?.id}
            isLoading={projectsLoading}
            onSelect={selectProject}
          />
          <EpisodeCommandCenter project={selectedProject} episodes={selectedEpisodes} onLog={logEvent} />
          <StoryAnalysisPanel activeEpisode={activeEpisode} project={selectedProject} />
          <AgentBoard project={selectedProject} />
          <StoryboardKanban activeEpisode={activeEpisode} />
          <AssetLibrary activeEpisode={activeEpisode} />
          <TimelineEditor activeEpisode={activeEpisode} />
        </div>
      </section>
      <JobsRail />
    </main>
  )
}

function Sidebar() {
  const items = [
    { label: 'Projects', icon: Layers3 },
    { label: 'Command', icon: Clapperboard },
    { label: 'Agents', icon: Sparkles },
    { label: 'Assets', icon: Library },
    { label: 'Timeline', icon: Film },
  ]

  return (
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark">
        <WandSparkles aria-hidden="true" />
        <div>
          <strong>漫幕</strong>
          <span>AI Manju Studio</span>
        </div>
      </div>
      <nav className="nav-list">
        {items.map((item, index) => (
          <button className={index === 0 ? 'nav-item active' : 'nav-item'} key={item.label} type="button">
            <item.icon aria-hidden="true" />
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
      <div className="sidebar-card">
        <CircleDollarSign aria-hidden="true" />
        <span>Budget guard</span>
        <strong>¥0.00 / MVP</strong>
      </div>
    </aside>
  )
}

function StudioHeader({ project }: { project?: Project }) {
  return (
    <header className="studio-header">
      <div>
        <p className="eyebrow">AI generated manju production workspace</p>
        <h1>{project ? project.name : 'Create your first Manmu project'}</h1>
        <p className="header-copy">
          Story analysis, character and scene maps, agent approvals, shot generation, timeline export.
        </p>
      </div>
      <div className="header-stats" aria-label="Production status">
        <StatusCard label="Workflow" value="Fixed SOP" icon={ListChecks} />
        <StatusCard label="Realtime" value="SSE ready" icon={Radio} />
        <StatusCard label="Export" value="FFmpeg planned" icon={Play} />
      </div>
    </header>
  )
}

function StatusCard({ label, value, icon: Icon }: { label: string; value: string; icon: typeof Activity }) {
  return (
    <div className="status-card">
      <Icon aria-hidden="true" />
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function ProjectPanel({
  projects,
  selectedProjectId,
  isLoading,
  onSelect,
}: {
  projects: Project[]
  selectedProjectId?: string
  isLoading: boolean
  onSelect: (projectId: string) => void
}) {
  const createProject = useCreateProject()
  const [name, setName] = useState('')

  const submitProject = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    createProject.mutate(
      { name, description: 'AI manju production project' },
      {
        onSuccess: (project) => {
          onSelect(project.id)
          setName('')
        },
      },
    )
  }

  return (
    <section className="panel project-panel" aria-labelledby="project-panel-title">
      <PanelTitle icon={Layers3} title="Project list" subtitle="Go API backed CRUD" id="project-panel-title" />
      <form className="inline-form" onSubmit={submitProject}>
        <label>
          <span>Project name</span>
          <input
            minLength={1}
            onChange={(event) => setName(event.target.value)}
            placeholder="例如：夜幕契约"
            required
            value={name}
          />
        </label>
        <button disabled={createProject.isPending} type="submit">
          <Plus aria-hidden="true" />
          Create
        </button>
      </form>
      <div className="project-list" aria-busy={isLoading}>
        {projects.length === 0 ? (
          <EmptyState title="No projects yet" text="Create a project to start story analysis and asset mapping." />
        ) : (
          projects.map((project) => (
            <button
              className={project.id === selectedProjectId ? 'project-row active' : 'project-row'}
              key={project.id}
              onClick={() => onSelect(project.id)}
              type="button"
            >
              <span>{project.name}</span>
              <small>{project.status}</small>
            </button>
          ))
        )}
      </div>
    </section>
  )
}

function EpisodeCommandCenter({
  episodes,
  onLog,
  project,
}: {
  episodes: Episode[]
  onLog: (message: string) => void
  project?: Project
}) {
  const createEpisode = useCreateEpisode(project?.id)
  const startStoryAnalysis = useStartStoryAnalysis()
  const saveTimeline = useSaveEpisodeTimeline()
  const [title, setTitle] = useState('')

  const submitEpisode = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!project) return
    createEpisode.mutate(
      { title },
      {
        onSuccess: (episode) => {
          onLog(`Episode ${episode.number} created: ${episode.title}`)
          setTitle('')
        },
      },
    )
  }

  return (
    <section className="panel command-panel" aria-labelledby="episode-command-title">
      <PanelTitle icon={Clapperboard} title="Episode command center" subtitle="Script to production gates" id="episode-command-title" />
      <div className="command-layout">
        <form className="inline-form" onSubmit={submitEpisode}>
          <label>
            <span>Episode title</span>
            <input disabled={!project} onChange={(event) => setTitle(event.target.value)} required value={title} />
          </label>
          <button disabled={!project || createEpisode.isPending} type="submit">
            Add episode
          </button>
        </form>
        <div className="episode-list">
          {episodes.length === 0 ? (
            <EmptyState title="No episodes" text="Add an episode to unlock story analysis and storyboard planning." />
          ) : (
            episodes.map((episode) => (
              <div className="episode-row" key={episode.id}>
                <strong>EP{episode.number.toString().padStart(2, '0')}</strong>
                <span>{episode.title}</span>
                <small>{episode.status}</small>
                <button
                  className="secondary-action"
                  disabled={startStoryAnalysis.isPending}
                  onClick={() =>
                    startStoryAnalysis.mutate(episode.id, {
                      onSuccess: (result) => {
                        onLog(`Story analysis queued: ${result.generation_job.id.slice(0, 8)}`)
                      },
                    })
                  }
                  type="button"
                >
                  Start analysis
                </button>
                <button
                  className="secondary-action"
                  disabled={saveTimeline.isPending}
                  onClick={() =>
                    saveTimeline.mutate(
                      { episodeId: episode.id, request: { duration_ms: 15_000 } },
                      {
                        onSuccess: (timeline) => {
                          onLog(`Timeline saved v${timeline.version}: ${timeline.duration_ms}ms`)
                        },
                      },
                    )
                  }
                  type="button"
                >
                  Save timeline
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </section>
  )
}

function AgentBoard({ project }: { project?: Project }) {
  const { data: jobs = [], isLoading } = useGenerationJobs()
  const projectJobs = useMemo(
    () => jobs.filter((job) => job.project_id === project?.id),
    [jobs, project?.id],
  )
  const agentSteps = useMemo(() => buildAgentSteps(projectJobs, Boolean(project)), [projectJobs, project])

  return (
    <section className="panel" aria-labelledby="agent-board-title">
      <PanelTitle
        icon={Sparkles}
        title="Agent Board"
        subtitle={isLoading ? 'Loading workflow telemetry' : 'Live SOP state from generation jobs'}
        id="agent-board-title"
      />
      <div className="agent-list">
        {agentSteps.map((step) => (
          <div className={`agent-step ${step.status}`} key={step.name}>
            <div>
              <strong>{step.name}</strong>
              <span>{step.detail}</span>
            </div>
            <small>
              {step.statusLabel}
              {step.jobCount > 0 ? ` · ${step.jobCount} job${step.jobCount > 1 ? 's' : ''}` : ''}
            </small>
          </div>
        ))}
      </div>
    </section>
  )
}

function StoryAnalysisPanel({ activeEpisode, project }: { activeEpisode?: Episode; project?: Project }) {
  const { data: analyses = [], isLoading } = useStoryAnalyses(activeEpisode?.id)
  const latestAnalysis = analyses[0]

  return (
    <section className="panel analysis-panel" aria-labelledby="story-analysis-title">
      <PanelTitle
        icon={BookOpenText}
        title="Story analysis"
        subtitle={isLoading ? 'Loading structured artifacts' : 'Generated C/S/P seeds'}
        id="story-analysis-title"
      />
      {!project ? (
        <EmptyState title="Select a project" text="Story analysis artifacts appear after an episode analysis job succeeds." />
      ) : !activeEpisode ? (
        <EmptyState title="No episode selected" text="Create an episode, start analysis, then run the worker once." />
      ) : !latestAnalysis ? (
        <EmptyState title="No analysis artifact" text="Run the worker after starting analysis to generate the first structured artifact." />
      ) : (
        <StoryAnalysisSummary episode={activeEpisode} analysis={latestAnalysis} />
      )}
    </section>
  )
}

function StoryAnalysisSummary({ analysis, episode }: { analysis: StoryAnalysis; episode: Episode }) {
  const metrics = [
    { label: 'Characters', value: analysis.character_seeds.length },
    { label: 'Scenes', value: analysis.scene_seeds.length },
    { label: 'Props', value: analysis.prop_seeds.length },
  ]

  return (
    <div className="analysis-card">
      <div className="analysis-meta">
        <span>EP{episode.number.toString().padStart(2, '0')}</span>
        <strong>v{analysis.version}</strong>
        <small>{analysis.status}</small>
      </div>
      <p>{analysis.summary}</p>
      <div className="analysis-metrics" aria-label="Story analysis seed counts">
        {metrics.map((metric) => (
          <span key={metric.label}>
            <strong>{metric.value}</strong>
            {metric.label}
          </span>
        ))}
      </div>
      <div className="analysis-tags" aria-label="Story themes">
        {analysis.themes.map((theme) => (
          <span key={theme}>{theme}</span>
        ))}
      </div>
    </div>
  )
}

function buildAgentSteps(jobs: GenerationJob[], hasProject: boolean): AgentStep[] {
  const storyAnalysisDone = jobs.some((job) => job.task_type === 'story_analysis' && job.status === 'succeeded')
  return productionSteps.map((step) => {
    if (step.taskType === 'production_plan') {
      return {
        ...step,
        jobCount: 0,
        status: hasProject ? 'ready' : 'waiting',
        statusLabel: hasProject ? 'Ready' : 'Select project',
      }
    }

    const matchingJobs = jobs.filter((job) => job.task_type === step.taskType)
    if (matchingJobs.length > 0) {
      return summarizeAgentStep(step, matchingJobs)
    }
    return {
      ...step,
      jobCount: 0,
      status: storyAnalysisDone ? 'ready' : 'waiting',
      statusLabel: storyAnalysisDone ? 'Ready' : 'Waiting',
    }
  })
}

function summarizeAgentStep(step: (typeof productionSteps)[number], jobs: GenerationJob[]): AgentStep {
  const latestJob = [...jobs].sort((a, b) => b.updated_at.localeCompare(a.updated_at))[0]
  return {
    ...step,
    jobCount: jobs.length,
    latestJob,
    status: agentStatusFromJob(latestJob.status),
    statusLabel: latestJob.status.replaceAll('_', ' '),
  }
}

function agentStatusFromJob(status: GenerationJobStatus): AgentStepStatus {
  if (status === 'succeeded') return 'succeeded'
  if (blockedGenerationStatuses.includes(status)) return 'blocked'
  if (activeGenerationStatuses.includes(status)) return status === 'queued' ? 'queued' : 'running'
  return 'waiting'
}

function StoryboardKanban({ activeEpisode }: { activeEpisode?: Episode }) {
  const { data: shots = [], isLoading } = useStoryboardShots(activeEpisode?.id)
  const seedStoryboard = useSeedStoryboardShots()
  const columns = buildStoryboardColumns(shots)

  return (
    <section className="panel" aria-labelledby="storyboard-title">
      <PanelTitle
        icon={Boxes}
        title="Storyboard Kanban"
        subtitle={isLoading ? 'Loading shot cards' : 'Scene-to-shot prompt cards'}
        id="storyboard-title"
      />
      <PanelToolbar>
        <button
          className="secondary-action"
          disabled={!activeEpisode || seedStoryboard.isPending}
          onClick={() => activeEpisode && seedStoryboard.mutate(activeEpisode.id)}
          type="button"
        >
          Seed storyboard
        </button>
      </PanelToolbar>
      <div className="kanban-grid">
        {columns.map((column) => (
          <div className={`kanban-column ${column.tone}`} key={column.title}>
            <span>{column.title}</span>
            <strong>{column.count}</strong>
          </div>
        ))}
      </div>
      <ShotList activeEpisode={activeEpisode} shots={shots} />
    </section>
  )
}

function buildStoryboardColumns(shots: StoryboardShot[]) {
  return storyboardColumns.map((column) => {
    if (column.title === 'Prompt ready') return { ...column, count: shots.length }
    if (column.title === 'Planned') return { ...column, count: 0 }
    return column
  })
}

function ShotList({ activeEpisode, shots }: { activeEpisode?: Episode; shots: StoryboardShot[] }) {
  if (!activeEpisode) return <EmptyState title="No episode selected" text="Create an episode before seeding storyboard shots." />
  if (shots.length === 0) return <EmptyState title="No shot cards" text="Seed the storyboard after generating the story map." />

  return (
    <div className="shot-grid" aria-label="Storyboard shot cards">
      {shots.map((shot) => (
        <ShotCard key={shot.id} shot={shot} />
      ))}
    </div>
  )
}

function ShotCard({ shot }: { shot: StoryboardShot }) {
  const { data: promptPack } = useShotPromptPack(shot.id)
  const generatePromptPack = useGenerateShotPromptPack()
  const activePromptPack = generatePromptPack.data?.shot_id === shot.id ? generatePromptPack.data : promptPack

  return (
    <article className="shot-card">
      <div className="shot-card-header">
        <strong>{shot.code}</strong>
        <span>{Math.round(shot.duration_ms / 1000)}s</span>
      </div>
      <h3>{shot.title}</h3>
      <p>{shot.description}</p>
      <small>{shot.prompt}</small>
      <div className="prompt-pack-actions">
        <button
          className="secondary-action"
          disabled={generatePromptPack.isPending}
          onClick={() => generatePromptPack.mutate(shot.id)}
          type="button"
        >
          Generate SD2 pack
        </button>
        {activePromptPack ? <PromptPackPreview pack={activePromptPack} /> : null}
      </div>
    </article>
  )
}

function PromptPackPreview({ pack }: { pack: ShotPromptPack }) {
  const copyPrompt = () => {
    void navigator.clipboard.writeText(pack.direct_prompt)
  }

  return (
    <div className="prompt-pack-preview">
      <div className="prompt-pack-meta">
        <span>{pack.preset}</span>
        <span>{pack.task_type.replaceAll('_', ' ')}</span>
        <span>{pack.reference_bindings.length} refs</span>
      </div>
      <p>{pack.direct_prompt}</p>
      <div className="reference-token-list" aria-label="SD2 reference bindings">
        {pack.reference_bindings.map((ref) => (
          <span key={ref.asset_id}>
            {ref.token} · {ref.role} · {ref.kind}
          </span>
        ))}
      </div>
      <button className="secondary-action" onClick={copyPrompt} type="button">
        Copy prompt
      </button>
    </div>
  )
}

function AssetLibrary({ activeEpisode }: { activeEpisode?: Episode }) {
  const { data: storyMap, isLoading } = useStoryMap(activeEpisode?.id)
  const { data: assets = [] } = useEpisodeAssets(activeEpisode?.id)
  const seedStoryMap = useSeedStoryMap()
  const seedAssets = useSeedEpisodeAssets()

  return (
    <section className="panel" aria-labelledby="asset-library-title">
      <PanelTitle
        icon={Library}
        title="Asset library"
        subtitle={isLoading ? 'Loading C/S/P map' : 'Character, scene, and prop map'}
        id="asset-library-title"
      />
      <PanelToolbar>
        <button
          className="secondary-action"
          disabled={!activeEpisode || seedStoryMap.isPending}
          onClick={() => activeEpisode && seedStoryMap.mutate(activeEpisode.id)}
          type="button"
        >
          Seed C/S/P map
        </button>
        <button
          className="secondary-action"
          disabled={!activeEpisode || seedAssets.isPending}
          onClick={() => activeEpisode && seedAssets.mutate(activeEpisode.id)}
          type="button"
        >
          Seed asset candidates
        </button>
      </PanelToolbar>
      <StoryMapGrid activeEpisode={activeEpisode} storyMap={storyMap} />
      <AssetCandidateGrid activeEpisode={activeEpisode} assets={assets} />
    </section>
  )
}

function StoryMapGrid({
  activeEpisode,
  storyMap,
}: {
  activeEpisode?: Episode
  storyMap?: StoryMap
}) {
  if (!activeEpisode) return <EmptyState title="No episode selected" text="Create an episode before seeding C/S/P assets." />
  if (!storyMap) return <EmptyState title="No story map" text="Seed C/S/P map after story analysis succeeds." />

  return (
    <div className="story-map-grid" aria-label="Character scene prop map">
      <StoryMapColumn title="Characters" items={storyMap.characters} emptyText="No character seeds yet" />
      <StoryMapColumn title="Scenes" items={storyMap.scenes} emptyText="No scene seeds yet" />
      <StoryMapColumn title="Props" items={storyMap.props} emptyText="No prop seeds yet" />
    </div>
  )
}

function StoryMapColumn({ emptyText, items, title }: { emptyText: string; items: StoryMapItem[]; title: string }) {
  return (
    <div className="story-map-column">
      <h3>{title}</h3>
      {items.length === 0 ? <small>{emptyText}</small> : null}
      {items.map((item) => (
        <article className="story-map-card" key={item.id}>
          <strong>{item.code}</strong>
          <span>{item.name}</span>
          <small>{item.description}</small>
        </article>
      ))}
    </div>
  )
}

function AssetCandidateGrid({ activeEpisode, assets }: { activeEpisode?: Episode; assets: Asset[] }) {
  const lockAsset = useLockAsset()

  if (!activeEpisode) return null
  if (assets.length === 0) {
    return <EmptyState title="No asset candidates" text="Seed asset candidates after the C/S/P map is ready." />
  }

  return (
    <div className="asset-grid" aria-label="Asset candidates">
      {assets.map((asset) => (
        <article className="asset-card" key={asset.id}>
          <div className="asset-preview" />
          <span>
            {asset.kind} · {asset.purpose}
          </span>
          <small>{asset.status === 'ready' ? 'locked reference' : asset.uri}</small>
          <button
            className="secondary-action"
            disabled={asset.status === 'ready' || lockAsset.isPending}
            onClick={() => lockAsset.mutate({ assetId: asset.id, episodeId: activeEpisode.id })}
            type="button"
          >
            {asset.status === 'ready' ? 'Locked' : 'Lock'}
          </button>
        </article>
      ))}
    </div>
  )
}

function TimelineEditor({ activeEpisode }: { activeEpisode?: Episode }) {
  const { data: timeline } = useEpisodeTimeline(activeEpisode?.id)
  const { data: shots = [] } = useStoryboardShots(activeEpisode?.id)
  const saveTimeline = useSaveEpisodeTimeline()
  const startExport = useStartEpisodeExport()

  const saveShotTimeline = () => {
    if (!activeEpisode) return
    saveTimeline.mutate({ episodeId: activeEpisode.id, request: buildTimelineRequest(shots) })
  }

  return (
    <section className="panel timeline-panel" aria-labelledby="timeline-title">
      <PanelTitle icon={Film} title="Timeline editor" subtitle="Tracks, clips, and export handoff" id="timeline-title" />
      <PanelToolbar>
        <button
          className="secondary-action"
          disabled={!activeEpisode || saveTimeline.isPending}
          onClick={saveShotTimeline}
          type="button"
        >
          Save shot timeline
        </button>
        <button
          className="secondary-action"
          disabled={!activeEpisode || !timeline || startExport.isPending}
          onClick={() => activeEpisode && startExport.mutate(activeEpisode.id)}
          type="button"
        >
          Start export
        </button>
        {startExport.data ? <span className="export-status">Export {startExport.data.status}</span> : null}
      </PanelToolbar>
      <div className="timeline-stage">
        <div className="playhead" aria-hidden="true" />
        <TimelineTracks activeEpisode={activeEpisode} tracks={timeline?.tracks ?? []} />
      </div>
    </section>
  )
}

function buildTimelineRequest(shots: StoryboardShot[]): SaveTimelineRequest {
  const sourceShots = shots.length > 0 ? shots : [{ duration_ms: 15_000 }]
  return {
    duration_ms: sourceShots.reduce((total, shot) => total + shot.duration_ms, 0),
    tracks: [
      {
        clips: sourceShots.map((shot, index) => ({
          kind: 'video',
          start_ms: sourceShots.slice(0, index).reduce((total, item) => total + item.duration_ms, 0),
          duration_ms: shot.duration_ms,
        })),
        kind: 'video',
        name: 'Storyboard video',
        position: 1,
      },
    ],
  }
}

function TimelineTracks({ activeEpisode, tracks }: { activeEpisode?: Episode; tracks: TimelineTrack[] }) {
  if (!activeEpisode) return <EmptyState title="No episode selected" text="Create an episode before saving timeline clips." />
  if (tracks.length === 0) return <EmptyState title="No timeline tracks" text="Save a shot timeline after seeding storyboard shots." />

  return (
    <>
      {tracks.map((track) => (
        <div className="track-row" key={track.id}>
          <span>{track.name}</span>
          <div className="clip-strip">
            {track.clips.map((clip) => (
              <div className="clip-block" key={clip.id}>
                {clip.kind} · {Math.round(clip.duration_ms / 1000)}s
              </div>
            ))}
          </div>
        </div>
      ))}
    </>
  )
}

function JobsRail() {
  const { data: jobs = [] } = useGenerationJobs()
  const eventLog = useStudioStore((state) => state.eventLog)

  return (
    <aside className="jobs-rail" aria-label="Generation jobs and events">
      <PanelTitle icon={Activity} title="Jobs" subtitle="Generation queue" id="jobs-title" />
      <div className="jobs-list">
        {jobs.length === 0 ? <span className="muted">No generation jobs yet</span> : null}
        {jobs.map((job) => (
          <div className="job-row" key={job.id}>
            <strong>{job.task_type}</strong>
            <small>{job.status}</small>
          </div>
        ))}
      </div>
      <div className="event-log">
        {eventLog.map((event) => (
          <small key={event}>{event}</small>
        ))}
      </div>
    </aside>
  )
}

function PanelTitle({
  icon: Icon,
  id,
  subtitle,
  title,
}: {
  icon: typeof Activity
  id: string
  subtitle: string
  title: string
}) {
  return (
    <div className="panel-title">
      <Icon aria-hidden="true" />
      <div>
        <h2 id={id}>{title}</h2>
        <p>{subtitle}</p>
      </div>
    </div>
  )
}

function PanelToolbar({ children }: { children: ReactNode }) {
  return <div className="panel-toolbar">{children}</div>
}

function EmptyState({ text, title }: { text: string; title: string }) {
  return (
    <div className="empty-state">
      <strong>{title}</strong>
      <span>{text}</span>
    </div>
  )
}

export default App
