# React Studio UI Map

## Purpose

Define Manmu Studio's first React product shape: pages, layout, Agent Board, Storyboard Kanban, asset review surfaces, and timeline editor.

This is a planning document, not implementation code. It should guide the first frontend scaffold and API DTO contracts.

## Design direction

Manmu Studio is a professional AI video creation workspace, not a generic admin dashboard.

Recommended feel:

- immersive dark workspace for long creative sessions,
- high-contrast status and cost feedback,
- video-first previews,
- structured production progress,
- creator-friendly candidate selection,
- compact inspector panels for prompt/model/asset details.

From `ui-ux-pro-max`:

- Product pattern: video-first creative studio / dashboard.
- Visual style: high-contrast, kinetic, bold, content-dense, but avoid motion overload in the editor itself.
- Typography direction: technical/productive pairing such as Fira Sans + Fira Code or Inter + JetBrains Mono.
- Critical UX rules:
  - show loading/skeleton/progress for operations over 300ms,
  - disable async action buttons while running,
  - use visible focus states and keyboard navigation,
  - no icon-only actions without labels or tooltips,
  - lazy-load heavy editor modules,
  - virtualize large storyboards, asset grids, timeline rows, and job logs.

## Frontend stack recommendation

Recommended MVP stack:

```text
React + TypeScript
Vite or Next.js app route for Studio
TanStack Query for server state
Zustand for local editor/canvas/timeline interaction state
TanStack Router or Next router for deep links
Tailwind + shadcn/ui + Radix primitives
Lucide icons
react-virtuoso or tanstack/virtual for large lists/grids
SSE first for realtime workflow/job events; WebSocket later if bidirectional collaboration is needed
```

Why:

- TanStack Query keeps backend state authoritative and cacheable.
- Zustand is suitable for high-frequency local UI state such as timeline selection, drag state, zoom, viewport, and inspector tabs.
- SSE is simpler than WebSocket for one-way status events from workflow/job execution.
- Heavy modules like timeline, video preview, and prompt diff/editor should be dynamic imports.

## Navigation model

Top-level routes:

```text
/studio
  /projects
  /projects/:projectId
  /projects/:projectId/episodes/:episodeId
  /projects/:projectId/episodes/:episodeId/script
  /projects/:projectId/episodes/:episodeId/assets
  /projects/:projectId/episodes/:episodeId/storyboard
  /projects/:projectId/episodes/:episodeId/agent-board
  /projects/:projectId/episodes/:episodeId/editor
  /projects/:projectId/episodes/:episodeId/jobs
  /projects/:projectId/episodes/:episodeId/exports
  /settings/providers
  /settings/billing-cost
```

Navigation rules:

- Desktop uses a persistent left sidebar and top project/episode switcher.
- Mobile/tablet can collapse sidebar to a drawer, but editor views should recommend desktop for complex editing.
- Every production surface must be deep-linkable:
  - selected shot,
  - selected asset,
  - selected generation job,
  - selected approval gate.
- Back navigation must preserve filters, selected shot, scroll position, and timeline zoom where possible.

## Studio layout shell

Recommended desktop shell:

```text
┌──────────────────────────────────────────────────────────────────┐
│ Top bar: project, episode, save/sync, budget, export, profile    │
├──────────────┬───────────────────────────────────────┬───────────┤
│ Left nav     │ Main workspace                        │ Inspector │
│              │ - script / storyboard / agent board   │           │
│              │ - candidate grid / timeline / preview │           │
├──────────────┴───────────────────────────────────────┴───────────┤
│ Bottom rail: active jobs, cost warnings, errors, approvals        │
└──────────────────────────────────────────────────────────────────┘
```

Panel behavior:

- Left sidebar: project sections and production phases.
- Main workspace: the active creation surface.
- Right inspector: selected story/character/scene/prop/shot/job/asset details.
- Bottom rail: realtime generation queue, budget warnings, blocking errors, pending approvals.
- The inspector and bottom rail must be collapsible to protect screen space.

## Core pages

### 1. Project list

Purpose: create or resume manju projects.

Main UI:

