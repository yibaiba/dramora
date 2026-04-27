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
  tracks: TimelineTrack[]
  created_at: string
  updated_at: string
}

export type TimelineTrack = {
  id: string
  kind: string
  name: string
  position: number
  clips: TimelineClip[]
  created_at: string
  updated_at: string
}

export type TimelineClip = {
  id: string
  asset_id: string
  kind: string
  start_ms: number
  duration_ms: number
  trim_start_ms: number
  created_at: string
  updated_at: string
}

export type StoryAnalysis = {
  id: string
  project_id: string
  episode_id: string
  workflow_run_id: string
  generation_job_id: string
  version: number
  status: 'generated' | 'approved'
  summary: string
  themes: string[]
  character_seeds: string[]
  scene_seeds: string[]
  prop_seeds: string[]
  created_at: string
  updated_at: string
}

export type StoryMap = {
  characters: StoryMapItem[]
  scenes: StoryMapItem[]
  props: StoryMapItem[]
}

export type StoryMapItem = {
  id: string
  project_id: string
  episode_id: string
  code: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export type StoryboardShot = {
  id: string
  project_id: string
  episode_id: string
  scene_id: string
  code: string
  title: string
  description: string
  prompt: string
  position: number
  duration_ms: number
  created_at: string
  updated_at: string
}

export type Export = {
  id: string
  timeline_id: string
  status: 'queued' | 'rendering' | 'succeeded' | 'failed' | 'canceled'
  format: string
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
  tracks?: SaveTimelineTrackRequest[]
}

export type SaveTimelineTrackRequest = {
  kind: string
  name: string
  position: number
  clips?: SaveTimelineClipRequest[]
}

export type SaveTimelineClipRequest = {
  asset_id?: string
  kind: string
  start_ms: number
  duration_ms: number
  trim_start_ms?: number
}

export type CreateEpisodeRequest = {
  title: string
  number?: number
}
