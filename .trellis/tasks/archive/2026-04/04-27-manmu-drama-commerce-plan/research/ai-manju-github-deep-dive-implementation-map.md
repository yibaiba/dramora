# AI Manju GitHub Deep Dive: Implementation Map

## Purpose

Convert GitHub research into a concrete implementation map for Manmu.

The goal is not to clone any single project. Manmu should combine:

- AIComicBuilder's staged manju pipeline,
- AIYOU's node/workflow mental model,
- Seedance2's C/S/P asset and time-coded prompt discipline,
- Openjourney's candidate grid UX,
- StoryDiffusion/DiffSensei's consistency lessons,
- Wan/CogVideo/AnimateDiff provider capability modeling,
- MuseTalk/Wav2Lip optional voice/lip-sync pipeline,
- Manmu's own Go + React + PostgreSQL + workflow/job/cost architecture.

## Direct product references

### AIComicBuilder

Repository: `twwch/AIComicBuilder`

Why it matters:

- Strongest direct MVP reference for AI manju generation.
- End-to-end pipeline: script import -> role extraction -> storyboard -> start/end frames -> video prompt -> video generation -> FFmpeg composition.
- Uses storyboard drawer, inline character panel, and Kanban view.
- Supports project-level characters and cross-episode reuse.
- Supports OpenAI/Gemini/Kling/Seedance/Veo provider configuration.

Implementation lessons for Manmu:

- Keep the pipeline stage-based and manually controllable.
- Do not auto-spend on video generation before character/scene/shot approval.
- Make start/end keyframes first-class shot assets.
- Provide storyboard Kanban from day one.
- Add final video download and asset package export.
- Support script import from TXT/DOCX/PDF, but MVP can begin with pasted text and add file parsing later.

Differences:

- AIComicBuilder is Next/SQLite/Drizzle; Manmu will be Go/PostgreSQL.
- AIComicBuilder's task model should inspire UX but Manmu needs durable workflow/job tables.

### AIYOU open AI video drama generator

Repository: `yubowen123/AIYOU_open-ai-video-drama-generator`

Why it matters:

- Best node-workflow product reference for short drama/manju.
- Covers script outline, episode script, character design, storyboard, storyboard image, Sora/video generation, and drama analysis nodes.
- User-facing workflow feels like building a creative pipeline.

Implementation lessons for Manmu:

- Use typed workflow nodes and typed edges internally.
- Use node dependency rules:
  - allowed inputs,
  - allowed outputs,
  - min/max inputs,
  - circular dependency prevention.
- Make node output feed downstream artifacts.
- Surface node/job status, progress, fallback, and cancel controls.
- Keep renderer-independent workflow graph; React Flow/custom canvas should be swappable.

MVP decision:

- Do not expose freeform node editing in v1.
- Show fixed SOP as Agent Board / read-only graph.
- Add editable graph only after backend workflow graph and node handlers are stable.

### Seedance2 Storyboard Generator

Repository: `liangdabiao/Seedance2-Storyboard-Generator`

Why it matters:

- Best prompt/SOP reference for novel/story to multi-episode video.
- Defines C/S/P numbering for characters, scenes, props.
- Uses time-coded prompt slices:
  - 0-3s,
  - 3-6s,
  - 6-9s,
  - 9-12s,
  - 12-15s.
- Stores tail-frame descriptions for continuity.
- Documents provider limits: max reference images/videos, prompt length, sensitive words.

Implementation lessons for Manmu:

- Asset codes are required:
  - `Cxx` for characters,
  - `Sxx` for scenes,
  - `Pxx` for props.
- `PromptPack` should support timed segments, not just one long prompt string.
- Store tail-frame text and optional tail-frame asset per shot/episode.
- Provider capability metadata must include reference count and duration limits.
- Prompt Engineer Agent should shorten, segment, and provider-normalize prompts.

### Openjourney

Repository: `ammaarreshi/openjourney`

Why it matters:

- Best lightweight generation UX reference.
- 4-image candidate grid, 2x2 video grid, skeleton loading, hover preview, fullscreen lightbox, filmstrip navigation.

Implementation lessons for Manmu:

- Use candidate grid for character, scene, prop, keyframe, and video variations.
- Lock selected candidate as the version used downstream.
- Use filmstrip navigation for shot variants and regenerated clips.
- Loading states must be prominent and polished because model calls are slow.

## Consistency and manga references

### StoryDiffusion

Repository: `HVision-NKU/StoryDiffusion`

Why it matters:

- Focuses on consistent self-attention for long-range image generation.
- Comic sequence consistency is central to Manmu's product quality.
- Shows two-stage long video idea: consistent condition images first, then video between them.

Implementation lessons:

- Generate and lock references before batch shot generation.
- Treat consistency as product state, not a hidden model detail.
- Add Continuity Supervisor Agent before final timeline/export.
- Use start/end frames and condition images as the continuity spine.

### DiffSensei

Repository: `jianzongwu/DiffSensei`

Why it matters:

- Bridges MLLM and diffusion for customized manga panel generation.
- Supports flexible character adaptation from one input character image.
- Supports varied-resolution manga panels.

Implementation lessons:

- Future manga-panel mode can be a separate output format besides animated manju.
- Keep panel layout metadata and shot/storyboard metadata structured.
- MLLM layout reasoning can later improve storyboard/panel composition.

### MangaNinja

Repository: `ali-vilab/MangaNinjia`

Why it matters:

- Reference-based line-art colorization.

Implementation lessons:

- If Manmu later supports line-art storyboard first, colorization can be a separate provider task.

Risk:

- Non-commercial license; avoid commercial code/model reuse without permission.

## Video model and provider references

### Wan2.1

