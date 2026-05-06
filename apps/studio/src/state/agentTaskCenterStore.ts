import { create } from 'zustand'

import type { AgentStreamDoneFrame } from '../api/agentStream'
import type { AgentStreamEvent } from '../api/types'

export type AgentTaskStatus = 'running' | 'done' | 'error' | 'cancelled'

export type AgentTask = {
  id: string
  role: string
  sourceText: string
  contextKeys: string[]
  episodeId?: string
  createdAt: string
  updatedAt: string
  status: AgentTaskStatus
  streamedText: string
  runId?: string
  lastEventSequence?: number
  doneFrame?: AgentStreamDoneFrame
  errorMessage?: string
  events: AgentStreamEvent[]
}

type StartTaskInput = {
  role: string
  sourceText: string
  contextKeys: string[]
  episodeId?: string
}

type AgentTaskCenterState = {
  tasks: AgentTask[]
  notificationOpen: boolean
  panelExpanded: boolean
  selectedTaskId: string | null
  restoredTaskCount: number
  startTask: (input: StartTaskInput) => string
  attachRun: (taskId: string, runId: string) => void
  appendEvent: (taskId: string, event: AgentStreamEvent) => void
  resumeTask: (taskId: string) => void
  completeTask: (taskId: string, frame: AgentStreamDoneFrame) => void
  failTask: (taskId: string, message: string) => void
  cancelTask: (taskId: string) => void
  removeTask: (taskId: string) => void
  clearCompleted: () => void
  setNotificationOpen: (open: boolean) => void
  setPanelExpanded: (expanded: boolean) => void
  selectTask: (taskId: string | null) => void
  dismissRestoreNotice: () => void
}

const MAX_TASKS = 24
const MAX_EVENTS_PER_TASK = 80
const TASK_CENTER_STORAGE_KEY = 'dramora.agent-task-center'

type PersistedTaskCenterState = Pick<
  AgentTaskCenterState,
  'tasks' | 'notificationOpen' | 'panelExpanded' | 'selectedTaskId'
>

function nowIso(): string {
  return new Date().toISOString()
}