- project cards with cover, latest episode, production status, last edited,
- quick actions: open, duplicate, export package, archive,
- filters: active, waiting approval, exporting, failed jobs.

Empty state:

- "Start from idea / import novel / import script" actions.

### 2. Project overview

Purpose: show the project-level world bible and episode map.

Main UI:

- world bible summary,
- character map preview,
- scene/location map preview,
- episode cards with progress,
- total cost and budget summary,
- latest review issues.

### 3. Episode command center

Purpose: one page that answers "what should I do next?"

Main UI:

- production phase stepper,
- active agent/node,
- pending approval gate,
- latest artifacts,
- blocked jobs,
- cost remaining,
- primary CTA based on current state:
  - analyze story,
  - approve story direction,
  - lock characters,
  - approve storyboard,
  - generate videos,
  - open editor,
  - export.

### 4. Script and story analysis

Purpose: import/create script and review LLM analysis.

Main UI:

- left: story source editor/import panel,
- center: generated outline, episode split, scene breakdown,
- right: relationship graph/inspector,
- approval banner for story direction.

Key interactions:

- compare previous analysis versions,
- accept/reject/change request,
- convert scene breakdown into storyboard.

### 5. Asset library

Purpose: manage reusable Character/Scene/Prop/Keyframe/Video/Audio/Subtitle assets.

Main UI:

- tabs: Characters, Scenes, Props, Keyframes, Video, Audio, Subtitles, Exports,
- candidate grid with lock/regenerate/select,
- version history,
- lineage view: "generated from prompt/provider/input assets",
- usage list: shots/timeline clips referencing this asset.

Required asset card fields:

- preview,
- C/S/P code or asset code,
- status,
- locked badge,
- source provider/model,
- cost,
- used by shots,
- quick actions.

### 6. Character workshop

Purpose: create and lock consistent character references.

Main UI:

- character card form,
- full-body / three-view / expression / pose candidate grids,
- relationship notes,
- wardrobe variants,
- lock gate.

Rules:

- locked character versions are visually distinct.
- locked versions cannot be overwritten; new generation creates candidates or a new version.

### 7. Scene and prop workshop

Purpose: create Scene Map and Prop Map assets.

Scene UI:

- location hierarchy,
- scene cards,
- atmosphere/weather/time/lighting/color fields,
- concept art candidate grid,
- common camera angles/background plate references,
- lock gate.

Prop UI:

- prop cards,
- owner/scene association,
- reference image candidates,
- continuity notes.

### 8. Storyboard Kanban

Purpose: manage shot production from planned shot to approved video clip.

Columns:

```text
planned
prompt_ready
keyframe_generating
keyframe_ready
video_generating
review_needed
approved
blocked
```

Shot card fields:

- shot number,
- scene badge,
- character chips,
- prop chips,
- duration,
- camera movement,
- thumbnail/keyframe/video preview,
- job status,
- cost,
- warning badges: continuity, safety, provider failure.

Interactions:

- drag between allowed columns only when backend state permits,
- generate/regenerate keyframe,
- generate/regenerate video,
- open prompt pack,
- open inspector,
- approve shot,
- send to timeline.

Important rule:

- Kanban status reflects backend `shot.status`, `workflow_node_runs.status`, and `generation_jobs.status`; frontend cannot invent durable statuses.

### 9. Agent Board

Purpose: make multi-agent production understandable and controllable.

Layout:

```text
┌──────────────────────────┬────────────────────────────┐
│ Production DAG / phases  │ Current agent / node detail │
├──────────────────────────┼────────────────────────────┤
│ Agent activity feed      │ Artifacts + approval gates  │
├──────────────────────────┴────────────────────────────┤
│ Cost meter + blocking issues + retry/cancel controls   │
└────────────────────────────────────────────────────────┘
```

Must show:

- current workflow status,
- current node/agent,
- upstream dependencies,
- pending approvals,
- produced artifacts,
- errors and retryability,
- budget consumed/reserved/remaining,
- provider/model used,
- trace link to prompt render and generation job.

Agent states:

- queued,
- running,
- tool_calling,
- waiting_job,
- waiting_approval,
- succeeded,
- failed,
- canceled.

