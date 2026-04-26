# AIYOU Workflow Canvas Deep Dive

## Purpose

Deep dive into AIYOU's node canvas and workflow design, then map useful ideas into Manmu Studio.

Repository: https://github.com/yubowen123/AIYOU_open-ai-video-drama-generator

## Important correction

AIYOU README describes a React Flow-style node workflow, but the inspected code does not include React Flow / xyflow in `package.json`.

The current implementation is closer to a custom canvas:

- `AppNode` stores `x`, `y`, `width`, `height`, `title`, `status`, `data`, and `inputs`.
- `Connection` stores `from` and `to`.
- `ConnectionLayer.tsx` renders SVG Bezier curves between node ports.
- Zustand stores manage nodes, connections, viewport, selection, drag state, and workflow state.
- `utils/nodeValidation.ts` defines node dependency rules.
- `handlers/useNodeActions.ts` contains most node execution logic.
- `services/nodes` begins extracting node execution into a service registry.

For Manmu, the key lesson is not "must use React Flow"; the key lesson is "make production workflow a real graph with typed nodes, typed edges, validation, and traceable execution".

## Core data model observed

### Node types

AIYOU defines these node types:

- `PROMPT_INPUT`
- `IMAGE_GENERATOR`
- `VIDEO_GENERATOR`
- `VIDEO_ANALYZER`
- `IMAGE_EDITOR`
- `AUDIO_GENERATOR`
- `SCRIPT_PLANNER`
- `SCRIPT_EPISODE`
- `STORYBOARD_GENERATOR`
- `STORYBOARD_IMAGE`
- `STORYBOARD_SPLITTER`
- `CHARACTER_NODE`
- `DRAMA_ANALYZER`
- `DRAMA_REFINED`
- `STYLE_PRESET`
- `SORA_VIDEO_GENERATOR`
- `SORA_VIDEO_CHILD`
- `STORYBOARD_VIDEO_GENERATOR`
- `STORYBOARD_VIDEO_CHILD`
- `VIDEO_EDITOR`

### Node status

AIYOU uses:

- `IDLE`
- `WORKING`
- `SUCCESS`
- `ERROR`

Manmu should expand this for backend jobs:

- `draft`
- `ready`
- `queued`
- `running`
- `waiting_approval`
- `succeeded`
- `failed`
- `canceled`
- `skipped`

### Node payload

`AppNode.data` is a large union-like object with many optional fields, including:

- prompt/model/media fields,
- script planner fields,
- episode fields,
- storyboard fields,
- character profile fields,
- Sora task groups,
- storyboard video config,
- generated prompt,
- fused image,
- child node ids,
- progress/error/status fields.

Manmu should avoid one giant untyped payload. Use:

- `WorkflowNode` as generic shell.
- `node_type` and `node_version`.
- `input_artifact_ids`.
- `output_artifact_ids`.
- typed JSON schemas per node type.
- backend validation per schema.

## Canvas interaction design

### Custom canvas mechanics

AIYOU tracks:

- viewport pan/zoom,
- selected nodes,
- selected connections,
- dragging node id,
- resizing node id,
- selected group id,
- active group node ids,
- connection start point,
- selection rectangle,
- clipboard.

It renders connections through `ConnectionLayer.tsx`:

- calculates path from output port to input port,
- uses cubic Bezier curve,
- adds invisible thick path for click target,
- shows animated dashed line while creating a connection,
- memoizes rendering to reduce re-renders.

Manmu implication:

- For MVP fixed SOP, avoid a full freeform editor.
- For Agent Board / future workflow canvas, keep the graph model independent from the renderer so React Flow, custom SVG, or tldraw-like canvas can be swapped later.

## Dependency validation

AIYOU has a useful `NODE_DEPENDENCY_RULES` map:

- each node type defines:
  - allowed input types,
  - allowed output types,
  - min inputs,
  - max inputs,
  - description.
- connection validation checks:
  - allowed output,
  - allowed input,
  - max input count,
  - duplicate connection,
  - circular dependency,
  - self-connection.
- execution validation checks required input/config per node.

Manmu should adopt this as `Agent/Workflow Edge Policy`.

Example Manmu node dependency shape:

```text
StoryInput -> StoryAnalysis
StoryAnalysis -> EpisodeSplit
StoryAnalysis -> CharacterDesign
StoryAnalysis -> SceneDesign
StoryAnalysis -> PropDesign
EpisodeSplit -> Storyboard
CharacterDesign -> Storyboard
SceneDesign -> Storyboard
PropDesign -> Storyboard
Storyboard -> KeyframeGeneration
KeyframeGeneration -> VideoPrompt
VideoPrompt -> VideoGeneration
VideoGeneration -> ContinuityReview
ContinuityReview -> TimelineAssembly
TimelineAssembly -> Export
```

## Execution model

### Current mixed approach

AIYOU currently has two execution layers:

1. Large `useNodeActions.ts` switch-style handler for many node actions.
2. Newer `services/nodes` abstraction:
   - `BaseNodeService`
   - `NodeExecutionContext`
   - `NodeExecutionResult`
   - `NodeServiceRegistry`
   - `executeNodesInOrder` with topological sort.

The service registry direction is much cleaner:

- node services are registered per node type,
- context provides nodes/connections and input data,
- `executeNodesInOrder` sorts nodes by dependencies,
- status and data updates are passed as callbacks.

Manmu implication:

