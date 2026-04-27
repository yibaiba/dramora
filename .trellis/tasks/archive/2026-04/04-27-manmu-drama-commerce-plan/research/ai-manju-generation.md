# Research: AI Manju Generation Platform

## Corrected product direction

The product is an AI manju / animated comic drama generation platform, not a traditional drama ecommerce website.

The platform should help creators produce short animated comic drama assets through:

1. LLM-based writing and planning.
2. Story bible and character generation.
3. Shot/scene/storyboard generation.
4. Character card and character reference image generation.
5. Scene map, scene concept image, and prop reference generation.
6. Image/video model prompting.
7. TTS/subtitles.
8. Timeline editing, regeneration, review, and export.

## Product/platform references

### Runway

- Reference for professional AI video generation and editing UX.
- Useful patterns:
  - Scene/shot-level generation.
  - Timeline-style review.
  - Iterative regeneration.
  - Video-first product presentation.

### Pika

- Reference for fast and creator-friendly prompt-to-video workflows.
- Useful patterns:
  - Fast iteration loop.
  - Short clip generation.
  - Effects and camera-motion controls.

### Kling / 可灵

- Reference for Chinese-market high-quality text-to-video and image-to-video generation.
- Useful patterns:
  - Strong realism and cinematic shots.
  - Chinese creator expectations.
  - API/provider abstraction should allow plugging in local China-friendly vendors.

### Luma Dream Machine

- Reference for image-to-video and realistic motion.
- Useful patterns:
  - Turning still references/keyframes into video shots.
  - Simple prompt-driven generation UX.

### PixVerse

- Reference for social short-video generation and rapid experimentation.
- Useful patterns:
  - Lightweight prompt iteration.
  - Social-media-oriented output.

### Krea / StoryboardHero-style products

- Reference for script-to-storyboard and visual planning.
- Useful patterns:
  - Script import.
  - Storyboard frames.
  - Character consistency.
  - Team review and iteration.

### Chinese manju/AI comic references

Current web research suggests domestic AI comic/manju products often combine:

- Text-to-comic/storyboard.
- Role and scene templates.
- Dialogue bubbles.
- AI voice / subtitles.
- One-click short video publishing.

These product details need direct hands-on verification before implementation.

### BigBanana AI Director

Repository: https://github.com/shuyu-labs/BigBanana-AI-Director

- One-stop AI short drama / AI manju director platform.
- Uses Script-to-Asset-to-Keyframe.
- Important scene ideas:
  - worldbuilding resources before episode generation,
  - map / regions / locations,
  - scene and prop assetization,
  - scene concept images,
  - context-aware shot generation using current scene, character wardrobe, and props.

### CineGen AI Director

Repository: https://github.com/Will-Water/CineGen-AI

- Motion comic / animatic workbench.
- Important scene ideas:
  - script analysis creates scenes with time and mood,
  - set design generates environment references,
  - current scene image is injected when generating shots,
  - start/end keyframes are constrained by character and scene assets.

### ai-shotlive

Repository: https://github.com/sorker/ai-shotlive

- One-stop AI short drama / manju platform with frontend/backend separation and user service.
- Important scene/data ideas:
  - `script_scenes`: scene location, time period, atmosphere, concept image.
  - `story_paragraphs`: story paragraphs linked to scenes.
  - `shot_keyframes`: start/end keyframes and prompts.
  - `shot_video_intervals`: generated video clips.
  - CutOS editor integration for multi-track editing.

### AIComicBuilder

Repository: https://github.com/twwch/AIComicBuilder

- Direct AI manju generator reference.
- Important ideas:
  - script import from TXT/DOCX/PDF,
  - character extraction and project-level character reuse,
  - character four-view references,
  - storyboard Kanban,
  - start/end keyframe generation per shot,
  - video prompt generation,
  - final video stitching and subtitle burn-in.

### AIYOU open AI video drama generator

Repository: https://github.com/yubowen123/AIYOU_open-ai-video-drama-generator

