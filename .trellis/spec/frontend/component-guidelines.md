# Component Guidelines

> How components are built in this project.

---

## Overview

Manmu Studio uses accessible React components with a dark, video-first creative workspace style. Structural icons must use Lucide or another consistent SVG icon set, not emoji.

---

## Component Structure

Current scaffold keeps first-slice components in `App.tsx`; split into feature files once a component grows or gains tests.

---

## Props Conventions

- Props should be explicit object types.
- Use `import type` for type-only imports.
- Keep required user-visible labels explicit for accessibility.

---

## Styling Patterns

Use Tailwind v4 via `@tailwindcss/vite` plus custom CSS tokens in `src/index.css`. Preserve:

- dark background surfaces,
- high contrast text,
- 44px minimum target size,
- visible focus rings,
- `prefers-reduced-motion` handling.

---

## Accessibility

- Use semantic landmarks (`main`, `aside`, `nav`, `section`, `header`).
- Use `aria-label` for navigation and rail regions.
- Mark decorative icons with `aria-hidden="true"`.
- Inputs must have visible labels.
- Loading regions should use `aria-busy` where applicable.
- Agent/status boards must include text labels/counts; do not communicate progress by color alone.

---

## Common Mistakes

- Do not use emoji as structural icons.
- Do not remove focus outlines.
- Do not rely on hover-only interactions.
- Do not create tap targets below 44px height.
