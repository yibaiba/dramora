# State Management

> How state is managed in this project.

---

## Overview

Use TanStack Query for server state and Zustand for shared local Studio/editor state. Keep form-only state inside components.

---

## State Categories

- Server state: projects, episodes, generation jobs, workflow runs, timelines.
- Shared local state: selected project, local event rail, future playhead/zoom/selection.
- Component state: input fields, temporary hover/open states, and unsaved timeline editor drafts.
- URL state: introduce routing/deep links after page structure stabilizes.

---

## When to Use Global State

Promote to Zustand only when at least two panels need the same state or the state is high-frequency editor state.

---

## Server State

Server state must go through hooks in `src/api/hooks.ts`.

Current query keys:

```ts
['projects']
['episodes', projectId]
['generation-jobs']
['workflow-run', workflowRunId]
['story-sources', episodeId]
['story-analyses', episodeId]
['story-analysis', analysisId]
['approval-gates', episodeId]
['story-map', episodeId]
['storyboard-workspace', episodeId]
['storyboard-shots', episodeId]
['shot-prompt-pack', shotId]
['assets', episodeId]
['timeline', episodeId]
['export', exportId]
```

Mutations invalidate the relevant query key after success.

Agent Board should derive display state from `['generation-jobs']` instead of copying job rows into Zustand. Filter jobs by the selected project or episode in component memoization, then map job statuses to SOP step labels.
Workflow checkpoint observability should stay in server state via `useWorkflowRun(workflowRunId)`; do not mirror checkpoint summaries into Zustand.

Export status should remain server state under `['export', exportId]`. After `Start export`, use the returned export id to poll `useExport(exportId)` while status is `queued` or `rendering`; do not mirror export rows into Zustand or component effects.

Timeline editor draft state should stay component-local until the user explicitly saves through `useSaveEpisodeTimeline`. The canonical timeline remains the `['timeline', episodeId]` query result; when deriving an initial editable draft from server state, use keyed component remounting or explicit user actions instead of synchronously copying query data in an effect.

## Scenario: Studio production readiness and edit mutations

### 1. Scope / Trigger

- Trigger: Studio orchestrates multi-step production state and lets editors save storyboard/prompt changes.
- Applies when adding production buttons, generation queue panels, storyboard editors, or prompt-pack editors.

### 2. Signatures

Required hook boundary:

```ts
useStoryAnalyses(episodeId)
useStoryMap(episodeId)
useSaveCharacterBible()
useEpisodeAssets(episodeId)
useStoryboardShots(episodeId)
useGenerationJobs()
useUpdateStoryboardShot()
useSaveShotPromptPack()
```

Frontend command routes stay in `src/api/client.ts`:

```text
POST /api/v1/story-map-characters/{characterId}/character-bible:save
POST /api/v1/storyboard-shots/{shotId}:update
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:save
```

### 3. Contracts

- Production readiness is derived from query data, not copied into Zustand.
- `GET /episodes/{episodeId}/story-map` may return `200` with empty `characters`, `scenes`, and `props`; that is not ready for asset or storyboard generation.
- Treat story map as ready only when `characters.length + scenes.length + props.length > 0`.
- Seed storyboard actions require both at least one story analysis and a non-empty story map.
- Seed asset actions require a non-empty story map.
- `AssetsGraphPage` may keep in-progress Character Bible form drafts local, but persisted saves must go through `useSaveCharacterBible()` and never through component-local fetch logic.
- Generation queue UI filters `['generation-jobs']` by active episode in component memoization.
- Draft shot fields and prompt text may be component-local until the user presses Save; save actions must call mutation hooks.
- Prompt save may enrich `direct_prompt` with a deterministic Character Bible consistency block derived from matched `reference_assets`, but it must still save through `useSaveShotPromptPack()` and remain visible in the editor state.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| No active episode | Disable backend production commands. |
| No story analysis rows | Disable story-map and storyboard seeding; prompt user to run story analysis first. |
| Empty story map arrays | Show "generate story map", not "ready"; disable asset/storyboard seeding. |
| Character Bible save succeeds | Invalidate `['story-map', episodeId]` and `['storyboard-workspace', episodeId]`. |
| Local/demo shot without backend id | Keep edits local and disable or clearly localize backend-only commands. |
| Shot or prompt save succeeds | Invalidate `['storyboard-shots', episodeId]` or `['shot-prompt-pack', shotId]`. |
| Shot only falls back to "all episode references" instead of matched roles | Do not auto-inject every role into prompt save; require a confident match or a single clear role. |
| Generation command succeeds | Invalidate `['generation-jobs']`. |

