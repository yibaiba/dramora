# AI Manju Project Feature Map

## Purpose

Map strong GitHub / public AI manju, manga, storyboard, and AI video editor projects into concrete product modules for Manmu.

## High-signal reference projects

| Project | What it is | Key ideas for Manmu | Notes / risk |
| --- | --- | --- | --- |
| BigBanana AI Director | AI one-stop short drama / manju director platform | Project -> season -> episode; worldbuilding resources; Script-to-Asset-to-Keyframe; character, scene, prop assets; scene continuity; CutOS-style delivery | CC BY-NC-SA 4.0 / commercial constraints; use as reference, not direct code foundation |
| CineGen AI Director | AI motion comic / animatic workbench | Script-to-Asset-to-Keyframe; character consistency; set design; director workbench; start/end keyframes; context-aware generation | IndexedDB/local-first reference; model-specific implementation |
| ai-shotlive | One-stop AI short drama/manju platform, frontend/backend/user service | Novel -> script -> storyboard -> asset -> keyframe -> video -> AI editing; separate script scenes, characters, variations, keyframes, video intervals, generation tasks | Good data model reference; license/non-commercial constraints must be checked |
| StoryDiffusion | Consistent long-range image/video generation | Consistent characters across long story sequences; prompt chain; multi-character sequence identity | Useful for concept; self-hosting needs GPU, but MVP is external API first |
| DiffSensei | Customized manga generation with MLLM + diffusion | Multi-character control, panel layout control, character adaptation to panel text, MangaZero panel dataset | Strong research reference for future scene/panel layout control |
| AI Comic Generator | Gemini-based text-to-comic tool | JSON-driven workflow, global style config, character workshop, storyboard editor, background tasks | Good structured workflow reference; FastAPI/Vue stack not adopted directly |
| Storyboard_Diffusion | Human-computer collaborative storyboard web tool | Natural language storyboard generation; Stable Diffusion web UI backend; GPT prompt edit services | Older stack, useful for collaborative storyboard concept |
| VNVE | Visual Novel Video Editor | Scene-based video model; title/dialogue scene templates; PixiJS + WebCodecs export; text-first creation | Very relevant to dialogue-heavy manju scenes |
| FreeCut | Browser-based multi-track video editor | Timeline, preview, WebCodecs/WebGPU export, TTS, subtitles, media library, feature-domain architecture | Excellent editor architecture reference; scope too large for v1 |
| Twick | React video editor SDK | Timeline/canvas/live-player/browser-render/render-server/cloud-transcript packages | Good modular editor SDK pattern |
| CutOS | AI-first video editor | Natural language editing, semantic video search, AI dubbing, AI morph transitions, multi-track timeline | Good future editor AI-agent reference |
| AIComicBuilder | AI-driven manju generator from script to animated video | Script import, character extraction, character four-view, storyboard Kanban, start/end keyframes, video prompts, FFmpeg stitching | Strongest direct MVP reference; stack differs from Manmu |
| AIYOU open AI video drama generator | Node-based AI short drama/manju production platform | React Flow canvas, 12 intelligent nodes, script/character/storyboard/video nodes, model fallback | Strong workflow/UI reference; README says not production-ready |
| Seedance2 Storyboard Generator | Story/novel to multi-episode video workflow | C/S/P asset numbering, 0-3s timeline prompts, tail-frame continuity, video extension | Excellent prompt/SOP reference; not a full web product |
| Openjourney | MidJourney-like Imagen/Veo web UI | Candidate grid, one-click image-to-video, filmstrip navigation, polished loading states | Good generation UX reference |
| MangaNinja | Reference-based manga line-art colorization | Reference following, point control, line-art to color workflow | Future manga panel/colorization; non-commercial license risk |
| Wan2.1 | Open video foundation model suite | T2V/I2V/FLF2V/video editing, Chinese ecosystem, consumer GPU option | Strong future self-hosted/video-worker candidate |
| CogVideoX | Open T2V/I2V/video continuation model | Prompt optimization, I2V background input, 10s video support, diffusers/quantization | Useful provider capability reference |
| AnimateDiff | T2I model animation module | MotionLoRA camera moves, sparse RGB/sketch controls | Future animation/self-hosting reference |
| MuseTalk / Wav2Lip | Lip sync / dubbing references | Multilingual lip sync, talking face generation, external API option | Optional voice/lip-sync pipeline; check license |

## Feature map for Manmu MVP

### 1. Project / world map

Reference: BigBanana

- Project, season, episode hierarchy.
- World bible:
  - genre,
  - target duration,
  - tone,
  - global visual style,
  - geography / map,
  - regions,
  - locations,
  - music and mood anchors.

### 2. Story analysis map

Reference: BigBanana, ai-shotlive, AI Comic Generator

