# Error Handling

> How errors are handled in this project.

---

## Overview

Backend errors must stay explicit across layers. Domain packages expose sentinel errors and validators; HTTP handlers translate errors into the API error envelope; entrypoints log fatal startup/runtime errors and exit non-zero.

---

## Error Types

- Domain sentinel errors live in `internal/domain/errors.go`.
- Current sentinel:
  ```go
  var ErrInvalidTransition = errors.New("invalid status transition")
  var ErrInvalidInput = errors.New("invalid input")
  var ErrNotFound = errors.New("not found")
  ```
- Wrap sentinel errors with `%w` so callers can use `errors.Is`.

---

## Error Handling Patterns

### 1. Scope / Trigger

- Trigger: the backend scaffold introduced domain transition validation and HTTP response helpers.
- Applies to domain validators, HTTP handlers, config loading, API responses, and future service/repo errors.

### 2. Signatures

Domain transition validators:

```go
func (s WorkflowRunStatus) ValidateTransition(next WorkflowRunStatus) error
func (s GenerationJobStatus) ValidateTransition(next GenerationJobStatus) error
```

HTTP error helper:

```go
func writeError(w http.ResponseWriter, status int, code string, message string)
```

HTTP JSON helper:

```go
func writeJSON(w http.ResponseWriter, status int, payload any)
```

### 3. Contracts

API error response:

```json
{
  "error": {
    "code": "invalid_transition",
    "message": "workflow cannot transition from succeeded to running"
  }
}
```

Rules:

- `error.code` is stable snake_case for frontend branching.
- `error.message` is safe for display/logging and must not include secrets.
- Domain validators return wrapped sentinel errors.
- HTTP helpers must set status and content type before writing the body.
- Do not call `http.Error` from handlers; use `writeError`.

### 4. Validation & Error Matrix

| Condition | Error behavior |
| --- | --- |
| Invalid workflow/job status transition | Return error wrapping `domain.ErrInvalidTransition`; HTTP maps to 409 or 400 depending endpoint semantics. |
| Invalid request input | Return error wrapping `domain.ErrInvalidInput`; HTTP maps to 400 with `invalid_request`. |
| Missing entity | Return `domain.ErrNotFound`; HTTP maps to 404 with `not_found`. |
| Invalid config duration | `LoadConfig` returns parse error; entrypoint logs and exits non-zero. |
| JSON response payload cannot marshal | Return plain 500 before writing success status. |
| SSE unsupported by writer | Return JSON error with code `stream_not_supported` and status 500. |
| Provider secret/API failure | Future provider layer returns normalized error; handler must not expose secret values. |

### 5. Good/Base/Bad Cases

- Good: `fmt.Errorf("%w: %s to %s", domain.ErrInvalidTransition, current, next)`.
- Base: placeholder handlers return stable JSON envelopes until repositories exist.
- Bad: returning `"something went wrong"` without a code or swallowing provider errors as success.

### 6. Tests Required

- Domain validators: assert valid transitions return nil and invalid transitions satisfy `errors.Is(err, ErrInvalidTransition)`.
- Handlers: assert status code, `Content-Type`, and JSON error envelope.
- Config errors: assert invalid duration produces an error.
- SSE: assert unsupported streaming path returns `stream_not_supported` when test writer does not implement `http.Flusher`.

### 7. Wrong vs Correct

#### Wrong

```go
if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
```

#### Correct

```go
if err != nil {
    writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
    return
}
```

The correct form keeps the response shape stable and avoids leaking implementation details.

---

## API Error Responses

Use the JSON envelope:

```json
{
  "error": {
    "code": "stable_snake_case_code",
    "message": "safe human-readable message"
  }
}
```

---

## Common Mistakes

- Do not write a success status before encoding a response body. Marshal first, then set headers/status and write.
- Do not expose raw provider, database, or config values in client-facing messages.
- Do not convert all domain errors to 500; map expected validation/state errors to client-safe statuses.
