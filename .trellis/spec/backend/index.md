# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

This directory contains executable guidelines for Manmu backend development. The current backend is a Go modular monolith scaffold.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module organization and file layout | Active |
| [Database Guidelines](./database-guidelines.md) | ORM patterns, queries, migrations | Active |
| [Error Handling](./error-handling.md) | Error types, handling strategies | Active |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns | Active |
| [Logging Guidelines](./logging-guidelines.md) | Structured logging, log levels | Active |

---

## Pre-Development Checklist

Before changing backend code:

1. Read `directory-structure.md` for package boundaries.
2. Read `error-handling.md` before adding handlers, services, or domain errors.
3. Read `database-guidelines.md` before adding migrations, repositories, or transactions.
4. Read `logging-guidelines.md` before adding logs or middleware.
5. Read `quality-guidelines.md` before validation.
6. Run Go checks with the local toolchain unless the project intentionally upgrades Go:
   ```bash
   GOTOOLCHAIN=local go test ./...
   GOTOOLCHAIN=local go build ./...
   ```

## Quality Check

For backend changes, run:

```bash
gofmt -w apps internal
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go build ./...
```

If API behavior changed, smoke-check the route with `go run ./apps/api` and `curl`.
