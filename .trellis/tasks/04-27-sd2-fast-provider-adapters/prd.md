# SD2 Fast Provider Adapters

## Goal

Build the next Manmu core slice around Seedance/SD2 fast video generation: provider adapter boundaries, provider-specific prompt packs, and Studio controls that turn storyboard shots plus locked C/S/P references into model-ready generation requests.

## What I already know

- User wants to continue provider adapters, approval gates, timeline editing, and export worker, with immediate focus on `sd2 fast`.
- User noted SD2 prompts are different and asked to inspect优秀 GitHub designs first.
- Current backend has `internal/provider.Adapter` with capabilities only, `jobs.Worker` polling queued generation jobs, and `generation_jobs` storing `prompt`, `params`, `provider_task_id`, costs, and `result_asset_id`.
- Current Studio already has project/episode, story analysis, C/S/P map, storyboard shots, timeline graph, export scaffold, and asset candidate lock flows.
- Frontend-facing APIs must remain GET/POST only.
- Secrets must not be committed; real provider calls must read secret refs/env at runtime.

## Research Notes

### GitHub references inspected

- `zhanghaonan777/Seedance2-skill`
  - Lists Volcengine Ark model IDs including `doubao-seedance-2-0-260128`, `doubao-seedance-1-0-pro-fast-251015`, and `doubao-seedance-1-0-lite-i2v-250428`.
  - CLI payload uses `POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks`.
  - Payload shape is content-array based: text prompt plus optional `image_url` roles such as `first_frame`, `last_frame`, `reference_image`; also video/audio refs.
  - Supports `ratio`, `duration`, `resolution`, `seed`, `camera_fixed`, `generate_audio`, `draft`, `return_last_frame`, `service_tier`, `frames`, `callback_url`.
  - Prompt quality system gates: memorability, surprise, emotion arc, narrative change.
- `Anil-matcha/seedance2-comfyui`
  - Separates T2V, I2V, Omni Reference, Consistent Character, Consistent Character Video, Extend, and Save Video nodes.
  - Uses explicit reference tokens `@image1`…`@image9`, `@video1`…`@video3`, `@audio1`…`@audio3`.
  - Emphasizes consistent character via character sheet / `@character:<id>` and first-frame outputs for downstream workflows.
  - MuAPI-style endpoints are separated by task type: T2V, I2V, Omni, character, extend, poll.
- `Anil-matcha/n8n-nodes-seedance2`
  - Confirms useful workflow automation categories: text-to-video, image-to-video, video extension, advanced aspect ratio controls, polling.
- `verysun/seedance-video-prompt-generation`
  - Encodes Seedance prompts as structured shot segments.
  - Important prompt design: each segment ≤15s; include camera movement, shot size, lighting/style, stability constraints, character consistency constraints, and dialogue timing if present.
  - Supports five storyboard methods: timeline-by-seconds, shot-number, visual blocks, act structure, and action sequence.

### Prompt design conclusions for Manmu

- Do not store only one generic `prompt` for SD2.
- Store a provider-specific `prompt pack` per shot or per generation request:
  - shot code and duration,
  - direct copyable SD2 prompt,
  - time slices such as `0-3s / 3-6s / 6-9s`,
  - first frame / last frame text and/or asset refs,
  - reference asset bindings (`@image1`, `@image2`, etc.),
  - camera movement and shot size,
  - stability constraints,
  - character/scene/prop continuity constraints,
  - model parameters: model id, ratio, resolution, duration, seed, service tier, return-last-frame.
- For Manmu MVP, `sd2 fast` should map to a low-latency model preset, likely `doubao-seedance-1-0-pro-fast-251015`, while preserving a capability catalog so we can add Seedance 2.0 Omni later.

### Constraints from this repo

- PostgreSQL is source of truth.
- `db/queries/*.sql` stays aligned with hand-written pgx SQL.
- Components must consume hooks, not direct fetch.
- API uses GET for reads and POST for writes/actions.
- `internal/domain` must not import provider SDKs.
- Provider SDK response types must not leak outside `internal/provider`.

## Requirements

- Add a Seedance/SD2-aware provider capability layer that can represent:
  - text-to-video,
  - image-to-video,
  - first/last-frame video,
  - multi-reference/omni reference,
  - video extension.
- Add a provider-specific prompt pack contract that can be generated from existing `storyboard_shots`, locked assets, C/S/P map, and timeline duration.
- Preserve real-provider execution as asynchronous jobs; no synchronous long video generation request from HTTP handlers.
- Support `sd2 fast` as a named preset without hardcoding credentials.
- Keep tests runnable without real provider credentials by using fake transport/no-op provider behavior.
- Use a hybrid adapter boundary: fake mode by default, real Volcengine Ark POST mode when `ARK_API_KEY` is present.
- Include image-to-video (`image2` / `@image2`) reference bindings in the first prompt-pack slice.