### 5. Good/Base/Bad Cases

- Good: `const storyMapReady = characters.length + scenes.length + props.length > 0`.
- Good: `useSaveCharacterBible()` persists role-only Character Bible edits and refreshes both `story-map` and `storyboard-workspace`.
- Base: generation queue panel reads all jobs through `useGenerationJobs()` and filters by `activeEpisode.id`.
- Bad: `Boolean(storyMap)` marks an empty story map as ready and exposes commands that will fail with not found.

### 6. Tests Required

- Frontend lint/build must pass after hook and DTO changes.
- API integration changes require Vite proxy smoke while Go API is running.
- Cross-layer route changes require OpenAPI parse plus GET/POST-only route/client scan.

### 7. Wrong vs Correct

#### Wrong

```ts
const storyMapReady = Boolean(storyMap)
```

#### Correct

```ts
const storyMapReady = Boolean(storyMap) &&
  storyMap.characters.length + storyMap.scenes.length + storyMap.props.length > 0
```

The correct form distinguishes an empty persisted story-map container from a usable production map.

---

## Scenario: Assets Graph Character Bible persistence

### 1. Scope / Trigger

- Trigger: `AssetsGraphPage` lets editors persist character-specific bible data for role consistency.
- Applies when changing Character Bible DTOs, Studio hooks, Assets / Graph editor state, or story-map projections that surface persisted bible data.

### 2. Signatures

Required hook boundary:

```ts
useStoryMap(episodeId)
useStoryboardWorkspace(episodeId)
useSaveCharacterBible()
```

Frontend command route:

```text
POST /api/v1/story-map-characters/{characterId}/character-bible:save
```

### 3. Contracts

- Character Bible persistence is character-only for now; scene and prop nodes stay preview-only unless a new backend contract is added.
- `StoryMapItem.character_bible` is the persisted source of truth for character nodes shown in `AssetsGraphPage`.
- `StoryMapItem.character_bible.reference_assets[]` stores angle-to-locked-asset bindings and should be edited through the same Character Bible draft/save flow.
- Local draft edits may exist while typing, but Save must use `useSaveCharacterBible()` from `src/api/hooks.ts`.
- Successful saves must refresh both `['story-map', episodeId]` and `['storyboard-workspace', episodeId]` so Assets / Graph and Storyboard stay aligned.
- `anchor` is required; UI should block or surface backend validation rather than sending an empty anchor as a silent no-op.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Selected node is not a character | Show preview/read-only guidance; do not expose the Character Bible save mutation. |
| Character Bible anchor is blank | Treat as invalid input and surface the validation error. |
| Reference asset is not locked / not this character node | Surface backend validation; do not fake a local success state. |
| Save succeeds | Re-read story map/workspace projections so persisted bible data appears across pages. |
| Story map read returns character without `character_bible` | Treat as "not authored yet", not as an API failure. |

### 5. Good/Base/Bad Cases

- Good: `AssetsGraphPage` derives initial form values from `node.character_bible ?? defaults` and saves through `useSaveCharacterBible()`.
- Good: angle-to-reference-asset bindings live inside the same Character Bible draft object and persist via `reference_assets`.
- Base: Storyboard workspace reads the updated character projection after save without a special second endpoint.
- Bad: storing persisted Character Bible only in component state or trying to piggyback the edit on `POST /story-map:seed`.

### 6. Tests Required

