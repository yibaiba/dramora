# Frontend Development Guidelines

> Best practices for frontend development in this project.

---

## Overview

This directory contains executable guidelines for Manmu Studio frontend development. The current Studio app is Vite + React + TypeScript under `apps/studio`.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module organization and file layout | Active |
| [Component Guidelines](./component-guidelines.md) | Component patterns, props, composition | Active |
| [Hook Guidelines](./hook-guidelines.md) | Custom hooks, data fetching patterns | To fill |
| [State Management](./state-management.md) | Local state, global state, server state | Active |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns | Active |
| [Type Safety](./type-safety.md) | Type patterns, validation | Active |

---

## Pre-Development Checklist

Before changing Studio code:

1. Read `directory-structure.md`.
2. Read `component-guidelines.md` for accessibility and layout rules.
3. Read `state-management.md` before adding server/local state.
4. Read `type-safety.md` before changing API DTOs.
5. Read `quality-guidelines.md` before validation.
6. Run:
   ```bash
   cd apps/studio
   npm run lint
   npm run build
   ```

## Quality Check

For Studio changes:

```bash
cd apps/studio
npm run lint
npm run build
```

If API integration changed, smoke-check through Vite proxy while the Go API is running.
