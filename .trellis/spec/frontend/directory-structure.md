# Directory Structure

> How frontend code is organized in this project.

---

## Overview

The Studio frontend is a Vite React TypeScript app at `apps/studio`. It is a professional dark AI video creation workspace, not a generic admin dashboard.

---

## Directory Layout

```text
apps/studio/
├── go.mod              # nested empty Go module to stop root go ./... from scanning node_modules
├── package.json
├── vite.config.ts
└── src/
    ├── api/
    │   ├── client.ts       # fetch wrappers
    │   ├── hooks.ts        # TanStack Query hooks
    │   └── types.ts        # API DTO types
    ├── state/
    │   └── studioStore.ts  # Zustand local Studio state
    ├── App.tsx             # first Studio shell
    ├── index.css           # Tailwind import + design tokens
    └── main.tsx            # QueryClientProvider bootstrap
```

---

## Module Organization

### Scenario: Studio scaffold

#### 1. Scope / Trigger

- Trigger: React Studio scaffold established app, API client, state, and design system contracts.

#### 2. Signatures

Commands:

```bash
cd apps/studio
npm run dev
npm run lint
npm run build
```

Environment:

```bash
VITE_MANMU_API_BASE_URL=http://127.0.0.1:8080
```

#### 3. Contracts

- Vite proxy maps `/api`, `/healthz`, and `/readyz` to the Go API during local dev.
- `apps/studio/go.mod` is intentional: it prevents root `go test ./...` from traversing `node_modules`.
- `src/api/types.ts` mirrors `api/openapi.yaml` route DTOs.
- `src/api/client.ts` owns raw fetch calls.
- `src/api/hooks.ts` owns TanStack Query keys and cache invalidation.
- `src/state/studioStore.ts` owns local UI/editor state only.
- Components should not call `fetch` directly.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| API route DTO changes | Update OpenAPI, `src/api/types.ts`, and hooks/client together. |
| API mutation needed | Use `POST`; Studio-to-API integration does not use `PUT`, `PATCH`, or `DELETE`. |
| Server-state fetch needed | Add function in `client.ts`, then hook in `hooks.ts`. |
| Local UI state needed | Add to Zustand store if shared across panels; otherwise keep component-local. |
| Vite proxy unavailable | Use `VITE_MANMU_API_BASE_URL`. |
| Root `go test ./...` enters `node_modules` | Keep `apps/studio/go.mod`; do not delete it unless Go validation command changes. |
| New workflow mutation | Invalidate `['generation-jobs']` so Jobs rail reflects queued work. |

#### 5. Good/Base/Bad Cases

- Good: component calls `useProjects()` and renders typed `Project`.
- Base: page-level placeholders for Agent Board/Timeline while backend rows are not yet interactive.
- Bad: component calls `fetch('/api/v1/projects')` directly and duplicates DTO parsing.

#### 6. Tests Required

- `npm run lint`.
- `npm run build`.
- Smoke-check Vite proxy for API integration changes.
- Add component tests when UI behavior becomes conditional or complex.

#### 7. Wrong vs Correct

##### Wrong

```tsx
const response = await fetch('/api/v1/projects')
```

inside a panel component.

##### Correct

```tsx
const { data: projects = [] } = useProjects()
```

Components consume hooks; hooks consume the API client.

---

## Naming Conventions

- React components use PascalCase.
- API files use lowercase names under `src/api`.
- Zustand stores use `use<Name>Store`.
- CSS class names use readable kebab-case for shell-level layout.

---

## Examples

- `src/api/client.ts`: API boundary.
- `src/api/hooks.ts`: server-state boundary.
- `src/state/studioStore.ts`: local UI state.