- Put orchestration in Go backend, not in React.
- React Studio should request execution, display status, and allow approvals.
- Backend should implement node registry / step handlers with durable job state.
- Frontend may still run local preview/validation, but source of truth is backend.

## Story / character / storyboard flow

AIYOU's effective content flow:

```text
PromptInput
  -> ScriptPlanner
  -> ScriptEpisode
  -> auto-created PromptInput episode child nodes
  -> CharacterNode
  -> StoryboardImage
  -> StoryboardSplitter
  -> StoryboardVideoGenerator / SoraVideoGenerator
  -> StoryboardVideoChild / SoraVideoChild
  -> VideoEditor
```

Important details:

- `SCRIPT_EPISODE` generates episode child nodes automatically.
- `getUpstreamContext` recursively collects upstream text.
- `getUpstreamStyleContext` recursively finds script planner/style information.
- `STORYBOARD_IMAGE` can parse structured storyboard JSON or fallback to text parsing.
- `STORYBOARD_IMAGE` supports 6/9/16/25-panel grids.
- `STORYBOARD_IMAGE` extracts character reference images from upstream `CHARACTER_NODE`.
- character Chinese names are mapped to generic English labels like `Character A` / `Character B` to improve prompt following.
- `STORYBOARD_VIDEO_GENERATOR` fetches split shots, generates a provider prompt, uploads fused images, submits a video job, polls progress, then creates child result nodes.

Manmu implication:

- Keep upstream context traversal, but make it artifact-based:
  - `StoryAnalysisArtifact`
  - `EpisodeScriptArtifact`
  - `CharacterReferenceArtifact`
  - `SceneReferenceArtifact`
  - `PropReferenceArtifact`
  - `ShotCardArtifact`
  - `KeyframeArtifact`
  - `VideoClipArtifact`
- Automatically created child nodes are useful, but backend should create persisted artifacts/jobs instead of only frontend nodes.

## Professional storyboard language

AIYOU has a dedicated optimization document for more cinematic shot structure.

Useful fields:

- shot size,
- camera angle,
- camera movement,
- lens focal length,
- camera equipment,
- lens,
- camera,
- aperture,
- director intent,
- emotional tone,
- narrative purpose,
- enhanced visual description.

Manmu should include these as optional but first-class shot fields:

```text
ShotCard
  ├── shot_size
  ├── camera_angle
  ├── camera_movement
  ├── lens_focal_length
  ├── camera_equipment
  ├── lens_style
  ├── aperture_style
  ├── director_intent
  ├── emotional_tone
  └── narrative_purpose
```

This is especially useful for the Director Agent and Cinematographer Agent.

## Video provider abstraction

AIYOU's `videoPlatforms` abstraction defines:

- platform type: `yunwuapi`, `official`, `custom`.
- model type: `veo`, `luma`, `runway`, `minimax`, `volcengine`, `grok`, `qwen`, `sora`.
- unified config:
  - aspect ratio,
  - duration,
  - quality.
- provider interface:
  - `supportsImageToVideo`
  - `supportsDuration`
  - `submitTask`
  - `checkStatus`

Manmu should generalize this:

```text
ModelProvider
  ├── provider_type
  ├── model_type
  ├── supported_tasks
  ├── supported_durations
  ├── supported_aspect_ratios
  ├── max_reference_images
  ├── max_reference_videos
  ├── supports_first_last_frame
  ├── supports_video_continuation
  ├── cost_rules
  └── license_policy
```

## What Manmu should borrow

Borrow:

- typed workflow graph,
- node dependency rules,
- recursive upstream context collection,
- automatic child artifact/result generation,
- storyboard grid/split flow,
- character reference injection,
- shot cinematic metadata,
- provider abstraction with submit/poll,
- abort/cancel support,
- status/progress messages,
- model fallback concept.

Avoid:

- one huge optional `data` object for every node type,
- placing durable orchestration primarily in frontend state,
- letting prompt parsing rely on fragile regex as the primary path,
- mixing many unrelated node action cases in one file,
- using localStorage/API keys as the main production security model.

## Recommended Manmu design

### MVP UI

Use fixed production SOP with visual graph display:

- left: project steps / Agent Board,
- center: storyboard cards or current artifact editor,
- right: inspector for selected artifact/job,
- bottom: generation queue / cost / errors.

### Advanced UI

Expose node canvas later:

- start from read-only graph visualization,
- then allow limited configurable branches,
- only later support fully custom workflow editing.

### Backend source of truth

Every node execution must become a persisted backend job:

```text
workflow_run
workflow_node_run
agent_run
generation_job
artifact
artifact_edge
approval_gate
```

React node IDs should map to backend IDs, not create irreversible local-only state.

## Manmu workflow node map inspired by AIYOU

```text
StoryInputNode
  -> StoryAnalysisNode
  -> EpisodeSplitNode
  -> CharacterDesignNode
  -> SceneDesignNode
  -> PropDesignNode
  -> StoryboardNode
  -> StoryboardPanelNode
  -> ShotSplitterNode
  -> PromptBuildNode
  -> KeyframeNode
  -> VideoGenerationNode
  -> VideoResultNode
  -> VoiceSubtitleNode
  -> TimelineAssemblyNode
  -> ExportNode
```

## PRD-impacting decisions

- Manmu MVP should not depend on React Flow being available or perfect.
- Manmu must define workflow nodes and edges as backend domain models.
- The UI can render fixed SOP as a graph/board first.
- Agent Board and workflow canvas should share the same underlying workflow run data.
