# GitHub Research: AI Manju / AI Video Drama Projects

## Purpose

Continue mapping GitHub projects that are closer to Manmu's target product: AI manju / animated comic drama generation, story-to-video workflow, character consistency, keyframe-to-video, dubbing, and editing.

## Best direct product references

### AIComicBuilder

Repository: https://github.com/twwch/AIComicBuilder

Positioning: AI-powered manju generator from script to animated video.

Key capabilities:

- Script import from TXT/DOCX/PDF.
- AI parses text, extracts characters, and splits episodes.
- Project-level episode management.
- Project-level character management with cross-episode reuse.
- Character four-view references: front, three-quarter, side, back.
- AI storyboard generation with composition, lighting, and camera movement.
- Start/end keyframe generation per shot.
- Video prompt generation based on storyboard and reference frames.
- Video generation from start/end frames.
- Final video stitching and subtitle burn-in using FFmpeg.
- Storyboard drawer, inline character panel, and Kanban view.
- Provider support: OpenAI, Gemini, Kling, Seedance, Veo.

Stack:

- Next.js 16, React 19, Tailwind CSS 4, Zustand, Base UI.
- SQLite + Drizzle ORM.
- AI SDK for text providers.
- FFmpeg / fluent-ffmpeg for final composition.

Manmu implications:

- Very strong MVP reference.
- Keep generation pipeline stage-based and manually triggerable.
- Use storyboard Kanban to show shot progress.
- Include start/end frame as first-class shot assets.
- Keep model provider configurable per project.
- Add final video + all assets download/export.

Risks:

- Apache 2.0 for repo, but verify provider terms and included assets before code reuse.
- Project is Next/Node/SQLite while Manmu target is Go + React + PostgreSQL.

### AIYOU open AI video drama generator

Repository: https://github.com/yubowen123/AIYOU_open-ai-video-drama-generator

Positioning: AI short drama / manju production platform with node-based workflow.

Key capabilities:

- Node-based creation canvas.
- 12 intelligent nodes covering script, character, storyboard, and video.
- Script outline generation for 5-50 episodes.
- Episode script generation with dialogue and scene description.
- Character design with three-view and expression grid.
- Storyboard generation with shot size, angle, camera movement, and duration.
- Storyboard image generation with multi-panel layouts.
- Video prompt construction for Sora / other models.
- Node connections pass script, character, and storyboard data between steps.
- React Flow-based editor.
- Multi-model support: Sora, Runway, Veo, Luma, MiniMax, Gemini, DeepSeek.

Stack:

- React 19, TypeScript, Vite, Zustand, React Flow, Tailwind.
- Express backend.
- Tencent COS for file storage.

Manmu implications:

- Canvas / node editor is a strong future UI direction.
- MVP can start with fixed SOP, then expose the workflow as a node canvas later.
- Inspected source suggests the current AIYOU canvas is custom rather than React Flow-based: `AppNode` stores x/y/data, `ConnectionLayer` draws SVG Bezier edges, Zustand stores editor state, and `nodeValidation` defines typed dependency rules.
- React Flow may still be suitable for Manmu, but the more important lesson is renderer-independent workflow graph modeling.
- "Node output feeds next node" maps well to Manmu AgentArtifact.

Risks:

- README states product is not production-ready and has bugs.
- Good for product workflow and prompt structure reference, not direct backend architecture.
- Detailed deep dive: `research/aiyou-workflow-canvas-deep-dive.md`.

### Seedance2 Storyboard Generator

Repository: https://github.com/liangdabiao/Seedance2-Storyboard-Generator

Positioning: Claude Code skill + Seedance 2.0 workflow that turns stories/novels into multi-episode videos.

Key workflow:

```text
theme -> script -> asset descriptions -> image generation -> storyboard script -> episode video generation
```

Important details:

- Four-act script structure.
- Numbered asset prompts:
  - Cxx for characters.
  - Sxx for scenes.
  - Pxx for props.
