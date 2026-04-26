import {
  Activity,
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
import type { FormEvent } from 'react'
import {
  useCreateEpisode,
  useCreateProject,
  useEpisodes,
  useGenerationJobs,
  useProjects,
  useSaveEpisodeTimeline,
  useStartStoryAnalysis,
} from './api/hooks'
import { useStudioStore } from './state/studioStore'
import type { GenerationJob, GenerationJobStatus, Project } from './api/types'

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

const assetTypes = ['C01 主角', 'S01 夜幕街区', 'P01 黑曜石戒指', 'Keyframe 001']

function App() {
  const { data: projects = [], isLoading: projectsLoading } = useProjects()
  const { selectedProjectId, selectProject, logEvent } = useStudioStore()
  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId) ?? projects[0],
    [projects, selectedProjectId],
  )

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
          <EpisodeCommandCenter project={selectedProject} onLog={logEvent} />
          <AgentBoard project={selectedProject} />
          <StoryboardKanban />
          <AssetLibrary />
          <TimelineEditor project={selectedProject} />
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

function EpisodeCommandCenter({ project, onLog }: { project?: Project; onLog: (message: string) => void }) {
  const { data: episodes = [] } = useEpisodes(project?.id)
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
                      { durationMs: 15_000, episodeId: episode.id },
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

function StoryboardKanban() {
  return (
    <section className="panel" aria-labelledby="storyboard-title">
      <PanelTitle icon={Boxes} title="Storyboard Kanban" subtitle="Shot state map" id="storyboard-title" />
      <div className="kanban-grid">
        {storyboardColumns.map((column) => (
          <div className={`kanban-column ${column.tone}`} key={column.title}>
            <span>{column.title}</span>
            <strong>{column.count}</strong>
          </div>
        ))}
      </div>
    </section>
  )
}

function AssetLibrary() {
  return (
    <section className="panel" aria-labelledby="asset-library-title">
      <PanelTitle icon={Library} title="Asset library" subtitle="C/S/P and keyframe candidates" id="asset-library-title" />
      <div className="asset-grid">
        {assetTypes.map((asset) => (
          <div className="asset-card" key={asset}>
            <div className="asset-preview" />
            <span>{asset}</span>
            <small>candidate grid ready</small>
          </div>
        ))}
      </div>
    </section>
  )
}

function TimelineEditor({ project }: { project?: Project }) {
  return (
    <section className="panel timeline-panel" aria-labelledby="timeline-title">
      <PanelTitle icon={Film} title="Timeline editor" subtitle="Narrow MVP editor foundation" id="timeline-title" />
      <div className="timeline-stage">
        <div className="playhead" aria-hidden="true" />
        {['Video', 'Keyframe', 'Voice', 'Subtitle'].map((track) => (
          <div className="track-row" key={track}>
            <span>{track}</span>
            <div className="clip-block">{project ? 'Waiting for generated clips' : 'Select a project'}</div>
          </div>
        ))}
      </div>
    </section>
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

function EmptyState({ text, title }: { text: string; title: string }) {
  return (
    <div className="empty-state">
      <strong>{title}</strong>
      <span>{text}</span>
    </div>
  )
}

export default App