- Node-based AI short drama / manju production platform.
- Important ideas:
  - React Flow canvas for script, character, storyboard, and video nodes,
  - node output passes structured data to the next node,
  - character three-view and expression grid,
  - storyboard generation with shot size, camera angle, movement, duration,
  - multi-model/fallback orientation.

### Seedance2 Storyboard Generator

Repository: https://github.com/liangdabiao/Seedance2-Storyboard-Generator

- Story/novel to multi-episode video workflow.
- Important ideas:
  - C/S/P asset numbering for characters, scenes, props,
  - four-act script structure,
  - time-coded 0-3s / 3-6s / 6-9s prompt segments,
  - tail-frame descriptions for next-episode continuity,
  - provider constraints such as max reference images/videos and prompt-following limits.

### Openjourney

Repository: https://github.com/ammaarreshi/openjourney

- MidJourney-like web UI using Imagen/Veo.
- Important ideas:
  - candidate image/video grids,
  - one-click image-to-video,
  - filmstrip navigation,
  - polished loading/skeleton states.

## Open-source engineering references

### ComfyUI

Repository: https://github.com/Comfy-Org/ComfyUI

- Described as a modular diffusion model GUI, API, and backend with a graph/nodes interface.
- Best reference for:
  - Node-based AI generation workflow.
  - Visual graph execution.
  - Reusable model and prompt nodes.
  - Workflow serialization.
- Manmu implication:
  - Do not hardcode a single generation pipeline.
  - Represent generation as workflow steps/nodes internally, even if MVP UI starts with a simple wizard.

### ComfyUI Manager / workflow ecosystem

Repositories:

- https://github.com/Comfy-Org/ComfyUI-Manager
- https://github.com/ZHO-ZHO-ZHO/ComfyUI-Workflows-ZHO
- https://github.com/kijai/ComfyUI-WanVideoWrapper
- https://github.com/kijai/ComfyUI-HunyuanVideoWrapper
- https://github.com/kijai/ComfyUI-CogVideoXWrapper

Useful lessons:

- Ecosystem value comes from installable nodes and reusable workflows.
- Video generation will likely need wrapper nodes around different models.
- Manmu should save each generation recipe as a versioned workflow/preset.

### CogVideo / CogVideoX

Repository: https://github.com/zai-org/CogVideo

- Text-to-video and image-to-video generation reference.
- Useful for understanding open video model serving requirements and prompt controls.
- Important provider-capability ideas:
  - T2V,
  - I2V,
  - video continuation,
  - prompt optimization/extension as a separate model step.

### Wan2.1

Repository: https://github.com/Wan-Video/Wan2.1

- Important video generation model in the Chinese/open ecosystem.
- Useful for evaluating self-hosted GPU worker feasibility.
- Important provider-capability ideas:
  - T2V,
  - I2V,
  - first-last-frame-to-video,
  - video editing,
  - video-to-audio,
  - Chinese prompt preference for some tasks,
  - consumer-GPU 1.3B option for future self-hosting.

### AnimateDiff

Repository: https://github.com/guoyww/AnimateDiff

- Plug-and-play motion module that turns personalized T2I diffusion models into animation generators.
- Important ideas:
  - MotionLoRA for camera moves,
  - sparse RGB/sketch controls,
  - future self-hosted animation worker option.

### MuseTalk / Wav2Lip

Repositories:

- https://github.com/TMElyralab/MuseTalk
- https://github.com/Rudrabha/Wav2Lip

- Useful for optional lip-sync and dubbing.
- MuseTalk supports multilingual lip sync and real-time inference in suitable GPU environments.
- Wav2Lip open-source model is research/non-commercial; use as conceptual reference or external provider adapter only after license review.

### HunyuanVideo

Repository: https://github.com/Tencent-Hunyuan/HunyuanVideo

- Large video generation framework from Tencent Hunyuan.
- Useful for Chinese model ecosystem and self-hosted inference reference.

### Dify

Repository: https://github.com/langgenius/dify