- Frontend lint/build must pass after DTO/client/hook changes.
- HTTP integration coverage should save Character Bible and verify it is readable from both `GET /story-map` and `GET /storyboard-workspace`.
- OpenAPI/client/types must change in the same slice as the new save route.

---

## Scenario: Storyboard workspace aggregate read model

### 1. Scope / Trigger

- Trigger: Storyboard 页面已经是独立路由，需要一个稳定的工作台主读模型，而不是在页面层手工拼装 story map、assets、approval gates、generation jobs、storyboard shots。
- Applies when changing StoryboardPage data flow, frontend DTO/client/hook types, or the episode-scoped storyboard workspace route.

### 2. Signatures

Required hook boundary:

```ts
useStoryboardWorkspace(episodeId)
useUpdateStoryboardShot()
useSaveShotPromptPack()
useGenerateShotPromptPack()
useStartShotVideoGeneration()
useApproveApprovalGate()
useRequestApprovalChanges()
useResubmitApprovalGate()
```

Frontend command routes stay in `src/api/client.ts`:

```text
GET  /api/v1/episodes/{episodeId}/storyboard-workspace
POST /api/v1/storyboard-shots/{shotId}:update
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:save
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:generate
POST /api/v1/storyboard-shots/{shotId}/videos:generate
POST /api/v1/approval-gates/{gateId}:approve
POST /api/v1/approval-gates/{gateId}:request-changes
POST /api/v1/approval-gates/{gateId}:resubmit
```

### 3. Contracts

- `StoryboardPage` should use `['storyboard-workspace', episodeId]` as the primary read query for workspace data.
- The aggregate response may include an empty `story_map`, zero `storyboard_shots`, zero `approval_gates`, and zero `generation_jobs` while still returning `200`; empty readiness is not an error state.
- `storyboard_workspace.summary.story_map_ready` is the readiness source for storyboard/asset generation controls; do not recompute it from partial page-local data when the aggregate read model is present.
- Per-shot `prompt_pack` inside the aggregate payload is a summary/readiness signal; full prompt editing may still read `['shot-prompt-pack', shotId]` for the selected shot.
- Mutations that change storyboard workspace state must invalidate `['storyboard-workspace', episodeId]` in addition to their resource-specific query keys.
- Shared route/page context such as `selectedProjectId` and `selectedEpisodeId` stays in shared shell state; storyboard workspace payload itself must remain server state in TanStack Query.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| No active episode | Disable workspace reads and backend storyboard actions. |
| Story map has empty arrays | Return workspace `200` with `summary.story_map_ready = false`; do not treat it as an API failure. |
| Prompt pack missing for a shot | Allow `prompt_pack = null` in aggregate payload; selected-shot detail query may still return `404` until generated. |
| Shot/prompt/approval/video mutation succeeds | Invalidate `['storyboard-workspace', episodeId]` plus affected resource query keys. |
| Storyboard page reads aggregate route | Do not simultaneously rebuild the same main workspace state from top-level `useStoryMap`, `useEpisodeAssets`, `useEpisodeApprovalGates`, and `useGenerationJobs` queries. |

### 5. Good/Base/Bad Cases

- Good: `StoryboardPage` reads `useStoryboardWorkspace(activeEpisode?.id)` for its main data and keeps write operations on existing mutation hooks.
- Base: selected-shot prompt editor still uses `useShotPromptPack(selectedShot.id)` for the full prompt-pack payload while the page shell uses the aggregate route.
- Bad: page-level code fans out across 5+ independent queries for the main storyboard workspace after the aggregate route exists.

### 6. Tests Required

- Frontend lint/build must pass after adding workspace DTO/client/hook types.
- API integration changes require Vite proxy smoke while the Go API is running.
- Cross-layer route changes require OpenAPI parse plus GET/POST-only route/client scan.
- Storyboard page behavior should preserve empty-state rendering when the workspace route returns an empty story map and no shots.

### 7. Wrong vs Correct

#### Wrong