## Acceptance Criteria

- [x] Manmu can generate and read an SD2 prompt pack for a storyboard shot.
- [x] Prompt pack includes time slices, camera motion, first-frame/reference guidance, reference bindings, stability constraints, and model preset params.
- [x] `sd2 fast` provider preset is visible through backend capabilities and Studio prompt pack UI.
- [x] Worker submits Seedance video jobs asynchronously and persists provider task ids.
- [x] Worker polls submitted/polling Seedance jobs and advances completed output through postprocessing.
- [x] Worker creates a ready video result asset and writes `generation_jobs.result_asset_id` before completion.
- [x] Studio shows export worker status with polling after `Start export`.
- [x] No real API key or provider secret is committed.
- [x] Backend tests cover prompt pack generation and provider adapter payload normalization.
- [x] Studio can show/copy SD2 prompt pack through GET/POST-only routes.
- [x] Existing Go and Studio validation passes.

## Definition of Done

- Tests added/updated for new backend and Studio behavior.
- `GOTOOLCHAIN=local go test ./...`
- `GOTOOLCHAIN=local go build ./...`
- `cd apps/studio && npm run lint -- --quiet`
- `cd apps/studio && npm run build`
- OpenAPI and Trellis specs updated for new route/data contracts.
- GET/POST-only API contract scan passes.

## Out of Scope

- No committed provider secrets.
- No self-hosted GPU inference.
- No full River migration in this slice unless required by adapter execution.
- No advanced timeline trim/split UI beyond what is required to feed prompt packs.
- No production-grade billing ledger; use existing cost placeholders until provider calls are real.

## Technical Approach

Recommended MVP sequence:

1. Add a `PromptPack`/`ShotPromptPack` contract and route:
   - `POST /api/v1/storyboard-shots/{shotId}/prompt-pack:generate`
   - `GET /api/v1/storyboard-shots/{shotId}/prompt-pack`
2. Add SD2 prompt renderer that transforms Manmu shot + C/S/P refs into:
   - `direct_prompt`,
   - `time_slices`,
   - `reference_bindings`,
   - `provider_params`.
3. Add Seedance/SD2 adapter scaffold in `internal/provider`:
   - capability metadata,
   - normalized request payload builder,
   - fake transport tests,
   - env/secret-ref runtime config only.
4. Add Studio panel affordance:
   - generate/copy prompt pack,
   - show SD2 fast model preset,
   - enqueue generation later via generation job.

## Open Questions

- Resolved: first implementation uses hybrid mode. Fake mode is default; real Ark POST mode is available inside the provider adapter when `ARK_API_KEY` is present. HTTP handlers still only generate/read prompt packs; video job submission remains a follow-up slice.

## Implementation Notes

- Added persistent `shot_prompt_packs` with JSONB time slices, reference bindings, and provider params.
- Added `sd2_fast` preset mapped to `doubao-seedance-1-0-pro-fast-251015`.
- Added Seedance adapter request builder and hybrid fake/Ark submission boundary.
- Studio shot cards can generate, display, and copy SD2 prompt packs.
- Added `POST /api/v1/storyboard-shots/{shotId}/videos:generate` to queue a generation job from the current prompt pack.
- Added MVP approval gates with seed/list/approve/request-changes routes and Studio approval board.
- Enhanced Studio timeline editing with a component-local draft: build clips from storyboard, append locked asset clips, edit clip start/length, remove clips, and save through the existing `POST /api/v1/episodes/{episodeId}/timeline` route.
- Added backend timeline graph validation for blank track/clip fields, negative timing, and clips exceeding the timeline duration.
- Added export worker execution: `StartEpisodeExport` enqueues `export.render`, the worker processes queued/rendering exports, and exports advance `queued -> rendering -> succeeded` through repository-backed status updates.
- Added real Seedance worker boundary: queued SD2 video jobs advance to `submitting`, call the provider adapter, persist `provider_task_id`, then poll submitted/polling jobs and complete fake/finished provider tasks via `downloading -> postprocessing -> succeeded`.
- Expanded generation job repository reads to include `prompt`, `params`, and `provider_task_id` so worker execution uses the persisted prompt pack payload instead of rebuilding provider input in HTTP handlers.
- Added Studio export status polling through `useExport(exportId)` and an accessible timeline export status card for queued/rendering/succeeded/failed/canceled states.
- Added provider result asset persistence: Seedance poll extracts a provider result URI, the worker creates a ready `video` asset during the `downloading` step, and `generation_jobs.result_asset_id` is persisted and exposed in the generation job DTO.