- Input: idea, outline, novel chapter, or script.
- LLM outputs:
  - story summary,
  - themes,
  - conflict,
  - timeline,
  - emotional curve,
  - character relationships,
  - scene list,
  - prop list,
  - episode structure.

### 3. Character map

Reference: CineGen, BigBanana, ai-shotlive, StoryDiffusion, DiffSensei

- Character card:
  - name,
  - role,
  - age range,
  - personality,
  - motivation,
  - visual description,
  - wardrobe variants,
  - relationships.
- Reference outputs:
  - full body,
  - turnaround / three views,
  - expression pack,
  - common pose pack.
- Version locking:
  - lock accepted character version before storyboard/video generation.
  - shots reference locked character assets.

### 4. Scene map

Reference: CineGen set design, BigBanana scene assets, ai-shotlive `script_scenes`.

Scene map must be first-class, not just text inside a shot.

- Scene fields:
  - name,
  - location,
  - region / map node,
  - time of day,
  - weather,
  - season,
  - atmosphere,
  - lighting,
  - color palette,
  - camera mood,
  - recurring props,
  - scene concept prompt,
  - scene concept images,
  - continuity notes.
- Scene reference outputs:
  - wide establishing shot,
  - key background plate,
  - alternate angle references,
  - lighting/mood variants if needed.
- Usage:
  - every shot references one scene.
  - image/video prompt generation injects current scene concept image + locked characters + props.
  - scene continuity reduces "not the same place" failures.

### 5. Prop map

Reference: BigBanana, ai-shotlive

- Prop card:
  - name,
  - owner / related character,
  - visual description,
  - function in plot,
  - reference image,
  - continuity notes.
- Shots can reference props just like characters and scenes.

### 6. Shot / keyframe map

Reference: CineGen, BigBanana, ai-shotlive, AIComicBuilder, Seedance2 Storyboard Generator

- Each shot includes:
  - scene id,
  - participating character ids,
  - prop ids,
  - dialogue,
  - action,
  - camera movement,
  - start frame prompt,
  - optional end frame prompt,
  - duration,
  - video provider settings.
- Keyframe-driven generation:
  - generate start frame first.
  - optionally generate end frame.
  - video provider interpolates or animates from frames.
- Prompt timeline slices:
  - support 0-3s / 3-6s / 6-9s style beat prompts for providers that follow time-coded instructions.
  - store tail-frame description/image for next-shot or next-episode continuity.
- Provider capabilities:
  - T2V, I2V, first-last-frame-to-video, video continuation, max references, max duration, prompt language preference.

### 7. Timeline / editor map

Reference: VNVE, FreeCut, Twick, CutOS

- Timeline tracks:
  - video,
  - image/keyframe,
  - audio/TTS,
  - subtitles/text.
- Required operations:
  - trim,
  - split,
  - move,
  - delete,
  - duplicate,
  - reorder,
  - fade/crossfade/slide transition.
- Export:
  - frontend real-time preview.
  - server-side MP4 export.

### 8. Generation workspace map

Reference: AIComicBuilder, AIYOU, Openjourney

- Candidate grid:
  - show 4 image candidates or multiple video variants.
  - allow pick/lock/regenerate.
- Storyboard Kanban:
  - columns by shot state: planned, prompt ready, keyframe ready, video generating, review needed, approved.
- Node canvas:
  - advanced/future mode for exposing generation workflow.
  - MVP can keep fixed SOP while visualizing nodes in Agent Board.

## Recommended Manmu MVP map

```text
Project
  ├── WorldBible
  │   ├── Regions / Locations
  │   ├── VisualStyle
  │   └── MusicMood
  ├── Episode
  │   ├── StoryAnalysis
  │   ├── CharacterMap
  │   ├── SceneMap
  │   ├── PropMap
  │   ├── ShotList
  │   └── Timeline
  ├── AssetLibrary
  │   ├── CharacterRefs
  │   ├── SceneConcepts
  │   ├── PropRefs
  │   ├── Keyframes
  │   ├── VideoClips
  │   ├── Audio
  │   └── Subtitles
  └── GenerationJobs
```

## Additional research

See `research/github-ai-video-drama-projects.md` for detailed notes on AIComicBuilder, AIYOU, Seedance2 Storyboard Generator, Openjourney, StoryDiffusion, DiffSensei, MangaNinja, Wan2.1, CogVideoX, AnimateDiff, MuseTalk, and Wav2Lip.

## Architecture implications

- `Scene` and `Location` should be separate entities.
- `Shot` must reference `scene_id`, not duplicate scene text.
- Prompt generation must compose:
  - global style,
  - scene concept,
  - locked character references,
  - prop references,
  - shot action/camera/dialogue.
- Asset lineage must track whether a generated image/video came from:
  - story analysis,
  - character reference,
  - scene concept,
  - prop reference,
  - shot keyframe,
  - final video interval.