```ts
const { data: storyMap } = useStoryMap(activeEpisode?.id)
const { data: assets = [] } = useEpisodeAssets(activeEpisode?.id)
const { data: gates = [] } = useEpisodeApprovalGates(activeEpisode?.id)
const { data: jobs = [] } = useGenerationJobs()
const { data: storyboardShots = [] } = useStoryboardShots(activeEpisode?.id)
```

for the page's main storyboard workspace state.

#### Correct

```ts
const { data: storyboardWorkspace } = useStoryboardWorkspace(activeEpisode?.id)
const shots = storyboardWorkspace?.storyboard_shots ?? []
const gates = storyboardWorkspace?.approval_gates ?? []
```

The correct form makes the route boundary explicit and keeps the page from re-assembling the same workspace contract client-side.

---

## Scenario: Novel source input and multi-agent analysis display

### 1. Scope / Trigger

- Trigger: Studio lets users save novel/story source text and inspect deterministic multi-agent story analysis outputs.
- Applies when adding story source forms, analysis result panels, API DTOs, or story-analysis readiness logic.

### 2. Signatures

Required hook boundary:

```ts
useStorySources(episodeId)
useCreateStorySource(episodeId)
useStoryAnalyses(episodeId)
```

Frontend command routes stay in `src/api/client.ts`:

```text
GET  /api/v1/episodes/{episodeId}/story-sources
POST /api/v1/episodes/{episodeId}/story-sources
```

### 3. Contracts

- Source form state is component-local until submit; canonical source rows come from `['story-sources', episodeId]`.
- `useCreateStorySource` invalidates `['story-sources', episodeId]` after success.
- Analysis display reads `StoryAnalysis.outline` and `StoryAnalysis.agent_outputs`; do not derive fake agent output in components.
- Agent follow-up feedback (`已采纳 / 待跟进`) stays page-local, but may persist to `localStorage` keyed by `episode_id + analysis.id`; do not promote it to Zustand unless multiple pages truly need shared live editing.
- Story source save requires an active episode and non-blank `content_text`; disabled controls must use semantic disabled state.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| No active episode | Disable source save and analysis commands. |
| Blank source text | Disable submit; backend still validates and returns `400 invalid_request`. |
| No analysis rows | Show an empty analysis result state, not fake generated content. |
| Analysis rows exist | Render outline/person/scene/prop lists from API DTO fields. |

### 5. Good/Base/Bad Cases

- Good: save source with `useCreateStorySource`, then start analysis and display `analysis.outline`.
- Good: persist StoryAnalysis feedback in browser storage keyed per analysis so refresh keeps the current follow-up queue without leaking state to other episodes.
- Base: latest source label reads from `sources[0]` because backend returns newest first.
- Bad: storing generated outline only in React state or duplicating API DTO types in components.

### 6. Tests Required

- `cd apps/studio && npm run lint -- --quiet && npm run build`.
- API contract changes require OpenAPI parse and GET/POST-only route/client scan.

### 7. Wrong vs Correct

#### Wrong

```ts
const fakeAgents = ['故事分析', '人物分析']
```

#### Correct

```ts
analysis.agent_outputs.map((agent) => agentRoleLabel(agent.role))
```

The correct form keeps Studio a projection of server state and avoids success-shaped UI when backend artifacts are missing.

---

## Examples

- `apps/studio/src/main.tsx` configures a single `QueryClientProvider` for server state and disables window-focus refetch churn by default.
- `apps/studio/src/state/studioStore.ts` stores only shared local UI state: selected project id and a small local event log.
- `apps/studio/src/App.tsx` keeps temporary prompt/timeline/editor interaction state component-local and reads backend rows through hooks from `src/api/hooks.ts`.

---

## Common Mistakes

- Do not duplicate server responses into Zustand.
- Do not call API client functions directly from multiple panels.
- Do not keep selected project only in component state because Agent Board, Timeline, and Jobs need shared context.
- Do not hard-code Agent Board status once a matching server job exists; derive it from generation job rows.
- Do not copy timeline server state into local editor state with `useEffect` setters; React lint treats that as cascading render work.
- Do not treat an empty story map response as production-ready; check item counts before enabling dependent actions.
