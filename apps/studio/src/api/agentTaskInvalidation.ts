import type { QueryClient, QueryKey } from '@tanstack/react-query'

import type { AgentTask } from '../state/agentTaskCenterStore'

function addEpisodeScoped(keys: QueryKey[], episodeId: string, ...prefixes: string[]) {
  for (const prefix of prefixes) {
    keys.push([prefix, episodeId])
  }
}

function toolDrivenKeys(task: AgentTask): QueryKey[] {
  const toolNames = task.events
    .flatMap((event) => {
      if (event.tool_name) return [event.tool_name]
      if (event.tool_calls?.length) return event.tool_calls.map((tool) => tool.name)
      return []
    })
    .map((name) => name.toLowerCase())

  if (!task.episodeId || toolNames.length === 0) return []

  const keys: QueryKey[] = []
  for (const toolName of toolNames) {
    if (toolName.includes('story') && toolName.includes('analysis')) {
      addEpisodeScoped(keys, task.episodeId, 'story-analyses', 'storyboard-workspace')
    }
    if (toolName.includes('story_map') || toolName.includes('character') || toolName.includes('scene') || toolName.includes('prop')) {
      addEpisodeScoped(keys, task.episodeId, 'story-map', 'storyboard-workspace')
    }
    if (toolName.includes('storyboard') || toolName.includes('prompt')) {
      addEpisodeScoped(keys, task.episodeId, 'storyboard-shots', 'storyboard-workspace')
    }
    if (toolName.includes('image') || toolName.includes('asset')) {
      addEpisodeScoped(keys, task.episodeId, 'assets', 'storyboard-workspace')
      keys.push(['generation-jobs'])
    }
    if (toolName.includes('video')) {
      addEpisodeScoped(keys, task.episodeId, 'storyboard-workspace', 'timeline')
      keys.push(['generation-jobs'])
    }
    if (toolName.includes('approval')) {
      addEpisodeScoped(keys, task.episodeId, 'approval-gates', 'storyboard-workspace')
    }
  }
  return keys
}

function roleDrivenKeys(task: AgentTask): QueryKey[] {
  if (!task.episodeId) return []

  const keys: QueryKey[] = []
  switch (task.role) {
    case 'story_analyst':
    case 'outline_planner':
      addEpisodeScoped(keys, task.episodeId, 'story-analyses', 'storyboard-workspace')
      break
    case 'character_analyst':
    case 'scene_analyst':
    case 'prop_analyst':
      addEpisodeScoped(keys, task.episodeId, 'story-map', 'storyboard-workspace')
      break
    case 'screenwriter':
    case 'director':
    case 'cinematographer':
    case 'voice_subtitle':
      addEpisodeScoped(keys, task.episodeId, 'storyboard-shots', 'storyboard-workspace', 'approval-gates')
      break
    default:
      break
  }
  return keys
}

export function resolveAgentTaskInvalidationKeys(task: AgentTask): QueryKey[] {
  if (task.status !== 'done' || !task.episodeId) return []

  const keys = [...roleDrivenKeys(task), ...toolDrivenKeys(task)]
  const deduped = new Map<string, QueryKey>()
  for (const key of keys) {
    deduped.set(JSON.stringify(key), key)
  }
  return [...deduped.values()]
}

export async function invalidateAgentTaskQueries(queryClient: QueryClient, task: AgentTask): Promise<void> {
  const keys = resolveAgentTaskInvalidationKeys(task)
  for (const queryKey of keys) {
    await queryClient.invalidateQueries({ queryKey })
  }
}