function createTaskId(): string {
  return `agent-task-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function updateTask(tasks: AgentTask[], taskId: string, updater: (task: AgentTask) => AgentTask): AgentTask[] {
  return tasks.map((task) => (task.id === taskId ? updater(task) : task))
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object'
}

function sanitizeEvent(value: unknown): AgentStreamEvent | null {
  if (!isRecord(value) || typeof value.type !== 'string') return null
  return {
    type: value.type as AgentStreamEvent['type'],
    run_id: typeof value.run_id === 'string' ? value.run_id : undefined,
    sequence: typeof value.sequence === 'number' ? value.sequence : undefined,
    replay: typeof value.replay === 'boolean' ? value.replay : undefined,
    occurred_at: typeof value.occurred_at === 'string' ? value.occurred_at : undefined,
    role: typeof value.role === 'string' ? value.role : undefined,
    content: typeof value.content === 'string' ? value.content : undefined,
    output: typeof value.output === 'string' ? value.output : undefined,
    error: typeof value.error === 'string' ? value.error : undefined,
    tool_name: typeof value.tool_name === 'string' ? value.tool_name : undefined,
    highlights: Array.isArray(value.highlights)
      ? value.highlights.filter((entry): entry is string => typeof entry === 'string')
      : undefined,
    token_count: typeof value.token_count === 'number' ? value.token_count : undefined,
    duration_ms: typeof value.duration_ms === 'number' ? value.duration_ms : undefined,
    tool_calls: Array.isArray(value.tool_calls)
      ? value.tool_calls
          .filter(
            (tool): tool is { id: string; name: string; arguments: string } =>
              isRecord(tool) &&
              typeof tool.id === 'string' &&
              typeof tool.name === 'string' &&
              typeof tool.arguments === 'string',
          )
          .map((tool) => ({ id: tool.id, name: tool.name, arguments: tool.arguments }))
      : undefined,
  }
}

function sanitizeDoneFrame(value: unknown): AgentStreamDoneFrame | undefined {
  if (!isRecord(value)) return undefined
  if (
    typeof value.role !== 'string' ||
    typeof value.output !== 'string' ||
    !Array.isArray(value.highlights) ||
    typeof value.token_count !== 'number' ||
    typeof value.duration_ms !== 'number'
  ) {
    return undefined
  }
  return {
    role: value.role,
    output: value.output,
    highlights: value.highlights.filter((entry): entry is string => typeof entry === 'string'),
    token_count: value.token_count,
    duration_ms: value.duration_ms,
  }
}

function reviveTask(task: AgentTask): AgentTask {
  if (task.status !== 'running') return task
  return {
    ...task,
    status: 'cancelled',
    updatedAt: nowIso(),
    errorMessage: '页面刷新或连接中断后，运行中的任务已转为本地回放状态，请重新运行。',
    events: [
      ...task.events,
      {
        type: 'CANCELLED' as const,
        role: task.role,
        error: '页面刷新或连接中断，已从本地缓存恢复。',
      },
    ].slice(-MAX_EVENTS_PER_TASK),
  }
}

function sanitizeTask(value: unknown): AgentTask | null {
  if (!isRecord(value)) return null
  if (
    typeof value.id !== 'string' ||
    typeof value.role !== 'string' ||
    typeof value.sourceText !== 'string' ||
    !Array.isArray(value.contextKeys) ||
    typeof value.createdAt !== 'string' ||
    typeof value.updatedAt !== 'string' ||
    typeof value.streamedText !== 'string' ||
    !Array.isArray(value.events)
  ) {
    return null
  }
  if (
    value.status !== 'running' &&
    value.status !== 'done' &&
    value.status !== 'error' &&
    value.status !== 'cancelled'
  ) {
    return null
  }
  const events = value.events
    .map((event) => sanitizeEvent(event))
    .filter((event): event is AgentStreamEvent => Boolean(event))
    .slice(-MAX_EVENTS_PER_TASK)
  return reviveTask({
    id: value.id,
    role: value.role,
    sourceText: value.sourceText,
    contextKeys: value.contextKeys.filter((entry): entry is string => typeof entry === 'string'),
    episodeId: typeof value.episodeId === 'string' ? value.episodeId : undefined,
    createdAt: value.createdAt,
    updatedAt: value.updatedAt,
    status: value.status,
    streamedText: value.streamedText,
    doneFrame: sanitizeDoneFrame(value.doneFrame),
    errorMessage: typeof value.errorMessage === 'string' ? value.errorMessage : undefined,
    runId: typeof value.runId === 'string' ? value.runId : undefined,
    lastEventSequence: typeof value.lastEventSequence === 'number' ? value.lastEventSequence : undefined,
    events,
  })
}

function readStoredTaskCenterState(): PersistedTaskCenterState {
  if (typeof window === 'undefined') {
    return {
      tasks: [],
      notificationOpen: false,
      panelExpanded: true,
      selectedTaskId: null,
    }
  }
  const raw = window.localStorage.getItem(TASK_CENTER_STORAGE_KEY)
  if (!raw) {
    return {
      tasks: [],
      notificationOpen: false,
      panelExpanded: true,
      selectedTaskId: null,
    }
  }
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!isRecord(parsed)) {
      return {
        tasks: [],
        notificationOpen: false,
        panelExpanded: true,
        selectedTaskId: null,
      }
    }
    const tasks = Array.isArray(parsed.tasks)
      ? parsed.tasks
          .map((task) => sanitizeTask(task))
          .filter((task): task is AgentTask => Boolean(task))
          .slice(0, MAX_TASKS)
      : []
    return {
      tasks,
      notificationOpen: typeof parsed.notificationOpen === 'boolean' ? parsed.notificationOpen : tasks.length > 0,
      panelExpanded: typeof parsed.panelExpanded === 'boolean' ? parsed.panelExpanded : true,
      selectedTaskId: typeof parsed.selectedTaskId === 'string' ? parsed.selectedTaskId : tasks[0]?.id ?? null,
    }
  } catch {
    window.localStorage.removeItem(TASK_CENTER_STORAGE_KEY)
    return {
      tasks: [],
      notificationOpen: false,
      panelExpanded: true,
      selectedTaskId: null,
    }
  }
}

function writeStoredTaskCenterState(state: PersistedTaskCenterState) {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.setItem(TASK_CENTER_STORAGE_KEY, JSON.stringify(state))
}

const storedState = readStoredTaskCenterState()

export const useAgentTaskCenterStore = create<AgentTaskCenterState>((set) => ({
  tasks: storedState.tasks,
  notificationOpen: storedState.notificationOpen,
  panelExpanded: storedState.panelExpanded,
  selectedTaskId: storedState.selectedTaskId,
  restoredTaskCount: storedState.tasks.length,
  startTask: (input) => {
    const id = createTaskId()
    const task: AgentTask = {
      id,
      role: input.role,
      sourceText: input.sourceText,
      contextKeys: input.contextKeys,
      episodeId: input.episodeId,
      createdAt: nowIso(),
      updatedAt: nowIso(),
      status: 'running',
      streamedText: '',
      events: [],
    }
    set((state) => ({
      tasks: [task, ...state.tasks].slice(0, MAX_TASKS),
      notificationOpen: true,
      panelExpanded: true,
      selectedTaskId: id,
    }))
    return id
  },
  attachRun: (taskId, runId) =>
    set((state) => ({
      tasks: updateTask(state.tasks, taskId, (task) => ({
        ...task,
        runId,
        updatedAt: nowIso(),
      })),
    })),
  appendEvent: (taskId, event) =>
    set((state) => ({
      tasks: updateTask(state.tasks, taskId, (task) => ({
        ...task,
        updatedAt: nowIso(),
        runId: event.run_id ?? task.runId,
        lastEventSequence: event.sequence ?? task.lastEventSequence,
        streamedText:
          event.type === 'CONTENT' && event.content ? `${task.streamedText}${event.content}` : task.streamedText,
        events: [...task.events, event].slice(-MAX_EVENTS_PER_TASK),
      })),
    })),
  resumeTask: (taskId) =>
    set((state) => ({
      tasks: updateTask(state.tasks, taskId, (task) => ({
        ...task,
        status: 'running',
        updatedAt: nowIso(),
        errorMessage: undefined,
      })),
    })),
  completeTask: (taskId, frame) =>
    set((state) => ({
      tasks: updateTask(state.tasks, taskId, (task) => ({
        ...task,
        status: 'done',
        updatedAt: nowIso(),
        doneFrame: frame,
        streamedText: frame.output || task.streamedText,
      })),
    })),
  failTask: (taskId, message) =>
    set((state) => ({
      tasks: updateTask(state.tasks, taskId, (task) => ({
        ...task,
        status: 'error',
        updatedAt: nowIso(),
        errorMessage: message,
      })),
    })),
  cancelTask: (taskId) =>
    set((state) => ({
      tasks: updateTask(state.tasks, taskId, (task) => ({
        ...task,
        status: 'cancelled',
        updatedAt: nowIso(),
        errorMessage: task.errorMessage ?? '已手动中止',
      })),
    })),
  removeTask: (taskId) =>
    set((state) => {
      const tasks = state.tasks.filter((task) => task.id !== taskId)
      return {
        tasks,
        selectedTaskId:
          state.selectedTaskId === taskId ? tasks[0]?.id ?? null : state.selectedTaskId,
      }
    }),
  clearCompleted: () =>
    set((state) => {
      const tasks = state.tasks.filter((task) => task.status === 'running')
      return {
        tasks,
        selectedTaskId: tasks[0]?.id ?? null,
      }
    }),
  setNotificationOpen: (open) =>
    set((state) => ({
      notificationOpen: open,
      panelExpanded: open ? state.panelExpanded : false,
    })),
  setPanelExpanded: (expanded) =>
    set((state) => ({
      panelExpanded: expanded,
      notificationOpen: expanded ? true : state.notificationOpen,
    })),
  selectTask: (taskId) => set({ selectedTaskId: taskId }),
  dismissRestoreNotice: () => set({ restoredTaskCount: 0 }),
}))

if (typeof window !== 'undefined') {
  useAgentTaskCenterStore.subscribe((state) => {
    writeStoredTaskCenterState({
      tasks: state.tasks,
      notificationOpen: state.notificationOpen,
      panelExpanded: state.panelExpanded,
      selectedTaskId: state.selectedTaskId,
    })
  })
}