Interactions:

- approve/reject/request changes,
- retry failed agent/job,
- cancel workflow/node/job,
- switch model/provider for retry,
- view prompt and parameters,
- view cost breakdown.

### 10. Generation jobs page

Purpose: operational visibility for all async work.

Main UI:

- table of jobs with filters:
  - task type,
  - status,
  - provider,
  - model,
  - retryable,
  - cost range,
  - related shot/asset.
- expandable rows for attempts and provider events,
- bulk cancel for safe statuses,
- retry as new candidate.

Accessibility/performance:

- virtualized table,
- status is shown with text + icon + color,
- error row uses `role="alert"` or equivalent announcement in focused contexts.

### 11. Timeline editor

Purpose: complete the manju video inside Manmu.

Detailed technical selection: see `timeline-editor-tech-selection.md`.

MVP track types:

- video,
- image/keyframe,
- audio/TTS,
- subtitles/text,
- effect/transition.

Main zones:

```text
┌──────────────────────────────┬──────────────────────────┐
│ Preview player               │ Clip / shot inspector    │
├──────────────────────────────┴──────────────────────────┤
│ Tool bar: select, split, trim, transition, captions      │
├──────────────────────────────────────────────────────────┤
│ Timeline ruler                                           │
│ Track: video clips                                       │
│ Track: keyframes/images                                  │
│ Track: audio / TTS                                       │
│ Track: subtitles                                         │
└──────────────────────────────────────────────────────────┘
```

Required operations:

- trim,
- split,
- move,
- delete,
- duplicate,
- reorder,
- snap to shot boundaries,
- fade/crossfade/dissolve/slide transition,
- edit subtitle text/time/style preset,
- regenerate selected shot clip,
- replace clip while preserving timeline placement,
- preview selected range,
- export MP4 through server job.

Timeline state rules:

- Persist canonical `timeline` JSON/backend rows.
- Use local state only for transient selection, drag, zoom, and playhead.
- Use optimistic concurrency with `version_no` or `updated_at`.
- If export starts from a stale timeline version, show a conflict and require refresh/retry.

### 12. Export page

Purpose: render and download final deliverables.

Main UI:

- export presets: MP4 1080p, MP4 720p, storyboard PDF, asset package ZIP,
- render queue status,
- output asset preview/download,
- export error with recovery action,
- cost/time estimate.

## Component map

Shared components:

- `StudioShell`
- `ProjectSwitcher`
- `EpisodeSwitcher`
- `BudgetMeter`
- `StatusBadge`
- `ProgressStepper`
- `ApprovalGateCard`
- `AgentRunCard`
- `WorkflowGraphPreview`
- `GenerationJobRow`
- `CandidateGrid`
- `AssetCard`
- `LineagePopover`
- `ShotCard`
- `StoryboardKanban`
- `PromptPreview`
- `ProviderModelPicker`
- `TimelineEditor`
- `PreviewPlayer`
- `ClipInspector`
- `BottomJobRail`
- `ReviewIssueList`

Design rules:

- Use Lucide/SVG icons only; no emoji structural icons.
- Every icon-only action needs `aria-label` and tooltip.
- Buttons that trigger generation/export must show loading and be disabled while submitting.
- Destructive actions such as cancel/delete must be separated and confirmed.
- Status colors must include text labels; never rely on color alone.

## Frontend state categories

| State type | Owner | Examples |
| --- | --- | --- |
| Server state | TanStack Query | projects, episodes, assets, shots, workflow runs, jobs, approvals |
| Realtime events | SSE subscription + query cache updates | workflow status, generation progress, cost warnings |
| URL state | router query/path | selected episode, selected shot, active tab, filters |
| Local UI state | component state | open panels, active modal, hovered clip |
| High-frequency editor state | Zustand | timeline playhead, zoom, drag operation, selected clips, viewport |

Rules:

- Backend status is authoritative.
- React state must not be the only place where workflow/node/job progress exists.
- Avoid duplicating derived status logic across pages; create a shared status mapping layer.
- Validate API payloads at boundaries if generated types are not available.

