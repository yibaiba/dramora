# Hook Guidelines

> How hooks are used in this project.

---

## Overview

Hooks are the only frontend entrypoint for server state. Components call hooks from
`src/api/hooks.ts`; hooks call typed client functions from `src/api/client.ts`.

---

## Custom Hook Patterns

- Name query hooks as `use<NounPlural>` for lists and `use<Noun>` for single resources.
- Name mutation hooks as `use<Action><Resource>`, for example `useCreateProject` or `useSaveEpisodeTimeline`.
- Keep API URL construction in `client.ts`, not inside components or hooks.
- Return TanStack Query results directly unless the UI needs a narrow view model.

---

## Data Fetching

- Use TanStack Query for all backend server state.
- Use `enabled` for queries that require selected ids.
- Mutations must invalidate or update every affected query key after success.
- Do not mirror server response data into Zustand; use Zustand only for shared local Studio/editor state.
- Frontend API integration uses only `GET` for reads and `POST` for writes/actions; do not set `PUT`, `PATCH`, or `DELETE` in `client.ts`.

---

## Naming Conventions

- Hook names must start with `use`.
- Hook files stay in `src/api/hooks.ts` until the API surface grows enough to split by resource.
- Type request/response payloads in `src/api/types.ts`; do not inline API payload shapes in components.

---

## Common Mistakes

- Do not call `fetch` directly from React components.
- Do not duplicate route strings across multiple panels.
- Do not forget invalidation after mutations that change projects, episodes, jobs, or timelines.
- Do not mirror REST verb semantics in Studio. Use command-style `POST` routes such as `/assets/{assetId}:lock` or `/timeline-clips/{clipId}:remove`.
