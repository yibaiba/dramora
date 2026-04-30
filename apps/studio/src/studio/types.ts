export type StudioShot = {
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
  latestGenerationJobId?: string
}

export type InspectorTab = 'details' | 'prompt' | 'references' | 'notes'
export type ViewMode = 'grid' | 'compact'
export type FlowStepState = 'done' | 'active' | 'waiting'

export type ShotDraft = {
  description: string
  durationMS: number
  title: string
}
