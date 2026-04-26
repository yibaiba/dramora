# Logging Guidelines

> How logging is done in this project.

---

## Overview

Use the standard library `log/slog` for backend structured logging. API and worker entrypoints should create a JSON handler writing to stdout.

---

## Log Levels

- `info`: server start, worker start, request completion, expected lifecycle events.
- `warn`: recoverable degraded behavior, future provider rate-limit warnings.
- `error`: startup failure, server failure, worker failure, unexpected handler/service failure.

---

## Structured Logging

Required request fields from middleware:

- `method`
- `path`
- `status`
- `bytes`
- `duration_ms`
- `request_id`

Process loggers should include `env`.

---

## What to Log

- API server listen address.
- Worker queue names on startup.
- HTTP request completion.
- Future generation job submission, retry, cancel, and provider failure events.

---

## What NOT to Log

- Do not log provider API keys, database URLs, signed object URLs, prompts containing private user content, or generated media payloads.
- Do not log full request bodies by default.
