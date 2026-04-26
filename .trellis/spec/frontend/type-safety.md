# Type Safety

> Type safety patterns in this project.

---

## Overview

Use TypeScript with explicit API DTO types. `verbatimModuleSyntax` requires type-only imports for types.

---

## Type Organization

- API DTOs live in `apps/studio/src/api/types.ts`.
- Component-only prop types may live near the component.
- Types must mirror `api/openapi.yaml` until generated clients are introduced.

---

## Validation

Runtime validation is currently handled by the Go API. Add frontend runtime schema validation only when accepting untrusted non-API inputs or generated model JSON.

---

## Common Patterns

Use type-only imports:

```ts
import type { FormEvent } from 'react'
import type { Project } from './api/types'
```

---

## Forbidden Patterns

- Do not use `any`.
- Do not type-assert API responses inside components.
- Do not define duplicate DTO types in components.
