# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

Backend changes must remain buildable with the repository's pinned Go toolchain and must keep package boundaries clean. The current scaffold is Go 1.21-compatible and uses Chi v5.0.12.

---

## Forbidden Patterns

- Do not add infrastructure imports to `internal/domain`.
- Do not put business logic in `apps/*/main.go`.
- Do not add provider secrets, tokens, or real credentials to code, config examples, tests, or docs.
- Do not use broad success-shaped fallbacks for invalid input. Return explicit errors.
- Do not upgrade Go or Chi versions incidentally while adding unrelated code.

---

## Required Patterns

### Scenario: Go backend scaffold validation

#### 1. Scope / Trigger

- Trigger: backend implementation changes commands, routes, package boundaries, or env behavior.

#### 2. Signatures

Required validation commands:

```bash
gofmt -w apps internal
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go build ./...
```

Optional API smoke:

```bash
MANMU_HTTP_ADDR=127.0.0.1:18080 GOTOOLCHAIN=local go run ./apps/api
curl -fsS http://127.0.0.1:18080/healthz
```

#### 3. Contracts

- Keep `go.mod` on the repository's intended Go version unless the task explicitly upgrades it.
- Keep dependencies minimal and direct dependencies explicit.
- Add tests for domain validators and state transitions.
- Add route tests when handler behavior becomes non-placeholder.

#### 4. Validation & Error Matrix

| Condition | Required response |
| --- | --- |
| `go test ./...` fails | Fix before claiming completion. |
| `go build ./...` fails | Fix before claiming completion. |
| Dependency requires a newer Go version | Pin a compatible version or explicitly plan a Go upgrade. |
| Route contract changes | Update `api/openapi.yaml` and handler tests. |
| New env parsing logic | Test default, valid, and invalid cases. |
| `go test ./...` enters frontend `node_modules` | Keep `apps/studio/go.mod` as a nested module boundary. |

#### 5. Good/Base/Bad Cases

- Good: pin Chi to a Go-compatible version and validate with `GOTOOLCHAIN=local`.
- Base: placeholder handlers compile and return stable response envelopes.
- Bad: letting `go get` silently upgrade the module to a Go version unavailable in the current environment.

#### 6. Tests Required

- Domain transition tests for each new state machine.
- Handler tests for non-placeholder route validation and error mapping.
- Repository integration tests against PostgreSQL once SQL is introduced.
- Worker tests for idempotency and no-op execution once River/job execution is introduced.

#### 7. Wrong vs Correct

##### Wrong

```bash
go get github.com/go-chi/chi/v5@latest
go test ./...
```

This can silently require a newer Go toolchain.

##### Correct

```bash
GOTOOLCHAIN=local go get github.com/go-chi/chi/v5@v5.0.12
GOTOOLCHAIN=local go test ./...
```

Pin dependencies intentionally and validate against the local supported toolchain.

---

## Testing Requirements

- New status validators require table-driven unit tests.
- New handler behavior requires status code and response shape tests.
- New repository behavior should be tested against PostgreSQL rather than SQLite.
- New worker behavior must test no-op execution and retry/idempotency boundaries.

---

## Code Review Checklist

- Package boundaries match `directory-structure.md`.
- API error responses match `error-handling.md`.
- OpenAPI is updated for route changes.
- `go test ./...` and `go build ./...` pass with `GOTOOLCHAIN=local`.
- No secrets or provider credentials are introduced.
