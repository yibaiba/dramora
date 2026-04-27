# State Management

> How state is managed in this project.

---

## Overview

Use TanStack Query for server state and Zustand for shared local Studio/editor state. Keep form-only state inside components.

---

## State Categories

- Server state: projects, episodes, generation jobs, workflow runs, timelines.
- Shared local state: selected project, local event rail, future playhead/zoom/selection.
- Component state: input fields, temporary hover/open states.
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
['story-map', episodeId]
['storyboard-shots', episodeId]
['shot-prompt-pack', shotId]
['assets', episodeId]
['timeline', episodeId]
['export', exportId]
```

Mutations invalidate the relevant query key after success.

Agent Board should derive display state from `['generation-jobs']` instead of copying job rows into Zustand. Filter jobs by the selected project or episode in component memoization, then map job statuses to SOP step labels.

---

## Common Mistakes

- Do not duplicate server responses into Zustand.
- Do not call API client functions directly from multiple panels.
- Do not keep selected project only in component state because Agent Board, Timeline, and Jobs need shared context.
- Do not hard-code Agent Board status once a matching server job exists; derive it from generation job rows.
