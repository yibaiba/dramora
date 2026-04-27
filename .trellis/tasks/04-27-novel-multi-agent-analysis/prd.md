# brainstorm: novel multi-agent analysis

## Goal

Build the Manmu story pre-production pipeline so Studio can accept novel/story text, run a multi-agent-style analysis flow, and produce structured outline, character, scene, prop, and episode planning artifacts that feed the existing story map, storyboard, asset, and timeline pipeline.

## What I already know

- The user wants continued product completion, specifically novel analysis that generates outline, characters, scenes, and related artifacts through multi-agent analysis.
- Current backend already has `POST /api/v1/episodes/{episodeId}/story-analysis/start`.
- Current local inline worker can auto-complete queued `story_analysis` jobs.
- Current `StoryAnalysis` only stores `summary`, `themes`, `character_seeds`, `scene_seeds`, and `prop_seeds`.
- Current worker completion is deterministic/no-op: it writes fixed MVP seeds such as `C01 protagonist` and `S01 opening scene`.
- Existing downstream flow can seed story map, candidate assets, storyboard shots, and approval gates after a story analysis exists.
- Frontend server state must go through `apps/studio/src/api/hooks.ts`.
- Frontend-facing API must remain GET/POST only, using POST command routes for mutations.

## Assumptions (temporary)

- The first practical slice should produce richer deterministic/local analysis artifacts before relying on external LLM credentials.
- "Multi-agent" should be represented as explicit analysis stages/roles in backend data and UI, even if local execution is deterministic in MVP.
- The downstream C/S/P and storyboard pipeline should reuse generated analysis artifacts instead of introducing a separate disconnected flow.

## Open Questions

- None for the first MVP slice.

## Requirements (evolving)

- Add a story/novel source input for an episode.
- Run an analysis flow that separates agent roles such as story analyst, outline planner, character analyst, scene analyst, and prop analyst.
- Produce structured output that can drive C/S/P story map and storyboard generation.
- Show analysis progress and outputs in Studio.
- First slice uses deterministic local agents and does not require external LLM credentials.

## Acceptance Criteria (evolving)

- [ ] User can add/paste source story text for an episode.
- [ ] User can start novel/story analysis from Studio.
- [ ] Analysis completion creates structured outline, character, scene, and prop artifacts.
- [ ] Studio displays the generated outline/person/scene/prop results.
- [ ] Existing one-click production flow can use the generated artifacts.
- [ ] Go tests and Studio lint/build pass.

## Definition of Done (team quality bar)

- Tests added/updated for backend service, HTTP routes, and frontend data flow where appropriate.
- Lint/typecheck/build green.
- OpenAPI updated for any new/changed API contract.
- Trellis specs updated if new durable conventions are introduced.
- Rollout/rollback considered for migrations or schema changes.

## Out of Scope (explicit)

- Full production-quality LLM prompt optimization.
- Browser smoke testing unless explicitly requested.
- Real multi-provider billing/cost controls for this first slice.
- Long-document chunking beyond an MVP-safe input limit unless selected as MVP scope.

## Technical Notes

- Existing route: `internal/httpapi/workflows.go` starts story analysis and lists story analyses.
- Existing service: `internal/service/production_service.go` creates a queued `story_analysis` generation job.
- Existing worker: `internal/service/generation_worker_service.go` calls `completeGeneratedStoryAnalysis` for `story_analysis` no-op jobs.
- Existing artifact model: `internal/domain/production.go` has `StoryAnalysis` with summary/themes/C/S/P seeds only.
- Existing completion code: `internal/service/story_analysis_service.go` writes hardcoded seeds.
- Existing downstream route: `POST /api/v1/episodes/{episodeId}/production:seed` now seeds story map, assets, storyboard shots, and approval gates.

## Research Notes

### What similar tools do

- Prior project research described a pipeline from story input to analysis, episode split, character design, scene design, prop design, storyboard, prompt build, video generation, timeline assembly, and export.
- Common pattern: keep agent stages explicit so users can inspect/retry/approve each major creative decision.

### Constraints from our repo/project

- Studio mutations must be POST commands.
- Server state must go through `src/api/client.ts` and `src/api/hooks.ts`.
- Backend should keep handlers thin; orchestration belongs in `internal/service`.
- Local development should work without secrets, so deterministic or mock analysis remains necessary.

### Feasible approaches here

**Approach A: Deterministic staged analyzer MVP (Recommended)**

- How it works: add story source input and a local multi-stage analyzer that parses text into outline/person/scene/prop structures using deterministic rules/templates, while recording agent-stage outputs.
- Pros: testable, works locally, unblocks UI/data model, no secret dependency.
- Cons: creative quality is limited until real LLM provider integration is added.

**Approach B: Provider-backed LLM analyzer first**

- How it works: wire a real LLM adapter and prompt chain for story analyst, outline planner, character analyst, and scene analyst.
- Pros: closer to target product quality.
- Cons: needs provider selection, secrets, retries, cost controls, JSON repair, and more failure handling.

**Approach C: UI-only agent board first**

- How it works: show multi-agent stages and mock outputs in Studio without deep backend persistence.
- Pros: fastest visual progress.
- Cons: risks fake success and does not improve production data quality.

## Decision (ADR-lite)

**Context**: The project needs a visible multi-agent story pre-production flow, but local development must keep working without provider secrets and the existing downstream pipeline already depends on `StoryAnalysis` seeds.

**Decision**: Implement Approach A first: deterministic local multi-stage analysis with persisted story source input and structured outline/person/scene/prop/agent outputs. Keep provider-backed LLM analysis out of scope for this slice.

**Consequences**: This unblocks real end-to-end data flow and UI inspection while creative quality remains template/rule based. Later provider adapters can replace the deterministic analyzer without changing the Studio data contract.