Repository: `Wan-Video/Wan2.1`

Key capabilities:

- Text-to-video,
- image-to-video,
- video editing,
- text-to-image,
- video-to-audio,
- first-last-frame-to-video,
- VACE all-in-one video creation/editing,
- Chinese/English visual text generation,
- consumer GPU option: T2V-1.3B needs around 8.19GB VRAM and can generate 5s 480p on RTX 4090 in about 4 minutes without optimizations.

Implementation lessons:

- `model_catalog.supported_tasks` must include `t2v`, `i2v`, `flf2v`, `video_editing`, `v2a`.
- Manmu's shot model should keep start and end frame fields.
- Wan is a strong later self-hosted worker candidate; external API remains first.
- Provider adapter should expose `preferred_prompt_language` and `supports_first_last_frame`.

### CogVideo / CogVideoX

Repository: `zai-org/CogVideo`

Key capabilities:

- Text-to-video,
- image-to-video,
- video continuation,
- higher-resolution 10-second video support in CogVideoX1.5,
- prompt optimization with large models,
- Diffusers and quantized inference support.

Implementation lessons:

- Prompt optimization should be a recorded step.
- Provider adapter should distinguish:
  - prompt extension,
  - video generation,
  - video continuation.
- Store optimized prompt separately from user/original prompt.

### AnimateDiff

Repository: `guoyww/AnimateDiff`

Key capabilities:

- Plug-and-play motion module for personalized T2I diffusion models.
- MotionLoRA for zoom, pan, tilt, and rolling camera motions.
- SparseCtrl RGB/sketch controls.

Implementation lessons:

- Cinematographer Agent should normalize camera movement into provider-specific motion controls.
- Shot fields should include camera movement, shot size, angle, lens feel, and motion tags.
- Future self-hosted animation worker can support sketch/RGB controls.

## Voice and lip-sync references

### MuseTalk

Repository: `TMElyralab/MuseTalk`

Why it matters:

- Real-time high-fidelity video dubbing/lip-sync reference.
- Useful for future talking-character shots.

Implementation lessons:

- Voice & Subtitle Agent must output:
  - speaker character,
  - language,
  - line text,
  - audio asset,
  - subtitle timing,
  - optional lip-sync target shot/clip.
- Lip-sync should be optional by shot, not mandatory for every manju segment.

### Wav2Lip

Repository: `Rudrabha/Wav2Lip`

Why it matters:

- Strong conceptual reference for lip-syncing.

Risk:

- Open-source model is research/non-commercial; do not use commercially without permission.

Implementation lessons:

- Design `lip_sync` as a provider task type.
- Prefer external commercial APIs or licensed models for production.

## Implementation priority map

### P0: Manmu MVP must implement

- Project -> Episode production hierarchy.
- Story source input and story analysis.
- C/S/P asset numbering.
- Character card and character reference candidate generation.
- Scene card and scene concept candidate generation.
- Prop card and prop candidate generation.
- Lock selected asset versions.
- Storyboard/ShotCard generation.
- Shot prompt packs with provider-specific prompt text.
- Start-frame and optional end-frame assets.
- Video generation jobs per shot.
- Candidate grid and Storyboard Kanban.
- Timeline editor narrow MVP.
- Server-side FFmpeg export.
- Asset lineage and cost ledger.
- Agent Board with fixed SOP and approval gates.

### P1: strong near-term enhancements

- File import for TXT/DOCX/PDF.
- Tail-frame continuity and next-episode extension workflow.
- Timed prompt slices per shot.
- Continuity Supervisor Agent issue report.
- Prompt optimization as a recorded step.
- Provider model scoring and retry suggestions.
- Basic TTS/subtitle generation per dialogue line.
- Export asset package ZIP.

### P2: after core product works

- Read-only workflow graph visualization.
- Limited editable workflow branches.
- WebCodecs browser preview/export for small projects.
- Remotion render worker if license approved.
- ComfyUI/custom GPU worker adapters.
- Wan/CogVideo self-hosted workers.
- Lip-sync with licensed provider.
- Manga panel / black-and-white comic page mode.

### Explicitly avoid in v1

- Freeform Agent workflow builder.
- Full Premiere/CapCut-grade editing suite.
- Default self-hosted GPU inference.
- Commercial use of non-commercial models or assets.
- Browser-only export as the primary export path.

## Architecture updates from deep dive

Add or preserve these backend concepts:

- `asset_code` for C/S/P numbering.
- `candidate_group_id` for generated alternatives.
- `locked_version_id` pattern for character/scene/prop.
- `prompt_segments` for time-coded video prompts.
- `tail_frame_asset_id` and `tail_frame_description`.
- `provider_capabilities` for reference limits, duration, FLF2V, continuation, prompt length.
- `prompt_renders` for original and optimized prompts.
- `generation_jobs.request_key` for idempotency.
- `artifact_edges` for lineage.
- `review_issues` for continuity/safety findings.

## Product UX updates from deep dive

Add or preserve these frontend surfaces:

- candidate grid,
- filmstrip variant browser,
- storyboard Kanban,
- character inline panel in storyboard,
- prompt pack editor,
- keyframe comparison view,
- shot detail drawer,
- provider/model capability warnings before generation,
- budget warning before video fanout,
- export and asset package download.

## Final implementation stance

Manmu should be implemented as:

```text
AIComicBuilder-like staged manju pipeline
  + AIYOU-like typed workflow graph internally
  + Seedance2-like asset/prompt discipline
  + Openjourney-like candidate UX
  + Go/PostgreSQL durable workflow/job/cost backend
  + React Studio focused on approval, generation control, and final editing
```

This gives the fastest practical MVP while leaving room for node canvas, self-hosted models, and advanced editing later.
