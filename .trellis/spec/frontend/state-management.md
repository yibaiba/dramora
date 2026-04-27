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

---

## Common Mistakes

- Do not duplicate server responses into Zustand.
- Do not call API client functions directly from multiple panels.
- Do not keep selected project only in component state because Agent Board, Timeline, and Jobs need shared context.
- Do not hard-code Agent Board status once a matching server job exists; derive it from generation job rows.
- Do not copy timeline server state into local editor state with `useEffect` setters; React lint treats that as cascading render work.