- Seedance timeline prompt format, e.g. 0-3s, 3-6s, 6-9s, 9-12s, 12-15s.
- Last-frame description is stored for the next episode.
- Video extension uses previous episode as input to create smoother continuity.
- Unified style prefix for all assets.
- Explicit constraints:
  - max 9 reference images per generation,
  - max 3 videos / 15 seconds as reference,
  - long prompts may be followed inconsistently,
  - sensitive words may cause generation failure.

Manmu implications:

- Add asset numbering conventions internally even if UI shows friendly names.
- Store per-shot time-sliced prompt segments.
- Store tail-frame descriptions and optional tail-frame image for continuity.
- Add provider capability metadata: max reference images, max reference videos, max duration, prompt length risk.
- Prompt Engineer Agent should shorten / segment long prompts.

### Openjourney

Repository: https://github.com/ammaarreshi/openjourney

Positioning: MidJourney-like web UI clone with real Imagen/Veo generation.

Key capabilities:

- Prompt-first image/video generation.
- 4-image grid for image candidates.
- 2x2 video grid with hover preview.
- One-click image-to-video conversion.
- Loading skeletons, fullscreen lightbox, film-strip navigation.
- Next.js, TypeScript, Tailwind, Framer Motion, shadcn/ui, Radix.

Manmu implications:

- Use candidate grid for character, scene, prop, and keyframe selection.
- Let users pick one candidate to lock as the reference version.
- Use filmstrip navigation for generated shot variants.

## Manga / comic consistency references

### StoryDiffusion

Repository: https://github.com/HVision-NKU/StoryDiffusion

Key idea:

- Consistent self-attention for long-range image generation.
- Can generate consistent comic sequences.
- Uses condition images to generate longer videos in a two-stage approach.

Manmu implications:

- Treat long story consistency as a first-class problem.
- Generate and lock reference images before producing many shots.
- For later self-hosting, consider consistent-attention or equivalent methods for character stability.

### DiffSensei

Repository: https://github.com/jianzongwu/DiffSensei

Key idea:

- Bridges MLLM and diffusion for customized manga generation.
- Generates controllable black-and-white manga panels.
- Supports flexible character adaptation from one input character image.
- Supports varied-resolution manga panel generation.

Manmu implications:

- Future manga panel mode can combine MLLM layout reasoning with diffusion rendering.
- Useful for black-and-white manga / comic-page output in addition to video.

### MangaNinja

Repository: https://github.com/ali-vilab/MangaNinjia

Key idea:

- Reference-based line art colorization.
- Aligns reference image to line art for consistent colorization.
- Supports point control.

Manmu implications:

- If Manmu later supports line-art storyboard first, MangaNinja-like colorization can keep style/color consistent.
- Current license is CC BY-NC 4.0, so avoid commercial code/model reuse without permission.

## Video generation / animation references

### Wan2.1

Repository: https://github.com/Wan-Video/Wan2.1

Key capabilities:

- Text-to-video, image-to-video, video editing, text-to-image, video-to-audio.
- Consumer-grade option: T2V-1.3B can run on about 8GB VRAM for 480P.
- First-last-frame-to-video model.
- VACE all-in-one video creation/editing model.
- Chinese/English visual text generation.
- Important ecosystem: Wan video wrappers, Wan-Move, EchoShot, AniCrafter, TeaCache, DiffSynth-Studio.

Manmu implications:

- First-last-frame-to-video fits Manmu shot model very well.
- Keep provider capability fields for FLF2V support.
- External API first, but Wan is a strong later self-hosted worker candidate.

### CogVideo / CogVideoX

Repository: https://github.com/zai-org/CogVideo

Key capabilities:

- Text-to-video and image-to-video.
- CogVideoX1.5 supports higher-resolution 10-second videos.
- CogVideoX-5B-I2V supports background image input + prompt.
- Prompt optimization with large models is recommended.
- Diffusers and quantized inference support.

Manmu implications:

