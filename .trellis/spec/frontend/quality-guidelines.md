# Quality Guidelines

> Code quality standards for frontend development.

---

## Overview

Frontend changes must pass lint and production build. UI should follow the Manmu dark professional Studio design system.

---

## Forbidden Patterns

- No `any`.
- No emoji structural icons.
- No direct `fetch` calls inside UI components.
- No removed focus states.
- No raw API DTO duplication inside page components.

---

## Required Patterns

### Scenario: React Studio scaffold validation

#### 1. Scope / Trigger

- Trigger: UI pages, API client, design system, or frontend package dependencies change.

#### 2. Signatures

```bash
cd apps/studio
npm run lint
npm run build
```

#### 3. Contracts

- Vite app lives under `apps/studio`.
- `apps/studio/go.mod` must remain so root Go validation ignores frontend `node_modules`.
- API client and DTO types are centralized under `src/api`.
- Shared local state is centralized under `src/state`.
- Styles must support responsive layouts and reduced motion.

#### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| TypeScript build fails | Fix before claiming completion. |
| ESLint fails | Fix before claiming completion. |
| API route contract changes | Update DTOs/hooks and smoke through Vite proxy. |
| UI adds icon-only button | Add accessible label. |
| Root Go validation starts scanning `node_modules` | Restore/keep `apps/studio/go.mod`. |

#### 5. Good/Base/Bad Cases

- Good: route data through TanStack Query hooks and typed DTOs.
- Base: placeholder panels are acceptable if they reflect planned backend contracts.
- Bad: ship UI that only works on wide desktop or has no keyboard focus state.

#### 6. Tests Required

- `npm run lint`
- `npm run build`
- Smoke route/API proxy for API integration changes.
- Add component tests as interactions become more than simple rendering.

#### 7. Wrong vs Correct

##### Wrong

```tsx
<button>+</button>
```

with no accessible label or visible context.

##### Correct

```tsx
<button type="submit">
  <Plus aria-hidden="true" />
  Create
</button>
```

---

## Testing Requirements

- Build and lint are required for every frontend change.
- Add tests when behavior includes branching, validation, or error states.

---

## Code Review Checklist

- API DTOs align with `api/openapi.yaml`.
- Query invalidation is correct after mutations.
- UI respects accessibility, reduced motion, and responsive breakpoints.