## API DTOs needed by UI

Minimum DTOs:

```text
ProjectSummary
EpisodeSummary
ProductionStatusSummary
WorkflowRunDTO
WorkflowNodeRunDTO
AgentRunDTO
ApprovalGateDTO
GenerationJobDTO
GenerationJobAttemptDTO
CostBudgetDTO
CostReservationDTO
AssetSummaryDTO
AssetDetailDTO
CharacterVersionDTO
SceneVersionDTO
PropVersionDTO
ShotCardDTO
PromptPackDTO
TimelineDTO
TimelineClipDTO
ReviewIssueDTO
ExportDTO
RealtimeEventDTO
```

Cross-layer contract:

- API returns stable enum strings.
- API timestamps are ISO-8601.
- Progress is `0..100` or null.
- Money uses integer minor units plus currency.
- Asset preview URLs are short-lived or public-read safe; raw object storage credentials never reach frontend.
- Prompt and provider params can be viewed by authorized users but should hide secrets.

## Realtime event handling

SSE event categories from backend:

- `workflow.status_changed`
- `node.status_changed`
- `agent.output_created`
- `approval.requested`
- `generation.progress`
- `generation.completed`
- `generation.failed`
- `cost.warning`
- `review.issue_created`

UI behavior:

- Update query cache for visible objects.
- Show toast only for user-actionable or blocking events.
- Use bottom rail for continuous job progress to avoid noisy toast spam.
- Keep event history visible in Agent Board and job detail drawer.

## Responsive strategy

Desktop first for full production:

- 1440px+: full sidebar + main + inspector + bottom rail.
- 1024px-1439px: collapsible inspector and bottom rail.
- 768px-1023px: split pages; editor remains usable but less dense.
- 375px-767px: review/approval/job monitoring only; complex timeline editing should be read-only or simplified.

Mobile must still support:

- review generated candidates,
- approve/reject gates,
- monitor jobs,
- comment/request changes,
- download/export result.

## Performance constraints

- Lazy-load `TimelineEditor`, `PreviewPlayer`, heavy prompt editor/diff, graph visualization, and waveform components.
- Virtualize asset grids, storyboard cards, job tables, activity feeds, and prompt logs.
- Use thumbnails and poster frames in lists; load full video only on preview/open.
- Avoid rerendering the entire timeline on playhead ticks.
- Keep timeline drag operations local until commit.
- Use skeleton loading for page sections, not full blank screens.
- Store width/height/aspect ratio for media previews to prevent layout shift.

## Accessibility and interaction constraints

- Keyboard navigation for:
  - left nav,
  - Kanban columns/cards,
  - candidate grid,
  - timeline clip selection,
  - modal/dialog controls.
- Provide non-drag alternatives:
  - move shot status via menu,
  - move clip via numeric fields or buttons,
  - reorder via keyboard action.
- Focus management:
  - route changes focus main heading,
  - dialogs trap focus,
  - closing drawer returns focus to trigger.
- Error messages:
  - show near failed action,
  - include recovery path,
  - announce with `aria-live`/`role="alert"` where appropriate.
- Motion:
  - respect `prefers-reduced-motion`,
  - keep microinteractions around 150-300ms,
  - avoid decorative motion during editing.

## MVP implementation order

1. StudioShell + routing + project/episode switcher.
2. Episode command center with production status summary.
3. Asset library + candidate grid + lock/regenerate actions.
4. Storyboard Kanban with shot inspector.
5. Agent Board connected to workflow/node/agent/job DTOs.
6. Generation jobs table and bottom job rail.
7. Timeline editor MVP: preview, tracks, trim/split/move/delete, subtitles/TTS clips.
8. Export page and export job progress.

## Open decisions

- Studio app shell: Vite React SPA or Next.js app route.
- Timeline foundation: build narrow internal timeline first, adapt Twick/FreeCut ideas, or evaluate a reusable editor SDK.
- Graph visualization: custom read-only graph first or React Flow/xyflow.
- Realtime transport: SSE first or WebSocket first.
- Design tokens: dark-first only for Studio or light/dark from day one.
