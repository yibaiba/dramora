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
['story-analyses', episodeId]
['story-analysis', analysisId]
['approval-gates', episodeId]
['story-map', episodeId]
['storyboard-shots', episodeId]
['shot-prompt-pack', shotId]
['assets', episodeId]
['timeline', episodeId]
['export', exportId]
```

Mutations invalidate the relevant query key after success.

Agent Board should derive display state from `['generation-jobs']` instead of copying job rows into Zustand. Filter jobs by the selected project or episode in component memoization, then map job statuses to SOP step labels.

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
useEpisodeAssets(episodeId)
useStoryboardShots(episodeId)
useGenerationJobs()
useUpdateStoryboardShot()
useSaveShotPromptPack()
```

Frontend command routes stay in `src/api/client.ts`:

```text
POST /api/v1/storyboard-shots/{shotId}:update
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:save
```

### 3. Contracts

- Production readiness is derived from query data, not copied into Zustand.
- `GET /episodes/{episodeId}/story-map` may return `200` with empty `characters`, `scenes`, and `props`; that is not ready for asset or storyboard generation.
- Treat story map as ready only when `characters.length + scenes.length + props.length > 0`.
- Seed storyboard actions require both at least one story analysis and a non-empty story map.
- Seed asset actions require a non-empty story map.
- Generation queue UI filters `['generation-jobs']` by active episode in component memoization.
- Draft shot fields and prompt text may be component-local until the user presses Save; save actions must call mutation hooks.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| No active episode | Disable backend production commands. |
| No story analysis rows | Disable story-map and storyboard seeding; prompt user to run story analysis first. |
| Empty story map arrays | Show "generate story map", not "ready"; disable asset/storyboard seeding. |
| Local/demo shot without backend id | Keep edits local and disable or clearly localize backend-only commands. |
| Shot or prompt save succeeds | Invalidate `['storyboard-shots', episodeId]` or `['shot-prompt-pack', shotId]`. |
| Generation command succeeds | Invalidate `['generation-jobs']`. |

### 5. Good/Base/Bad Cases

- Good: `const storyMapReady = characters.length + scenes.length + props.length > 0`.
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