- Open-source LLM app development platform.
- Key features include workflow, broad model support, Prompt IDE, RAG, agents, LLMOps, and APIs.
- Manmu implication:
  - Prompt IDE, model provider management, workflow logs, and observability are core product features, not admin afterthoughts.

### Flowise

Repository: https://github.com/FlowiseAI/Flowise

- Visual AI agent/workflow builder with React UI, server, and components modules.
- Useful for visual workflow UX and node-based agent orchestration.

### Ollama

Repository: https://github.com/ollama/ollama

- Runs open models locally and exposes REST API.
- Useful for local/self-hosted LLM provider support.

### LangChainGo

Repository: https://github.com/tmc/langchaingo

- Go implementation of LangChain.
- Useful if the Go backend directly orchestrates some LLM calls.
- Caution: for complex multimodal workflows, a provider adapter layer may be simpler than adopting a full chain framework too early.

### LiteLLM / AI gateway pattern

Repository: https://github.com/BerriAI/litellm

- Unified API gateway for many LLM providers.
- Manmu implication:
  - Consider an internal `model_gateway` abstraction even if LiteLLM itself is not used.
  - Track provider, model, cost, latency, fallback, and errors uniformly.

### Langfuse / LLM observability pattern

Repository: https://github.com/langfuse/langfuse

- Reference for prompt tracing, evaluation, datasets, and LLM observability.
- Manmu implication:
  - Every generation should be reproducible from prompt + model + params + assets.
  - Store traces for debugging and quality improvement.

## Resolved direction

- Model strategy: external API first.
- Reason: fastest product validation, API-based cost accounting, no early GPU operations.
- Architecture implication: keep a provider adapter interface so external APIs can later be replaced or supplemented by ComfyUI / CogVideo / Wan / HunyuanVideo GPU workers.

## Recommended MVP

Build an AI Studio MVP with these core flows:

1. Project creation.
2. Idea / novel input.
3. LLM generates:
   - story summary,
   - story analysis,
   - world bible,
   - character cards,
   - scene map,
   - prop map,
   - episode outline,
   - shot list,
   - image/video prompts.
4. Image model generates enhanced character reference sheets:
   - full-body reference,
   - turnaround / three-view sheet,
   - expression pack,
   - common pose pack.
5. Image model generates scene concept images and prop reference images.
6. User locks accepted character, scene, and prop versions.
7. User edits storyboard cards.
8. System generates keyframes or short clips asynchronously using locked character, scene, and prop references.
9. User edits clips in the timeline with TTS, subtitles, and transitions.
10. Asset library records all inputs/outputs.
11. Export MP4 inside the site.

## System design implications

### Key domains

- Project
- StoryBible
- Character
- CharacterReference
- CharacterVersion
- SceneMap
- Scene
- Location
- SceneConcept
- Prop
- PropReference
- Episode
- Scene
- Shot
- Prompt
- ModelProvider
- GenerationJob
- Asset
- Export
- ReviewNote
- Timeline
- TimelineTrack
- TimelineClip

### Job state machine

```text
draft -> queued -> running -> succeeded
                       ├-> failed -> retrying -> queued
                       └-> cancelled
```

### Provider abstraction

```text
LLMProvider
ImageProvider
VideoProvider
AudioProvider
CompositionProvider
```

Each provider should normalize:

- model name
- input assets
- prompt
- params
- output assets
- cost
- duration
- raw request/response metadata
- error code/message

## Risks

- Model API availability and pricing can change quickly.
- Video generation latency is high, so synchronous request/response UX will fail.
- Character consistency is hard; needs reference images, LoRA/IPAdapter/consistent seed strategies later.
- Prompt and generated media may create copyright/safety risk; content moderation must be planned early.
- Self-hosted video models require GPU capacity planning and isolation.

## Recommended next decisions

1. Target creator: solo creator, studio team, or internal content ops.
2. First UI mode: wizard, timeline/storyboard, or node workflow.
3. Provider priority list for LLM, image, video, TTS, and export.