- Provider adapter should support T2V, I2V, video continuation, and prompt extension.
- Prompt optimization should be a separate recorded step, not hidden inside generation.

### AnimateDiff

Repository: https://github.com/guoyww/AnimateDiff

Key capabilities:

- Plug-and-play motion module that turns personalized text-to-image models into animation generators.
- MotionLoRA supports camera motions such as zoom in/out, pan, tilt, and rolling.
- SparseCtrl adds control using sparse RGB/sketch inputs.

Manmu implications:

- Cinematographer Agent can map camera movement terms to provider-specific motion controls.
- Future self-hosted animation mode can use sketch/keyframe controls.

## Voice / lip sync references

### MuseTalk

Repository: https://github.com/TMElyralab/MuseTalk

Key capabilities:

- Real-time high-quality lip-syncing.
- Supports Chinese, English, Japanese audio.
- Real-time inference 30fps+ on NVIDIA V100.
- Can tune face-region center point; web UI allows first-frame parameter adjustment.

Manmu implications:

- Voice & Subtitle Agent should output language, speaker, and timing metadata.
- Lip sync should be optional by shot/character, because many manju shots may use narration rather than talking faces.

### Wav2Lip

Repository: https://github.com/Rudrabha/Wav2Lip

Key capabilities:

- Accurate lip-sync for videos in the wild.
- Works with any identity, voice, language, CGI faces, and synthetic voices.
- Open-source model is non-commercial / research-only.

Manmu implications:

- Do not use Wav2Lip open-source model commercially without checking license.
- Product architecture should support external lip-sync APIs as provider adapters.

## Updated Manmu MVP recommendations

### Workflow

```text
import story / idea
  -> story analysis
  -> episode split
  -> C/S/P asset numbering
  -> character reference candidates
  -> scene concept candidates
  -> prop reference candidates
  -> user locks references
  -> storyboard / shot cards
  -> start frame and optional end frame
  -> provider-specific video prompts
  -> shot video generation
  -> continuity review
  -> voice, subtitle, optional lip-sync
  -> timeline edit
  -> MP4 export + asset package
```

### Must-have artifacts

- Project
- Episode
- CharacterAsset with C-number
- SceneAsset with S-number
- PropAsset with P-number
- ShotCard
- ShotStartFrame
- ShotEndFrame
- VideoPrompt
- VideoClip
- SubtitleSegment
- Timeline

## Deep-dive implementation map

See `ai-manju-github-deep-dive-implementation-map.md` for the implementation-oriented mapping across AIComicBuilder, AIYOU, Seedance2 Storyboard Generator, Openjourney, StoryDiffusion, DiffSensei, Wan2.1, CogVideoX, AnimateDiff, MuseTalk, and Wav2Lip.

Key final stance:

- Use AIComicBuilder-like staged pipeline as the direct MVP backbone.
- Use AIYOU-like typed workflow graph internally, but keep the first UI as fixed SOP + Agent Board.
- Use Seedance2-like C/S/P numbering, time-coded prompt segments, tail-frame continuity, and provider constraint metadata.
- Use Openjourney-like candidate grids and filmstrip variant browsing.
- Keep Wan/CogVideo/AnimateDiff/MuseTalk/Wav2Lip as provider/model capability references, not required MVP dependencies.
- Do not use non-commercial models commercially without licensing.
- Export

### UI patterns to borrow

- Candidate grids from Openjourney.
- Storyboard Kanban from AIComicBuilder.
- Node canvas from AIYOU as future advanced mode.
- Timeline prompt slices from Seedance2 workflow.
- Agent Board from multi-agent planning.

### Backend rules

- Every generated asset must have lineage:
  - source artifact ids,
  - prompt template version,
  - model/provider,
  - params,
  - cost,
  - status/error,
  - selected/locked version.
- Provider capability metadata is required:
  - text-to-video,
  - image-to-video,
  - first-last-frame-to-video,
  - video continuation,
  - max reference images,
  - max reference videos,
  - max duration,
  - prompt language preference,
  - commercial/license constraints.
