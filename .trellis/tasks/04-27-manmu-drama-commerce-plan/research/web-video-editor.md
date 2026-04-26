# Research: Web Online Video Editor References

## MVP decision

The MVP should include a complete online editor:

- timeline editing
- voice/TTS
- subtitles
- transitions
- preview
- export inside the website

This shifts Manmu from a generation-only studio to an AI generation + online editing platform.

## Reference projects

### VNVE

Repository: https://github.com/vnve/vnve

- Visual Novel Video Editor.
- Browser-based tool for making visual novel videos.
- Pure frontend TypeScript implementation using PixiJS + WebCodecs.
- Supports AI-powered rapid creation through APIs such as DeepSeek and OpenAI.
- Provides scene, image, text, sound, animation, template, and MP4 generation concepts.
- Strong fit for Manmu because AI manju is closer to visual novel / dialogue scene generation than generic video editing.

Useful lessons:

- Treat the output video as a sequence of scenes.
- Templates are important: title scene, dialogue scene, narration scene, transition scene.
- Programmatic video generation APIs can coexist with a visual editor.

### FreeCut

Repository: https://github.com/walterlow/freecut

- Browser-based multi-track video editor.
- Uses React + TypeScript, Vite, WebGPU, WebCodecs, File System Access API, Zustand, TanStack Router, Tailwind, shadcn/ui, Mediabunny, Transformers.js, Kokoro.js.
- Supports video/audio/text/image/shape tracks, trimming, split/join, transitions, keyframes, preview, waveform, subtitles, TTS, and in-browser export.

Useful lessons:

- A full editor needs explicit domains: timeline, preview, player, composition-runtime, export, effects, keyframes, media-library.
- WebCodecs/WebGPU can provide powerful browser-side editing, but browser support is mainly Chromium.
- Manmu can reference its structure while keeping MVP simpler.

### Twick

Repository: https://github.com/ncounterspecialist/twick

- React video editor SDK with timeline editing, canvas tools, AI captions, and MP4 export.
- Packages include timeline, canvas, live-player, video-editor, studio, browser-render, render-server, cloud-transcript, cloud-caption-video, cloud-export-video.
- Supports browser WebCodecs or server-side FFmpeg rendering.

Useful lessons:

- A modular editor SDK is better than a single monolithic editor component.
- Manmu should split timeline model, canvas/preview, live player, export worker, and AI caption pipeline.
- Server-side export is useful for browser compatibility and long renders.

### DesignCombo React Video Editor

Repository: https://github.com/designcombo/react-video-editor

- React + TypeScript video editor application.
- Supports timeline editing, effects/transitions, multi-track editing, export options, and real-time preview.
- Good UI/interaction reference.
- License should be reviewed before direct reuse.

### Remotion

Repository: https://github.com/remotion-dev/remotion

- Framework for creating videos programmatically using React.
- Strong for React-driven composition, transitions, captions, and server rendering.
- License requires review for commercial/company usage.

Useful lessons:

- Programmatic rendering lets AI-generated structured data become video reliably.
- A visual timeline can serialize to JSON and render via React components.

### ffmpeg.wasm

Repository: https://github.com/ffmpegwasm/ffmpeg.wasm

- WebAssembly port of FFmpeg for browser-side audio/video processing.
- Good for trimming, conversion, lightweight local processing.
- Not ideal as the only export path for long or complex projects due performance and memory constraints.

### IMG.LY video-editor-wasm-react

Repository: https://github.com/imgly/video-editor-wasm-react

- Simple React + ffmpeg.wasm demo.
- Supports upload, trim, GIF conversion, and download.
- Useful for learning browser-side FFmpeg basics, not enough for Manmu full editor.

### Vanta

Repository: https://github.com/itsjwill/vanta

- Open-source AI video engine built on Remotion.
- References voice cloning, talking-head avatars, animated captions, video generation, timeline, transitions, motion graphics, and multiple open-source integrations.
- Very relevant as a feature map, but it is early-stage and should be treated as inspiration, not a foundation without code review.

## AI comic/manju generation references

### StoryDiffusion

Repository: https://github.com/HVision-NKU/StoryDiffusion

- Generates consistent comics/storyboards/images and supports image-to-video style long-range generation concepts.
- Key insight: character/style consistency across panels is a core product problem.

### AI Comic Generator

Repository: https://github.com/Dapeng960208/AI-Comic-Generator

- Converts text stories into illustrated comics using Gemini.
- Includes JSON-driven workflow, global style config, character workshop, storyboard editor, background tasks, and visual editor.
- Strong reference for Manmu's structured data model.

### AI Comic Factory / LlamaGen

Repository: https://github.com/LlamaGenAI/LlamaGen

- AI comic / anime generation project reference.
- Useful as a directional reference, but less mature than StoryDiffusion or AI Comic Generator based on current GitHub metadata.

## Recommended Manmu editor architecture

Use a split architecture:

```text
React Studio
  -> timeline JSON / project JSON
  -> preview player
  -> subtitle editor
  -> voice panel
  -> transition/effects panel
  -> asset library

Go API
  -> project/timeline persistence
  -> generation jobs
  -> model provider adapters
  -> export job orchestration

Render worker
  -> MVP option A: FFmpeg worker driven by timeline JSON
  -> MVP option B: Node/Remotion render worker if license permits
  -> Later: browser WebCodecs export for lighter projects
```

## Recommended MVP scope for "complete editor"

Complete editor does not need to mean Premiere-level editor in v1. It should mean the user can complete an end-to-end manju video inside Manmu.

Must-have:

- Multi-track timeline: video, image/keyframe, audio/TTS, subtitle/text.
- Basic edit tools: trim, split, move, delete, duplicate, reorder.
- Basic transitions: fade, crossfade/dissolve, slide.
- Subtitle editor: line timing, text edit, style preset.
- TTS panel: generate voice audio per line/scene.
- AI shot generation: create or regenerate a clip from a shot prompt.
- Preview player synchronized with timeline.
- Export job: render MP4 in site and allow download.
- Timeline JSON versioning.

Can defer:

- Advanced color grading.
- Full keyframe graph editor.
- WebGPU effects suite.
- Multi-user collaboration.
- Browser-only export for all formats.
- Full marketplace/community publishing.

## Technical risks

- Full editor scope can explode quickly; keep initial track types and effects narrow.
- Browser-side export is attractive but has compatibility/performance constraints.
- Server-side export requires robust job queue, progress reporting, temporary storage, and cleanup.
- Remotion and tldraw require license review before commercial reuse.
- AI-generated clips may have inconsistent characters; timeline editor must support easy shot replacement.

## Technical selection update

Detailed decision: see `timeline-editor-tech-selection.md`.

MVP choice:

- Build a narrow internal timeline editor instead of adopting a full third-party editor SDK first.
- Use server-side FFmpeg/export worker as the authoritative MP4 export path.
- Borrow FreeCut's feature-domain architecture, VNVE's visual-novel scene templates, and Twick's modular timeline/render split.
- Avoid direct Twick/DesignCombo/Remotion dependency until license and product coupling are reviewed.
- Keep WebCodecs/ffmpeg.wasm/browser export as phase-2 or lightweight utility path, not core MVP export.
