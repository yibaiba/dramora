export type ProjectStatus = 'draft' | 'active' | 'archived'
export type EpisodeStatus = 'draft' | 'planning' | 'generating' | 'editing' | 'exported' | 'archived'
export type GenerationJobStatus =
  | 'draft'
  | 'preflight'
  | 'queued'
  | 'submitting'
  | 'submitted'
  | 'polling'
  | 'downloading'
  | 'postprocessing'
  | 'needs_review'
  | 'succeeded'
  | 'blocked'
  | 'failed'
  | 'timed_out'
  | 'canceling'
  | 'canceled'

export type Project = {
  id: string
  organization_id: string
  name: string
  description: string
  status: ProjectStatus
  created_at: string
  updated_at: string
}

export type Episode = {
  id: string
  project_id: string
  number: number
  title: string
  status: EpisodeStatus
  created_at: string
  updated_at: string
}

export type GenerationJob = {
  id: string
  project_id: string
  episode_id: string
  workflow_run_id: string
  provider: string
  model: string
  task_type: string
  status: GenerationJobStatus
  created_at: string
  updated_at: string
}

export type WorkflowRun = {
  id: string
  project_id: string
  episode_id: string
  status: 'draft' | 'running' | 'waiting_approval' | 'succeeded' | 'failed' | 'canceled'
  created_at: string
  updated_at: string
}

export type Timeline = {
  id: string
  episode_id: string
  status: 'draft' | 'saved' | 'exporting' | 'exported'
  version: number
  duration_ms: number
  tracks: Record<string, never>[]
  created_at: string
  updated_at: string
}

export type CreateProjectRequest = {
  name: string
  description?: string
}

export type StartStoryAnalysisResponse = {
  workflow_run: WorkflowRun
  generation_job: GenerationJob
}

export type SaveTimelineRequest = {
  duration_ms: number
}

export type CreateEpisodeRequest = {
  title: string
  number?: number
}
